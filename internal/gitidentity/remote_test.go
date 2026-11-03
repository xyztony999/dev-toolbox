package gitidentity

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func remoteSetup(t *testing.T) string {
	t.Helper()
	store := sandbox(t)
	repo := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	_ = store
	if _, err := rawGit(repo, "remote", "add", "origin", "https://github.com/demo/demo.git"); err != nil {
		t.Fatalf("add origin: %v", err)
	}
	return repo
}

func remoteNames(t *testing.T, repo string) map[string]RemoteInfo {
	t.Helper()
	remotes, err := RemoteList(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]RemoteInfo{}
	for _, item := range remotes {
		out[item.Name] = item
	}
	return out
}

func TestRemoteLifecycle(t *testing.T) {
	repo := remoteSetup(t)
	ctx := context.Background()

	// 新增
	if _, err := RemoteAdd(ctx, repo, "mirror", "https://gitee.com/demo/demo.git"); err != nil {
		t.Fatalf("add: %v", err)
	}
	remotes := remoteNames(t, repo)
	if _, ok := remotes["mirror"]; !ok {
		t.Fatalf("mirror remote missing: %+v", remotes)
	}
	// 重名拒绝
	if _, err := RemoteAdd(ctx, repo, "mirror", "https://x.example.com/a.git"); err == nil || !strings.HasPrefix(err.Error(), "GIT_REMOTE_EXISTS") {
		t.Fatalf("expected GIT_REMOTE_EXISTS, got %v", err)
	}
	// 改地址
	if _, err := RemoteSetUrl(ctx, repo, "mirror", "https://gitee.com/demo/demo2.git"); err != nil {
		t.Fatalf("set-url: %v", err)
	}
	if got := remoteNames(t, repo)["mirror"].URL; got != "https://gitee.com/demo/demo2.git" {
		t.Fatalf("set-url not applied: %q", got)
	}
	// 重命名（迁移全部 remote.* 键）
	if _, err := RemoteRename(ctx, repo, "mirror", "gitee"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	remotes = remoteNames(t, repo)
	if _, ok := remotes["mirror"]; ok {
		t.Fatal("old name still present after rename")
	}
	if got := remotes["gitee"].URL; got != "https://gitee.com/demo/demo2.git" {
		t.Fatalf("rename lost url: %+v", remotes["gitee"])
	}
	// 删除
	if _, err := RemoteRemove(ctx, repo, "gitee"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, ok := remoteNames(t, repo)["gitee"]; ok {
		t.Fatal("remote still present after remove")
	}
	// 不存在
	if _, err := RemoteRemove(ctx, repo, "gitee"); err == nil || !strings.HasPrefix(err.Error(), "GIT_REMOTE_NOT_FOUND") {
		t.Fatalf("expected GIT_REMOTE_NOT_FOUND, got %v", err)
	}
}

func TestRemoteValidationRejectsInjection(t *testing.T) {
	repo := remoteSetup(t)
	ctx := context.Background()
	bad := []struct{ name, url string }{
		{"..", "https://x.example.com/a.git"},                // 配置段注入
		{"a..b", "https://x.example.com/a.git"},              // 段内 .. 序列
		{"origin/x", "https://x.example.com/a.git"},          // 斜杠
		{"bad name", "https://x.example.com/a.git"},          // 空白
		{"good", "not a url"},                                // 非法 URL
		{"good", "file:///C:/Windows/win.ini"},               // 协议白名单外
		{"good", "https://x.example.com/a.git\nurl.other=1"}, // 换行注入
	}
	for _, item := range bad {
		if _, err := RemoteAdd(ctx, repo, item.name, item.url); err == nil {
			t.Fatalf("expected rejection for %q / %q", item.name, item.url)
		}
	}
	// scp 形态与各协议应通过校验
	for _, url := range []string{"git@github.com:demo/demo.git", "ssh://git@github.com/demo/demo.git", "git://example.com/demo.git"} {
		if _, err := RemoteAdd(ctx, repo, "probe", url); err != nil {
			t.Fatalf("valid url rejected: %v (%s)", err, url)
		}
		if _, err := RemoteRemove(ctx, repo, "probe"); err != nil {
			t.Fatal(err)
		}
	}
	// 非仓库
	if _, err := RemoteList(ctx, filepath.Join(repo, "..")); err == nil || !strings.HasPrefix(err.Error(), "GIT_NOT_A_REPOSITORY") {
		t.Fatalf("expected GIT_NOT_A_REPOSITORY, got %v", err)
	}
	_ = os.Environ()
}
