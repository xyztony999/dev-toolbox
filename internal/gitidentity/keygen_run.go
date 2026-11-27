package gitidentity

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// toolRunner starts one argv-array process (no shell). Tests may replace it.
type toolRunner func(ctx context.Context, cwd string, env []string, stdin string, name string, args ...string) (stdout, stderr string, err error)

var runTool toolRunner = defaultRunTool

func defaultRunTool(ctx context.Context, cwd string, env []string, stdin string, name string, args ...string) (string, string, error) {
	return runToolMode(ctx, cwd, env, stdin, false, name, args...)
}

// runDetachedTool runs a process with DevNull stdio so child daemons (gpg-agent)
// do not inherit pipes that close when the parent exits.
func runDetachedTool(ctx context.Context, cwd string, env []string, name string, args ...string) error {
	_, _, err := runToolMode(ctx, cwd, env, "", true, name, args...)
	return err
}

func runToolMode(ctx context.Context, cwd string, env []string, stdin string, detach bool, name string, args ...string) (string, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	exePath, err := resolveToolPath(name)
	if err != nil {
		return "", "", err
	}
	argv := append([]string{exePath}, args...)
	fullEnv := mergeToolEnv(env)

	var stdinFile *os.File
	if stdin != "" && !detach {
		reader, writer, pipeErr := os.Pipe()
		if pipeErr != nil {
			return "", "", pipeErr
		}
		stdinFile = reader
		go func() {
			_, _ = io.WriteString(writer, stdin)
			_ = writer.Close()
		}()
	} else {
		stdinFile, err = os.Open(os.DevNull)
		if err != nil {
			return "", "", err
		}
	}
	defer stdinFile.Close()

	var (
		stdoutReader, stdoutWriter *os.File
		stderrReader, stderrWriter *os.File
		outFile, errFile           *os.File
	)
	if detach {
		devNull, nullErr := os.OpenFile(os.DevNull, os.O_RDWR, 0)
		if nullErr != nil {
			return "", "", nullErr
		}
		defer devNull.Close()
		outFile, errFile = devNull, devNull
	} else {
		stdoutReader, stdoutWriter, err = os.Pipe()
		if err != nil {
			return "", "", err
		}
		stderrReader, stderrWriter, err = os.Pipe()
		if err != nil {
			stdoutReader.Close()
			stdoutWriter.Close()
			return "", "", err
		}
		outFile, errFile = stdoutWriter, stderrWriter
	}

	attr := &os.ProcAttr{
		Dir:   cwd,
		Env:   fullEnv,
		Files: []*os.File{stdinFile, outFile, errFile},
		Sys:   toolSysAttr(),
	}
	proc, err := os.StartProcess(exePath, argv, attr)
	if !detach {
		stdoutWriter.Close()
		stderrWriter.Close()
	}
	if err != nil {
		if !detach {
			stdoutReader.Close()
			stderrReader.Close()
		}
		return "", "", err
	}

	var outBuf, errBuf bytes.Buffer
	var wg sync.WaitGroup
	if !detach {
		wg.Add(2)
		go func() { defer wg.Done(); _, _ = io.Copy(&outBuf, stdoutReader) }()
		go func() { defer wg.Done(); _, _ = io.Copy(&errBuf, stderrReader) }()
	}

	waitDone := make(chan error, 1)
	go func() { _, waitErr := proc.Wait(); waitDone <- waitErr }()
	var waitErr error
	select {
	case waitErr = <-waitDone:
	case <-ctx.Done():
		_ = proc.Kill()
		<-waitDone
		if !detach {
			wg.Wait()
			stdoutReader.Close()
			stderrReader.Close()
		}
		return outBuf.String(), errBuf.String(), fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: 工具执行超时")
	}
	if !detach {
		wg.Wait()
		stdoutReader.Close()
		stderrReader.Close()
	}

	out, errOut := outBuf.String(), errBuf.String()
	if waitErr != nil {
		return out, errOut, fmt.Errorf("exit: %w", waitErr)
	}
	return out, errOut, nil
}

func resolveToolPath(name string) (string, error) {
	if strings.ContainsRune(name, '/') || strings.ContainsRune(name, '\\') || filepath.IsAbs(name) {
		if info, err := os.Stat(name); err == nil && !info.IsDir() {
			return name, nil
		}
		return "", fmt.Errorf("GIT_KEYGEN_TOOL_MISSING: 找不到可执行文件 %s", name)
	}
	path, err := findExecutable(name)
	if err != nil {
		return "", fmt.Errorf("GIT_KEYGEN_TOOL_MISSING: 未安装 %s，或不在 PATH 中", name)
	}
	return path, nil
}

