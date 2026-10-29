//go:build gui && windows

package desktop

import (
	"bytes"
	"io"
	"os"
	"strings"
	"syscall"
)

// powershellPath 是系统自带组件的固定位置（Windows PowerShell 5.1，
// Windows 10/11 必有）；os.StartProcess 不做 PATH 搜索，必须给全路径。
const powershellPath = `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`

// pickFolderScript 是编译期常量：整段脚本不含任何外部输入，
// 仅弹出 FolderBrowserDialog 并把所选路径写到标准输出。
const pickFolderScript = `Add-Type -AssemblyName System.Windows.Forms; ` +
	`$f = New-Object System.Windows.Forms.FolderBrowserDialog; ` +
	`$f.Description = '选择项目目录'; ` +
	`if ($f.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { [Console]::Out.Write($f.SelectedPath) }`

// pickFolderDialog 弹出系统文件夹选择对话框，返回所选绝对路径。
// 数组式启动（无 shell）；用户取消时返回 errCancelled。
func pickFolderDialog() (string, error) {
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		return "", err
	}
	attr := &os.ProcAttr{
		Files: []*os.File{os.Stdin, outWriter, outWriter},
		Sys:   &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000},
	}
	proc, err := os.StartProcess(powershellPath,
		[]string{powershellPath, "-NoProfile", "-STA", "-Command", pickFolderScript}, attr)
	outWriter.Close()
	if err != nil {
		outReader.Close()
		return "", err
	}
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, outReader)
		close(done)
	}()
	_, waitErr := proc.Wait()
	<-done
	outReader.Close()
	if waitErr != nil {
		return "", waitErr
	}
	path := strings.TrimSpace(buf.String())
	if path == "" {
		return "", errCancelled
	}
	return path, nil
}
