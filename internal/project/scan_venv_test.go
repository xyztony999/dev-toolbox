package project

import "testing"

// TestShouldIgnoreDirVenvVariants：venv 族自定义变体必须被忽略——
// .venv-workers 深入 site-packages 会触发深度上限，把整次扫描打成
// partial，阻止新判定快照落地（2026-11-23 用户重扫场景）。
func TestShouldIgnoreDirVenvVariants(t *testing.T) {
	for _, name := range []string{".venv", "venv", ".venv-workers", ".venv-prod", "Venv", ".VENV-X"} {
		if !shouldIgnoreDir(name) {
			t.Fatalf("%q must be ignored", name)
		}
	}
	for _, name := range []string{"env", "virtualenv", "vault", "src"} {
		if shouldIgnoreDir(name) {
			t.Fatalf("%q must not be ignored", name)
		}
	}
}