func probeToolVersion(ctx context.Context, name string) (path string, version string, available bool) {
	path, err := resolveToolPath(name)
	if err != nil {
		return "", "", false
	}
	if _, statErr := os.Stat(path); statErr != nil {
		return path, "", false
	}
	// gpg understands --version. OpenSSH ssh-keygen has no safe version flag
	// (-v starts generation); leave version empty when --version is rejected.
	out, errOut, _ := runTool(ctx, "", nil, "", path, "--version")
	text := strings.TrimSpace(out)
	if text == "" {
		text = strings.TrimSpace(errOut)
	}
	line := firstLine(text)
	lower := strings.ToLower(line)
	if line != "" && !strings.Contains(lower, "unknown option") &&
		!strings.Contains(lower, "illegal option") && !strings.Contains(lower, "option requires") &&
		!strings.Contains(lower, "usage:") && !strings.Contains(lower, "generating public") {
		if len(line) > 120 {
			line = line[:120]
		}
		return path, line, true
	}
	return path, "", true
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func truncateToolErr(stderr string) string {
	s := strings.TrimSpace(stderr)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if len(s) > 400 {
		s = s[:400] + "…"
	}
	// Never echo lines that look like private key material.
	var kept []string
	for _, line := range strings.Split(s, "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "-----begin") || strings.Contains(lower, "private key") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func gpgEnv(home, bin string) []string {
	if home == "" {
		return nil
	}
	return []string{"GNUPGHOME=" + gpgToolHomePath(home, bin)}
}

// gpgToolHomePath adapts a Windows filesystem path for the gpg/gpgconf that
// will consume GNUPGHOME / --homedir. Git for Windows ships an MSYS build that
// treats "C:\..." as a relative path under the process cwd; rewrite those to
// "/c/..." while leaving native Gpg4win on forward-slash drive paths.
func gpgToolHomePath(home, bin string) string {
	if home == "" || runtime.GOOS != "windows" {
		return home
	}
	abs, err := filepath.Abs(home)
	if err != nil {
		abs = home
	}
	slash := filepath.ToSlash(abs)
	exe := bin
	if exe == "" {
		exe = "gpg"
	}
	if path, err := resolveToolPath(exe); err == nil {
		exe = path
	}
	if looksLikeMSYSBinary(exe) {
		return windowsPathToMSYS(slash)
	}
	return slash
}

func looksLikeMSYSBinary(path string) bool {
	p := strings.ToLower(filepath.ToSlash(path))
	return strings.Contains(p, "/git/") ||
		strings.Contains(p, "/msys") ||
		strings.Contains(p, "/usr/bin/gpg") ||
		strings.Contains(p, "/usr/bin/gpgconf")
}

func windowsPathToMSYS(p string) string {
	// Do not use filepath.ToSlash: on non-Windows builders (and in unit tests
	// that feed Windows-style strings) backslash is not a separator.
	p = strings.ReplaceAll(p, `\`, `/`)
	if len(p) >= 2 && p[1] == ':' {
		return "/" + strings.ToLower(string(p[0])) + p[2:]
	}
	return p
}

// mergeToolEnv copies the process environment, drops keys that extras will
// set, then appends extras. On Windows the first duplicate key wins, so we
// must remove before adding (especially GNUPGHOME).
func mergeToolEnv(extras []string) []string {
	drop := map[string]bool{}
	for _, e := range extras {
		if i := strings.IndexByte(e, '='); i > 0 {
			drop[strings.ToUpper(e[:i])] = true
		}
	}
	base := os.Environ()
	out := make([]string, 0, len(base)+len(extras))
	for _, e := range base {
		i := strings.IndexByte(e, '=')
		if i <= 0 {
			continue
		}
		if drop[strings.ToUpper(e[:i])] {
			continue
		}
		out = append(out, e)
	}
	return append(out, extras...)
}

// prepareGPGHome ensures a usable GNUPGHOME for batch/loopback generation.
func prepareGPGHome(home string) error {
	if err := os.MkdirAll(home, 0o700); err != nil {
		return err
	}
	agentConf := filepath.Join(home, "gpg-agent.conf")
	content := "allow-loopback-pinentry\ndisable-scdaemon\n"
	if _, err := os.Stat(agentConf); errors.Is(err, os.ErrNotExist) {
		if writeErr := os.WriteFile(agentConf, []byte(content), 0o600); writeErr != nil {
			return writeErr
		}
	}
	return nil
}

// ensureGPGAgent launches gpg-agent for the given homedir when possible.
// On Windows, quick-generate-key often fails with "No agent running" if the
// agent was never started for that GNUPGHOME.
func ensureGPGAgent(ctx context.Context, tools KeygenTools, home string) error {
	gpgconf := siblingOrPath(tools.gpgBin(), "gpgconf")
	if _, err := resolveToolPath(gpgconf); err != nil {
		// Best-effort: some installs only ship gpg.exe.
		return nil
	}
	args := []string{}
	if home != "" {
		args = append(args, "--homedir", gpgToolHomePath(home, gpgconf))
	}
	args = append(args, "--launch", "gpg-agent")
	if runErr := runDetachedTool(ctx, "", gpgEnv(home, gpgconf), gpgconf, args...); runErr != nil {
		return fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: 无法启动 gpg-agent")
	}
	return nil
}

// siblingOrPath returns name.exe next to bin when present, else name for PATH lookup.
func siblingOrPath(bin, name string) string {
	if bin == "" {
		return name
	}
	path, err := resolveToolPath(bin)
	if err != nil {
		return name
	}
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	sibling := filepath.Join(filepath.Dir(path), name+ext)
	if info, err := os.Stat(sibling); err == nil && !info.IsDir() {
		return sibling
	}
	return name
}

func defaultSSHDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh"), nil
}

func defaultGPGHome() string {
	if env := os.Getenv("GNUPGHOME"); env != "" {
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if runtime.GOOS == "windows" {
		// GnuPG on Windows commonly uses %APPDATA%\gnupg.
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "gnupg")
		}
	}
	return filepath.Join(home, ".gnupg")
}
