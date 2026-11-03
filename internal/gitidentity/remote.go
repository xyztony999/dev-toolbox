package gitidentity

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// 仓库 remote 全生命周期管理（git-remote-management 模块）。
// 全部写操作经 `git remote` 子命令（参数数组）执行；名字与 URL 在
// 服务层先行校验——remote 名会拼进 remote.<name>.* 配置键，非法字符
// 可造成配置段注入，必须先拒绝。

var remoteNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// remoteURLPattern 接受 http(s)://、ssh://、git:// 与 user@host:path 的
// scp 形态；空白与换行由 gitArgs 参数级防御兜底拒绝。
var remoteURLPattern = regexp.MustCompile(`^(?:https?://[^\s]+|ssh://[^\s]+|git://[^\s]+|[A-Za-z0-9._~-]+@[A-Za-z0-9._-]+:[^\s]+)$`)

func validateRemoteName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || !remoteNamePattern.MatchString(name) || strings.Contains(name, "..") {
		return fmt.Errorf("GIT_REMOTE_NAME_INVALID: %q", name)
	}
	return nil
}

func validateRemoteURL(url string) error {
	url = strings.TrimSpace(url)
	if url == "" || !remoteURLPattern.MatchString(url) {
		return fmt.Errorf("GIT_REMOTE_URL_INVALID: %q", url)
	}
	return nil
}

func requireRepoRoot(ctx context.Context, repo string) (string, error) {
	root := gitAt(ctx, repo).repoRoot()
	if root == "" {
		return "", fmt.Errorf("GIT_NOT_A_REPOSITORY: %s", repo)
	}
	return root, nil
}

func findRemote(ctx context.Context, root, name string) (*RemoteInfo, error) {
	remotes := getRemotes(ctx, root)
	for i := range remotes {
		if remotes[i].Name == name {
			return &remotes[i], nil
		}
	}
	names := make([]string, 0, len(remotes))
	for _, item := range remotes {
		names = append(names, item.Name)
	}
	available := strings.Join(names, ", ")
	if available == "" {
		available = "（无）"
	}
	return nil, fmt.Errorf("GIT_REMOTE_NOT_FOUND: %q (available: %s)", name, available)
}

func remoteSnapshot(ctx context.Context, repo string) []RemoteInfo {
	remotes := MirrorList(ctx, repo)
	if remotes == nil {
		remotes = []RemoteInfo{}
	}
	return remotes
}

// RemoteAdd 新增 remote；重名报 GIT_REMOTE_EXISTS。
func RemoteAdd(ctx context.Context, repo, name, url string) ([]RemoteInfo, error) {
	if err := validateRemoteName(name); err != nil {
		return nil, err
	}
	if err := validateRemoteURL(url); err != nil {
		return nil, err
	}
	root, err := requireRepoRoot(ctx, repo)
	if err != nil {
		return nil, err
	}
	if _, exists := findRemote(ctx, root, name); exists == nil {
		return nil, fmt.Errorf("GIT_REMOTE_EXISTS: %q", name)
	}
	if out, ok := gitAt(ctx, root).allowFail("remote", "add", name, strings.TrimSpace(url)); !ok {
		return nil, fmt.Errorf("GIT_REMOTE_URL_INVALID: git remote add failed: %s", strings.TrimSpace(out))
	}
	return remoteSnapshot(ctx, root), nil
}

// RemoteRemove 删除 remote；不存在报 GIT_REMOTE_NOT_FOUND。
func RemoteRemove(ctx context.Context, repo, name string) ([]RemoteInfo, error) {
	if err := validateRemoteName(name); err != nil {
		return nil, err
	}
	root, err := requireRepoRoot(ctx, repo)
	if err != nil {
		return nil, err
	}
	if _, err := findRemote(ctx, root, name); err != nil {
		return nil, err
	}
	gitAt(ctx, root).allowFail("remote", "remove", name)
	return remoteSnapshot(ctx, root), nil
}

// RemoteRename 重命名 remote（git 迁移全部 remote.* 键）。
func RemoteRename(ctx context.Context, repo, from, to string) ([]RemoteInfo, error) {
	if err := validateRemoteName(to); err != nil {
		return nil, err
	}
	root, err := requireRepoRoot(ctx, repo)
	if err != nil {
		return nil, err
	}
	if _, err := findRemote(ctx, root, from); err != nil {
		return nil, err
	}
	if _, exists := findRemote(ctx, root, to); exists == nil {
		return nil, fmt.Errorf("GIT_REMOTE_EXISTS: %q", to)
	}
	if out, ok := gitAt(ctx, root).allowFail("remote", "rename", from, to); !ok {
		return nil, fmt.Errorf("GIT_REMOTE_NAME_INVALID: git remote rename failed: %s", strings.TrimSpace(out))
	}
	return remoteSnapshot(ctx, root), nil
}

// RemoteSetUrl 修改 fetch 地址（remote.<name>.url）。
func RemoteSetUrl(ctx context.Context, repo, name, url string) ([]RemoteInfo, error) {
	if err := validateRemoteURL(url); err != nil {
		return nil, err
	}
	root, err := requireRepoRoot(ctx, repo)
	if err != nil {
		return nil, err
	}
	if _, err := findRemote(ctx, root, name); err != nil {
		return nil, err
	}
	if out, ok := gitAt(ctx, root).allowFail("remote", "set-url", name, strings.TrimSpace(url)); !ok {
		return nil, fmt.Errorf("GIT_REMOTE_URL_INVALID: git remote set-url failed: %s", strings.TrimSpace(out))
	}
	return remoteSnapshot(ctx, root), nil
}

// RemoteList 返回全量 remote 信息（fetch/push/镜像/生效账号）。
func RemoteList(ctx context.Context, repo string) ([]RemoteInfo, error) {
	if _, err := requireRepoRoot(ctx, repo); err != nil {
		return nil, err
	}
	return remoteSnapshot(ctx, repo), nil
}
