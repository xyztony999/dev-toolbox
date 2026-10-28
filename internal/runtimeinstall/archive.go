package runtimeinstall

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ulikunitz/xz"
)

func fileDigest(path, algo string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	var sum interface {
		io.Writer
		Sum([]byte) []byte
	}
	switch algo {
	case "sha256":
		sum = sha256.New()
	case "sha512":
		sum = sha512.New()
	default:
		return "", errors.New("RUNTIME_CHECKSUM_MISSING")
	}
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

func verifyChecksum(path, expected, algo string) error {
	actual, err := fileDigest(path, algo)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, strings.TrimSpace(expected)) {
		return fmt.Errorf("RUNTIME_CHECKSUM_MISMATCH: %s digest does not match the announced value", algo)
	}
	return nil
}

func isHex(value string) bool {
	if value == "" {
		return false
	}
	for _, c := range value {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func regularFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular()
}

// safeMember rejects absolute paths, backslashes and anything escaping root.
func safeMember(root, name string) (string, error) {
	if strings.Contains(name, "\\") || filepath.IsAbs(name) || !filepath.IsLocal(filepath.FromSlash(name)) {
		return "", fmt.Errorf("unsafe archive member path %q", name)
	}
	target := filepath.Join(root, filepath.FromSlash(name))
	abs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	base, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(base, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive member %q escapes the staging directory", name)
	}
	return abs, nil
}

func extractZipArchive(path, root string) error {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer archive.Close()
	if len(archive.File) > maxArchiveFiles {
		return errors.New("archive file count limit exceeded")
	}
	var expanded int64
	for _, entry := range archive.File {
		if entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive symlink member rejected: %s", entry.Name)
		}
		target, err := safeMember(root, entry.Name)
		if err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			continue
		}
		expanded += int64(entry.UncompressedSize64)
		if expanded > maxExpandedBytes {
			return errors.New("archive expansion limit exceeded")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		src, err := entry.Open()
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, entry.Mode().Perm()&0o700)
		if err != nil {
			src.Close()
			return err
		}
		_, copyErr := io.Copy(dst, io.LimitReader(src, int64(entry.UncompressedSize64)+1))
		closeSrc := src.Close()
		closeDst := dst.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeSrc != nil {
			return closeSrc
		}
		if closeDst != nil {
			return closeDst
		}
	}
	return nil
}

func extractTarArchive(path, root string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	lower := strings.ToLower(path)
	var reader io.Reader
	switch {
	case strings.HasSuffix(lower, ".tar.xz"), strings.HasSuffix(lower, ".txz"):
		xr, err := xz.NewReader(file)
		if err != nil {
			return err
		}
		reader = xr
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		gr, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		defer gr.Close()
		reader = gr
	default:
		return errors.New("unsupported archive format")
	}
	tarReader := tar.NewReader(reader)
	var expanded int64
	count := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		count++
		if count > maxArchiveFiles {
			return errors.New("archive file count limit exceeded")
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA && header.Typeflag != tar.TypeDir {
			return fmt.Errorf("archive link or special member rejected: %s", header.Name)
		}
		target, err := safeMember(root, header.Name)
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0o700); err != nil {
				return err
			}
			continue
		}
		if header.Size < 0 {
			return errors.New("invalid archive member size")
		}
		expanded += header.Size
		if expanded > maxExpandedBytes {
			return errors.New("archive expansion limit exceeded")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(header.Mode).Perm()&0o700)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(dst, io.LimitReader(tarReader, header.Size))
		closeErr := dst.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
}

func extractArchive(archivePath, root string) error {
	if strings.HasSuffix(strings.ToLower(archivePath), ".zip") {
		return extractZipArchive(archivePath, root)
	}
	return extractTarArchive(archivePath, root)
}

// extractSingleRoot unpacks one archive and returns its only top-level
// directory; distributions that ship more than one root are rejected.
func extractSingleRoot(archivePath, extractDir string) (string, error) {
	if err := os.MkdirAll(extractDir, 0o700); err != nil {
		return "", err
	}
	if err := extractArchive(archivePath, extractDir); err != nil {
		return "", fmt.Errorf("RUNTIME_ARCHIVE_UNSAFE: %w", err)
	}
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return "", errors.New("RUNTIME_ARCHIVE_UNSAFE: expected a single root directory")
	}
	return filepath.Join(extractDir, entries[0].Name()), nil
}

// moveInto publishes a staged directory to its final home. The final
// conflict check runs immediately before the rename; an existing target is
// never deleted or overwritten.
func moveInto(srcDir, targetHome string) error {
	if _, err := os.Lstat(targetHome); err == nil {
		return errors.New("RUNTIME_TARGET_CONFLICT")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(srcDir, targetHome)
}

func makeExecutable(path string) {
	if regularFile(path) {
		_ = os.Chmod(path, 0o755)
	}
}
