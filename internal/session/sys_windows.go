//go:build windows

package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

func getEnv(key string) string { return os.Getenv(key) }

// createNewConsole 令 Windows GUI 子系统进程（桌面客户端）也能开出
// 独立控制台窗口；父进程侧不对外暴露终端窗口。
const createNewConsole = 0x00000010

// createUnicodeEnvironment：环境块以 UTF-16 传递。
const createUnicodeEnvironment = 0x00000400

// createSuspended：挂起启动——CREATE_NEW_CONSOLE 的 conhost 初始化与
// 子进程主线程并发赛跑。挂起创建 + Resume 前宽限属于稳健性保险
// （真正根治闪退的是显式 stdio，见 startProcessSession 注释）。
const createSuspended = 0x00000004

// consoleInitGrace 是 ResumeThread 前给 conhost 的初始化宽限。
const consoleInitGrace = 300 * time.Millisecond

// STARTF_USESTDHANDLES：StartupInfo 携带 std 三句柄。
const startfUseStdHandles = 0x00000100

// 命名管道与句柄复制的 Win32 常量（仅取所需子集）。
const (
	pipeAccessInbound         = 0x00000001
	fileFlagFirstPipeInstance = 0x00080000
	duplicateSameAccess       = 0x00000002
)

// securityAttributes：可继承句柄所需（bInheritHandle=1）。
type securityAttributes struct {
	Length             uint32
	SecurityDescriptor uintptr
	InheritHandle      uint32
}

var (
	procResumeThread      = modkernel32.NewProc("ResumeThread")
	procCreateNamedPipeW  = modkernel32.NewProc("CreateNamedPipeW")
	procCreateFileW       = modkernel32.NewProc("CreateFileW")
	procDuplicateHandle   = modkernel32.NewProc("DuplicateHandle")
	procGetCurrentProcess = modkernel32.NewProc("GetCurrentProcess")
)

var (
	modkernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateProcessW = modkernel32.NewProc("CreateProcessW")
	procCloseHandle    = modkernel32.NewProc("CloseHandle")
)

// launchFor 直接以目标终端的可执行文件启动新控制台窗口。
// 历史 conhost.exe 封装链路在 GUI 子系统实测派不出目标进程（用户
// 反馈"打不开终端"），从既有控制台运行时又会劫持当前控制台；
// CREATE_NEW_CONSOLE 语义直白，窗口即产品意图。
// 返回 (exe 全路径, argv)。
func launchFor(terminal TerminalInfo, cwd string) (string, []string) {
	switch terminal.ID {
	case "powershell":
		return terminal.ExePath, []string{"powershell", "-NoLogo", "-NoExit"}
	case "cmd":
		return terminal.ExePath, []string{"cmd.exe", "/K"}
	case "bash":
		return terminal.ExePath, []string{filepath.Base(terminal.ExePath), "--login", "-i"}
	default:
		return terminal.ExePath, []string{terminal.ID}
	}
}

// sessionSysProcAttr：新控制台标志；无需 HideWindow（窗口即产品意图）。
func sessionSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNewConsole}
}

