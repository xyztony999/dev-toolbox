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

// gitBinary is fixed; every invocation is an argument-array process start
// (never a shell string). gitArgs rejects control characters and runGit
// only allows the fixed subcommand set below — user input never becomes a
// command line (GC-03). Process creation uses os.StartProcess directly with
// an explicit argv; no command interpreter is involved on any platform.
const gitBinary = "git"

var allowedGitSubcommands = map[string]bool{
	"config": true, "rev-parse": true, "log": true, "credential": true,
	"--version": true,
}

func gitArgs(args ...string) ([]string, error) {
	if len(args) == 0 {
		return nil, errors.New("GIT_ARGUMENT_INVALID: empty git invocation")
	}
	if !allowedGitSubcommands[args[0]] {
		return nil, fmt.Errorf("GIT_ARGUMENT_INVALID: git subcommand %q is not allowlisted", args[0])
	}
	cleaned := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.ContainsAny(arg, "\x00\r\n") {
			return nil, fmt.Errorf("GIT_ARGUMENT_INVALID: rejected git argument %q", arg)
		}
		cleaned = append(cleaned, arg)
	}
	return cleaned, nil
}

// findExecutable resolves a bare binary name against PATH. On Windows only
// .exe binaries are considered — no script interpreter can be injected by
// planting a same-named .cmd/.bat.
func findExecutable(name string) (string, error) {
	if strings.ContainsRune(name, '/') || strings.ContainsRune(name, '\\') {
		return name, nil
	}
	extensions := []string{""}
	if runtime.GOOS == "windows" {
		extensions = []string{".exe"}
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		for _, ext := range extensions {
			candidate := filepath.Join(dir, name+ext)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
					continue
				}
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("git executable %q not found in PATH", name)
}

// gitRunner executes one git invocation. Tests may replace it; production
// always uses the real git binary with argument arrays and a context.
type gitRunner func(ctx context.Context, cwd string, env []string, stdin string, attachIO bool, args ...string) (stdout string, err error)

var runGit gitRunner = defaultRunGit

// startProcess executes one argv array against the fixed git binary — the
// only process-start primitive in this package; no shell is involved.
func startProcess(ctx context.Context, cwd string, env []string, stdin string, attachIO bool, args ...string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	exePath, err := findExecutable(gitBinary)
	if err != nil {
		return "", err
	}

	argv := append([]string{gitBinary}, args...)
	fullEnv := os.Environ()
	fullEnv = append(fullEnv, env...)

	var stdinFile *os.File
	if attachIO {
		stdinFile = os.Stdin
	} else if stdin != "" {
		reader, writer, err := os.Pipe()
		if err != nil {
			return "", err
		}
		stdinFile = reader
		go func() {
			_, _ = io.WriteString(writer, stdin)
			_ = writer.Close()
		}()
	} else {
		stdinFile, err = os.Open(os.DevNull)
		if err != nil {
			return "", err
		}
	}
	defer stdinFile.Close()

	collect := !attachIO
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return "", err
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		stdoutReader.Close()
		stdoutWriter.Close()
		return "", err
	}
	outFile, errFile := stdoutWriter, stderrWriter
	if attachIO {
		outFile, errFile = os.Stdout, os.Stderr
	}

	attr := &os.ProcAttr{
		Dir:   cwd,
		Env:   fullEnv,
		Files: []*os.File{stdinFile, outFile, errFile},
	}
	proc, err := os.StartProcess(exePath, argv, attr)
	stdoutWriter.Close()
	stderrWriter.Close()
	if err != nil {
		stdoutReader.Close()
		stderrReader.Close()
		return "", err
	}

	var outBuf, errBuf bytes.Buffer
	var wg sync.WaitGroup
	if collect {
		wg.Add(2)
		go func() { defer wg.Done(); _, _ = io.Copy(&outBuf, stdoutReader) }()
		go func() { defer wg.Done(); _, _ = io.Copy(&errBuf, stderrReader) }()
	} else {
		stdoutReader.Close()
		stderrReader.Close()
	}

	waitDone := make(chan error, 1)
	go func() { _, waitErr := proc.Wait(); waitDone <- waitErr }()
	var waitErr error
	select {
	case waitErr = <-waitDone:
	case <-ctx.Done():
		_ = proc.Kill()
		<-waitDone
		wg.Wait()
		stdoutReader.Close()
		stderrReader.Close()
		return "", errors.New("GIT_OPERATION_TIMEOUT")
	}
	wg.Wait()
	stdoutReader.Close()
	stderrReader.Close()

	out := outBuf.String()
	if waitErr != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = strings.TrimSpace(out)
		}
		if msg == "" {
			msg = fmt.Sprintf("git %s failed", args[0])
		}
		return out, fmt.Errorf("%s: %w", strings.Join(args, " "), errors.New(msg))
	}
	return strings.TrimRight(out, "\r\n"), nil
}

