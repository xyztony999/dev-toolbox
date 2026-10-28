package gitidentity

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// MirrorSet turns on per-repository mirror push: remote.<name>.pushurl
// becomes [original target, mirror]; fetch (remote.<name>.url) and the
// credential channel are untouched. Calling again replaces the mirror.
func MirrorSet(ctx context.Context, repo, remoteName, mirrorURL string) ([]string, error) {
	call := gitAt(ctx, repo)
	root := call.repoRoot()
	if root == "" {
		return nil, fmt.Errorf("GIT_NOT_A_REPOSITORY: %s", repo)
	}
	mirrorURL = strings.TrimSpace(mirrorURL)
	if mirrorURL == "" || strings.ContainsAny(mirrorURL, " \t") {
		return nil, errors.New("GIT_MIRROR_INVALID: mirror url must not be blank or contain spaces")
	}
	remotes := getRemotes(ctx, root)
	var remote *RemoteInfo
	for i := range remotes {
		if remotes[i].Name == remoteName {
			remote = &remotes[i]
		}
	}
	if remote == nil {
		names := make([]string, 0, len(remotes))
		for _, item := range remotes {
			names = append(names, item.Name)
		}
		return nil, fmt.Errorf("GIT_REPO_NOT_FOUND: remote %q does not exist (available: %s)", remoteName, strings.Join(names, ", "))
	}
	original := ""
	for _, pushURL := range remote.PushURLs {
		if pushURL != mirrorURL && original == "" {
			original = pushURL
		}
	}
	if original == "" {
		original = remote.URL
	}
	if original == "" {
		return nil, errors.New("GIT_REPO_NOT_FOUND: remote has no fetch url")
	}
	if original == mirrorURL {
		return nil, fmt.Errorf("GIT_MIRROR_INVALID: mirror url equals the existing target of %s", remoteName)
	}
	if len(remote.PushURLs) > 0 {
		call.allowFail("config", "--local", "--unset-all", fmt.Sprintf("remote.%s.pushurl", remoteName))
	}
	call.must("config", "--local", "--add", fmt.Sprintf("remote.%s.pushurl", remoteName), original)
	call.must("config", "--local", "--add", fmt.Sprintf("remote.%s.pushurl", remoteName), mirrorURL)
	return []string{original, mirrorURL}, nil
}

// MirrorUnset removes every pushurl so push falls back to the fetch url.
func MirrorUnset(ctx context.Context, repo, remoteName string) ([]string, error) {
	call := gitAt(ctx, repo)
	root := call.repoRoot()
	if root == "" {
		return nil, fmt.Errorf("GIT_NOT_A_REPOSITORY: %s", repo)
	}
	remotes := getRemotes(ctx, root)
	var remote *RemoteInfo
	for i := range remotes {
		if remotes[i].Name == remoteName {
			remote = &remotes[i]
		}
	}
	if remote == nil {
		return nil, fmt.Errorf("GIT_REPO_NOT_FOUND: remote %q does not exist", remoteName)
	}
	if len(remote.PushURLs) > 0 {
		call.allowFail("config", "--local", "--unset-all", fmt.Sprintf("remote.%s.pushurl", remoteName))
	}
	return remote.PushURLs, nil
}

// MirrorList lists remotes with their pushurls for one repository.
func MirrorList(ctx context.Context, repo string) []RemoteInfo {
	remotes := getRemotes(ctx, repo)
	sort.Slice(remotes, func(i, j int) bool { return remotes[i].Name < remotes[j].Name })
	return remotes
}
