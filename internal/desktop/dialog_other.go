//go:build gui && !windows

package desktop

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// zenityPath 在 PATH 中定位 zenity 可执行文件（os.StartProcess 不做
// PATH 搜索，Linux 上必须给全路径）。
func zenityPath() (string, error) {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, "zenity")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode().Perm()&0o111 != 0 {
			return candidate, nil
		}
	}
	return "", errCancelled
}

// pickFolderDialog 优先使用 zenity 的目录选择对话框；未安装时报不可用。
func pickFolderDialog() (string, error) {
	zenity, err := zenityPath()
	if err != nil {
		return "", err
	}
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		return "", err
	}
	attr := &os.ProcAttr{
		Files: []*os.File{os.Stdin, outWriter, outWriter},
	}
	proc, procErr := os.StartProcess(zenity,
		[]string{"zenity", "--file-selection", "--directory"}, attr)
	outWriter.Close()
	if procErr != nil {
		outReader.Close()
		return "", procErr
	}
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(&buf, outReader)
		close(done)
	}()
	waitErr := proc.Wait()
	<-done
	outReader.Close()
	if waitErr != nil {
		return "", errCancelled
	}
	path := strings.TrimSpace(buf.String())
	if path == "" {
		return "", errCancelled
	}
	return path, nil
}