func defaultRunGit(ctx context.Context, cwd string, env []string, stdin string, attachIO bool, args ...string) (string, error) {
	finalArgs, err := gitArgs(args...)
	if err != nil {
		return "", err
	}
	return startProcess(ctx, cwd, env, stdin, attachIO, finalArgs...)
}

type gitCall struct {
	ctx      context.Context
	cwd      string
	env      []string
	stdin    string
	attachIO bool
}

func gitAt(ctx context.Context, cwd string) gitCall { return gitCall{ctx: ctx, cwd: cwd} }

func (c gitCall) allowFail(args ...string) (string, bool) {
	out, err := runGit(c.ctx, c.cwd, c.env, c.stdin, c.attachIO, args...)
	if err != nil {
		return strings.TrimSpace(out), false
	}
	return strings.TrimSpace(out), true
}

type gitError struct{ msg string }

func (e gitError) Error() string { return e.msg }

func (c gitCall) must(args ...string) string {
	out, err := runGit(c.ctx, c.cwd, c.env, c.stdin, c.attachIO, args...)
	if err != nil {
		panic(gitError{msg: strings.TrimSpace(err.Error())})
	}
	return strings.TrimSpace(out)
}

func (c gitCall) withEnv(env ...string) gitCall {
	c.env = append(c.env, env...)
	return c
}

func (c gitCall) withStdin(input string) gitCall {
	c.stdin = input
	return c
}

func (c gitCall) withAttachedIO() gitCall {
	c.attachIO = true
	return c
}

func scopeFlag(scope string) string {
	if scope == "local" {
		return "--local"
	}
	return "--global"
}

func (c gitCall) cfgGet(scope, key string) (string, bool) {
	value, ok := c.allowFail("config", scopeFlag(scope), "--get", key)
	return value, ok && value != ""
}

func (c gitCall) cfgGetEffective(key string) (string, bool) {
	value, ok := c.allowFail("config", "--get", key)
	return value, ok && value != ""
}

func (c gitCall) cfgSet(scope, key, value string) {
	c.must("config", scopeFlag(scope), key, value)
}

func (c gitCall) cfgUnset(scope, key string) {
	c.allowFail("config", scopeFlag(scope), "--unset", key)
}

func (c gitCall) repoRoot() string {
	root, ok := c.allowFail("rev-parse", "--show-toplevel")
	if !ok {
		return ""
	}
	return root
}

// globalGitconfigPath mirrors gitid: GIT_CONFIG_GLOBAL wins, else ~/.gitconfig.
func globalGitconfigPath() string {
	if env := os.Getenv("GIT_CONFIG_GLOBAL"); env != "" {
		if abs, err := filepath.Abs(env); err == nil {
			return abs
		}
		return env
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".gitconfig")
}

// windowsDriveVariants adds the lower-case drive letter form; gitid records
// both spellings so includeIf matches regardless of how the path was
// spelled at clone time.
func windowsDriveVariants(patterns []string) []string {
	if runtime.GOOS != "windows" {
		return patterns
	}
	out := append([]string{}, patterns...)
	for _, pattern := range patterns {
		if len(pattern) >= 2 && pattern[1] == ':' && pattern[0] >= 'A' && pattern[0] <= 'Z' {
			lowered := string(pattern[0]+'a'-'A') + pattern[1:]
			out = append(out, lowered)
		}
	}
	return out
}