// startProcessSession 是 Windows 的进程启动原语（startSession 的平台实现）。
//
// 不用 os.StartProcess：它在 Windows 强制显式 stdio（Files 为 nil 直接
// EINVAL），而指向 NUL 设备的 stdin 会让交互 shell 读到 EOF 立即退出，
// -NoExit//K 也挡不住（2026-11-23 二分实验实证，即用户报告的"终端能
// 呼叫但立马闪退"）；继承父进程真实句柄则把子进程附着到父控制台，
// GUI 子系统下等于没有可见终端。
//
// 最终方案（10/10 连测存活）：显式 STARTF_USESTDHANDLES 传递 stdio——
// stdin 是命名管道读端（写端由父子双侧持有，永不 EOF）；stdout/stderr
// 指向 NUL。不传 stdio 时子进程依赖 conhost attach 时序随机获得有效
// stdio，一旦 .NET 运行时先于 attach 读 stdin 即得到 EOF 正常退出
// （exit 0，实测约 1/4~1/2 概率的"偶发闪退"，挂起+宽限治不了）；
// 显式有效 stdio 把存活从概率变成确定。键盘交互不受影响——交互式
// shell 的按键读取走 conhost 输入缓冲，不经该管道。
func startProcessSession(exePath string, argv []string, attr *os.ProcAttr) (int, error) {
	stdinRead, stdinWrite, err := createNeverClosingStdinPair()
	if err != nil {
		return 0, err
	}
	defer func() { _, _, _ = procCloseHandle.Call(stdinRead) }()
	defer func() { _, _, _ = procCloseHandle.Call(stdinWrite) }()

	nulHandle, err := openNulHandle()
	if err != nil {
		return 0, err
	}
	defer func() { _, _, _ = procCloseHandle.Call(nulHandle) }()

	flags := uint32(createNewConsole)
	if attr.Sys != nil {
		flags = attr.Sys.CreationFlags
	}
	flags |= createUnicodeEnvironment | createSuspended
	cmdline, err := syscall.UTF16PtrFromString(joinArgs(argv))
	if err != nil {
		return 0, err
	}
	exePtr, err := syscall.UTF16PtrFromString(exePath)
	if err != nil {
		return 0, err
	}
	var dirPtr *uint16
	if attr.Dir != "" {
		if dirPtr, err = syscall.UTF16PtrFromString(attr.Dir); err != nil {
			return 0, err
		}
	}
	var envPtr *uint16
	if len(attr.Env) > 0 {
		envPtr = utf16EnvBlock(attr.Env)
	}
	var si startupInfoWithStd
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = startfUseStdHandles
	si.StdInput = syscall.Handle(stdinRead)
	si.StdOutput = syscall.Handle(nulHandle)
	si.StdError = syscall.Handle(nulHandle)
	var pi syscall.ProcessInformation
	r1, _, callErr := procCreateProcessW.Call(
		uintptr(unsafe.Pointer(exePtr)),
		uintptr(unsafe.Pointer(cmdline)),
		0, 0,
		1, // bInheritHandles=TRUE：std 句柄经 StartupInfo 传递
		uintptr(flags),
		uintptr(unsafe.Pointer(envPtr)),
		uintptr(unsafe.Pointer(dirPtr)),
		uintptr(unsafe.Pointer(&si)),
		uintptr(unsafe.Pointer(&pi)),
	)
	if r1 == 0 {
		return 0, callErr
	}
	// 管道写端复制一份给子进程自持：任一进程退出（包括本应用关闭）
	// 都不会让终端的 stdin 变 EOF——终端生命周期独立于本应用。
	var childWrite uintptr
	currentProcess, _, _ := procGetCurrentProcess.Call()
	_, _, _ = procDuplicateHandle.Call(
		currentProcess, stdinWrite,
		uintptr(pi.Process), uintptr(unsafe.Pointer(&childWrite)),
		0, 0, duplicateSameAccess,
	)
	// conhost 宽限后恢复主线程；Resume 必须先于句柄关闭（Thread 句柄
	// 在此之前保持有效）。
	time.Sleep(consoleInitGrace)
	_, _, _ = procResumeThread.Call(uintptr(pi.Thread))
	// 句柄即刻关闭：Windows 无 Unix 式僵尸回收问题。
	_, _, _ = procCloseHandle.Call(uintptr(pi.Process))
	_, _, _ = procCloseHandle.Call(uintptr(pi.Thread))
	return int(pi.ProcessId), nil
}

// startupInfoWithStd 在 syscall.StartupInfo 之后按 STARTUPINFOW 布局
// 补上三个 std 句柄（STARTF_USESTDHANDLES 要求）。
type startupInfoWithStd struct {
	syscall.StartupInfo
	StdInput  syscall.Handle
	StdOutput syscall.Handle
	StdError  syscall.Handle
}

// createNeverClosingStdinPair 建一条字节型匿名语义的命名管道：读端
// 给子进程当 stdin，写端父自持 + DuplicateHandle 给子，双侧持有期间
// 读端永不 EOF。
func createNeverClosingStdinPair() (uintptr, uintptr, error) {
	sa := &securityAttributes{Length: uint32(unsafe.Sizeof(securityAttributes{})), InheritHandle: 1}
	name, err := syscall.UTF16PtrFromString(fmt.Sprintf(`\\.\pipe\devtoolbox-session-stdin-%d-%d`, os.Getpid(), time.Now().UnixNano()))
	if err != nil {
		return 0, 0, err
	}
	read, _, callErr := procCreateNamedPipeW.Call(
		uintptr(unsafe.Pointer(name)),
		pipeAccessInbound|fileFlagFirstPipeInstance,
		0, // PIPE_TYPE_BYTE | PIPE_WAIT
		1, 4096, 4096, 0,
		uintptr(unsafe.Pointer(sa)),
	)
	if read == uintptr(syscall.InvalidHandle) || read == 0 {
		return 0, 0, fmt.Errorf("CreateNamedPipeW: %v", callErr)
	}
	genericWrite := uint32(0x40000000)
	openExisting := uint32(3)
	write, _, callErr := procCreateFileW.Call(
		uintptr(unsafe.Pointer(name)),
		uintptr(genericWrite), 0,
		uintptr(unsafe.Pointer(sa)),
		uintptr(openExisting), 0, 0,
	)
	if write == uintptr(syscall.InvalidHandle) || write == 0 {
		_, _, _ = procCloseHandle.Call(read)
		return 0, 0, fmt.Errorf("CreateFileW(pipe write): %v", callErr)
	}
	return read, write, nil
}

// openNulHandle 打开可继承的 NUL 设备句柄（stdout/stderr 丢弃用）。
func openNulHandle() (uintptr, error) {
	sa := &securityAttributes{Length: uint32(unsafe.Sizeof(securityAttributes{})), InheritHandle: 1}
	name, err := syscall.UTF16PtrFromString("NUL")
	if err != nil {
		return 0, err
	}
	genericReadWrite := uint32(0xC0000000)
	fileShareReadWrite := uint32(0x3)
	openExisting := uint32(3)
	h, _, callErr := procCreateFileW.Call(
		uintptr(unsafe.Pointer(name)),
		uintptr(genericReadWrite), uintptr(fileShareReadWrite),
		uintptr(unsafe.Pointer(sa)),
		uintptr(openExisting), 0, 0,
	)
	if h == uintptr(syscall.InvalidHandle) || h == 0 {
		return 0, fmt.Errorf("CreateFileW(NUL): %v", callErr)
	}
	return h, nil
}

// joinArgs 以 EscapeArg 合成命令行；lpApplicationName 已精确指定应用，
// argv[0] 按惯例参与转义供 CRT 解析。
func joinArgs(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		parts[i] = syscall.EscapeArg(a)
	}
	return strings.Join(parts, " ")
}

// utf16EnvBlock 组装 CreateProcessW 的环境块：KEY=VAL 以 NUL 相接、
// 双 NUL 结尾；按变量名排序贴近 syscall.StartProcess 的行为。
func utf16EnvBlock(env []string) *uint16 {
	sorted := append([]string(nil), env...)
	sort.Slice(sorted, func(i, j int) bool {
		ni, _ := partitionEnv(sorted[i])
		nj, _ := partitionEnv(sorted[j])
		return ni < nj
	})
	var block []uint16
	for _, entry := range sorted {
		block = append(block, utf16.Encode([]rune(entry))...)
		block = append(block, 0)
	}
	block = append(block, 0)
	return &block[0]
}

func partitionEnv(entry string) (string, bool) {
	if i := strings.IndexByte(entry, '='); i >= 0 {
		return entry[:i], true
	}
	return entry, false
}
