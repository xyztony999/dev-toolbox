//go:build gui

package desktop

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"dev-toolbox/internal/gitidentity"
	"dev-toolbox/internal/project"
	"dev-toolbox/internal/runtimecatalog"
	"dev-toolbox/internal/runtimeinstall"
	"dev-toolbox/internal/userstate"
	webview "github.com/webview/webview_go"
)

//go:embed ui/index.html
var ui string

type request struct {
	RequestID string                     `json:"requestId"`
	Operation string                     `json:"operation"`
	Payload   map[string]json.RawMessage `json:"payload"`
}

type response struct {
	RequestID string `json:"requestId"`
	OK        bool   `json:"ok"`
	Data      any    `json:"data,omitempty"`
	Error     *fault `json:"error,omitempty"`
}

type fault struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Run() {
	runtime.LockOSThread()
	w := webview.New(false)
	if w == nil {
		panic("create WebView2 window failed")
	}
	defer w.Destroy()
	w.SetTitle("DevToolbox")
	w.SetSize(1180, 800, webview.HintNone)
	setWindowIcon(w)
	// The Bind callback runs on the UI thread; doing work here would freeze
	// the window for every process-spawning operation. Each request is
	// therefore acknowledged immediately, processed on a goroutine, and the
	// response is delivered back through __devToolboxResolve (WV-05).
	if err := w.Bind("__devToolboxInvoke", func(raw string) (map[string]any, error) {
		var req request
		if err := json.Unmarshal([]byte(raw), &req); err != nil {
			return nil, err
		}
		go func() {
			resp := invoke(req)
			payload, err := json.Marshal(resp)
			if err != nil {
				payload, _ = json.Marshal(response{RequestID: req.RequestID, OK: false, Error: &fault{Code: "RESPONSE_MARSHAL_FAILED", Message: err.Error()}})
			}
			w.Dispatch(func() {
				w.Eval("__devToolboxResolve(" + strconv.Quote(req.RequestID) + "," + string(payload) + ")")
			})
		}()
		return map[string]any{"accepted": true, "requestId": req.RequestID}, nil
	}); err != nil {
		panic(err)
	}
	if err := w.Bind("__devToolboxCancel", func(string) {}); err != nil {
		panic(err)
	}
	w.Init(`window.devToolbox={invoke:async r=>window.__devToolboxInvoke(JSON.stringify(r)),cancel:id=>window.__devToolboxCancel(id)};`)
	// Host-push install events: the manager sink marshals on its own
	// goroutine and Eval happens on the UI thread via Dispatch.
	runtimeinstall.Default.SetSink(func(evt runtimeinstall.Event) {
		payload, err := json.Marshal(evt)
		if err != nil {
			return
		}
		w.Dispatch(func() {
			w.Eval("window.__devToolboxEvent && window.__devToolboxEvent(" + string(payload) + ")")
		})
	})
	w.SetHtml(ui)
	w.Run()
}

func invoke(req request) response {
	if req.RequestID == "" || len(req.RequestID) > 128 {
		return failed(req.RequestID, errors.New("INVALID_REQUEST"))
	}
	ctx := context.Background()
	var data any
	var err error
	switch req.Operation {
	case "project.list":
		data, err = project.List()
	case "project.scanPreview":
		var p struct {
			RootPath string `json:"rootPath"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = project.ScanPreview(ctx, p.RootPath)
		}
	case "project.register":
		var p struct {
			RootPath       string `json:"rootPath"`
			DisplayName    string `json:"displayName"`
			CandidateID    string `json:"candidateId"`
			EvidenceDigest string `json:"expectedEvidenceDigest"`
		}
		if err = decode(req.Payload, &p); err == nil {
			var item project.Project
			var created bool
			item, created, err = project.Register(ctx, p.RootPath, p.DisplayName, p.CandidateID, p.EvidenceDigest)
			_ = created
			data = item
		}
	case "project.get":
		var p struct {
			ID string `json:"id"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = project.Get(p.ID)
		}
	case "project.rescan":
		var p struct {
			ID       string `json:"projectId"`
			Revision uint64 `json:"expectedRevision"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = project.Rescan(ctx, p.ID, p.Revision)
		}
	case "project.unregister":
		var p struct {
			ID       string `json:"projectId"`
			Revision uint64 `json:"expectedRevision"`
			Confirm  bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else {
				var removed bool
				removed, err = project.Unregister(p.ID, p.Revision)
				data = map[string]bool{"removed": removed}
			}
		}
	case "runtime.getUserState":
		data, err = userstate.Get()
	case "runtime.discover":
		var p struct {
			Tools []string `json:"tools"`
			Roots []string `json:"roots"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = runtimecatalog.Discover(ctx, p.Tools, p.Roots)
		}
	case "runtime.inspectPath":
		var p struct {
			Tool string `json:"tool"`
			Home string `json:"home"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = runtimecatalog.Inspect(p.Tool, p.Home, "user")
		}
	case "runtime.register":
		var p struct {
			Candidates       []runtimecatalog.CandidateRef `json:"candidates"`
			ExpectedRevision uint64                        `json:"expectedRevision"`
			Confirm          bool                          `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			var state userstate.State
			var counts userstate.RegistrationCounts
			state, counts, err = runtimecatalog.Register(ctx, p.Candidates, p.ExpectedRevision, p.Confirm)
			if err == nil {
				data = map[string]any{"state": state, "counts": counts}
			}
		}
	case "legacy.inspectDevSwitch":
		data, err = userstate.InspectLegacy()
	case "legacy.previewDevSwitchImport":
		data, err = userstate.PreviewLegacy(ctx)
	case "legacy.importDevSwitch":
		var p struct {
			ExpectedDigest string `json:"expectedSourceDigest"`
			ConfirmBackup  bool   `json:"confirmBackup"`
			ConfirmImport  bool   `json:"confirmImport"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = userstate.ImportLegacy(ctx, userstate.ImportOptions{ExpectedDigest: p.ExpectedDigest, ConfirmBackup: p.ConfirmBackup, ConfirmImport: p.ConfirmImport})
		}
	case "runtime.setGlobalDefault":
		var p struct {
			Runtime        string `json:"runtime"`
			InstallationID string `json:"installationId"`
			Revision       uint64 `json:"expectedRevision"`
			Confirm        bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = userstate.SetGlobalDefault(p.Runtime, p.InstallationID, p.Revision, p.Confirm)
		}
	case "runtime.install.options":
		var p struct {
			Tool string `json:"tool"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = runtimeinstall.Default.Options(p.Tool)
		}
	case "runtime.install.preview":
		var p struct {
			Tool             string `json:"tool"`
			Selector         string `json:"selector"`
			Mirror           string `json:"mirror"`
			ExpectedRevision uint64 `json:"expectedRevision"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = runtimeinstall.Default.Preview(ctx, p.Tool, p.Selector, p.Mirror, p.ExpectedRevision)
		}
	case "runtime.install.start":
		var p struct {
			PlanID           string `json:"planId"`
			PlanDigest       string `json:"planDigest"`
			ExpectedRevision uint64 `json:"expectedRevision"`
			Confirm          bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = runtimeinstall.Default.Start(p.PlanID, p.PlanDigest, p.ExpectedRevision, p.Confirm)
		}
	case "runtime.install.status":
		var p struct {
			TaskID string `json:"taskId"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = runtimeinstall.Default.Status(p.TaskID)
		}
	case "runtime.install.cancel":
		var p struct {
			TaskID string `json:"taskId"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = runtimeinstall.Default.Cancel(p.TaskID)
		}
	case "runtime.install.events":
		var p struct {
			SinceSequence uint64 `json:"sinceSequence"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data = runtimeinstall.Default.Events(p.SinceSequence)
		}
	case "git.bootstrap":
		store := gitStore()
		profiles, marks, listErr := gitidentity.ListProfiles(ctx, store)
		if listErr != nil {
			err = listErr
		} else {
			settings := store.Settings
			data = map[string]any{
				"profiles":    profiles,
				"marks":       marks,
				"credentials": gitidentity.CredentialList(ctx, store, ""),
				"settings":    settings,
			}
		}
	case "git.listProfiles":
		var data0 any
		profiles, marks, err0 := gitidentity.ListProfiles(ctx, gitStore())
		if err0 != nil {
			err = err0
		} else {
			data0 = map[string]any{"profiles": profiles, "marks": marks}
		}
		data = data0
	case "git.saveProfile":
		var p struct {
			Profile gitidentity.Profile `json:"profile"`
			Confirm bool                `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else if err = p.Profile.Validate(); err == nil {
				if err = gitSaveProfileOp(p.Profile); err == nil {
					data = p.Profile
				}
			}
		}
	case "git.removeProfile":
		var p struct {
			ID      string `json:"id"`
			Confirm bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else {
				err = gitidentity.RemoveProfile(ctx, gitStore(), p.ID)
				data = map[string]bool{"removed": err == nil}
			}
		}
	case "git.importGlobalIdentity":
		var p struct {
			AsID    string `json:"asId"`
			Confirm bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else {
				data, err = gitidentity.ImportGlobalIdentity(ctx, gitStore(), p.AsID)
			}
		}
	case "git.applyProfile":
		var p struct {
			ID      string   `json:"id"`
			Scope   string   `json:"scope"`
			Repos   []string `json:"repos"`
			DryRun  bool     `json:"dryRun"`
			Confirm bool     `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if p.DryRun {
				if p.Scope == "local" && len(p.Repos) == 0 {
					p.Repos = []string{"."}
				}
				data = gitidentity.ApplyToRepos(ctx, gitStore(), p.ID, p.Repos, true)
			} else if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else if p.Scope == "global" {
				var changes []gitidentity.ConfigChange
				changes, err = gitidentity.ApplyProfile(ctx, gitStore(), p.ID, "global", "", false)
				data = map[string]any{"scope": "global", "changes": changes}
			} else if len(p.Repos) == 0 {
				p.Repos = []string{"."}
				data = gitidentity.ApplyToRepos(ctx, gitStore(), p.ID, p.Repos, false)
			} else {
				data = gitidentity.ApplyToRepos(ctx, gitStore(), p.ID, p.Repos, false)
			}
		}
	case "git.effectiveIdentity":
		var p struct {
			Path   string `json:"path"`
			Prompt bool   `json:"prompt"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = gitidentity.ResolveEffective(ctx, gitStore(), p.Path, p.Prompt)
		}
	case "git.bindProfile":
		var p struct {
			ID      string `json:"id"`
			Dir     string `json:"dir"`
			Confirm bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else {
				var profile *gitidentity.Profile
				profile, _, err = gitidentity.Bind(ctx, gitStore(), p.ID, p.Dir)
				data = profile
			}
		}
	case "git.unbindProfile":
		var p struct {
			ID      string `json:"id"`
			Dir     string `json:"dir"`
			Confirm bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else {
				var profile *gitidentity.Profile
				var drift *gitidentity.SnippetDrift
				profile, drift, err = gitidentity.Unbind(ctx, gitStore(), p.ID, p.Dir)
				data = map[string]any{"profile": profile, "drift": drift}
			}
		}
	case "git.clearIdentity":
		var p struct {
			Scope   string `json:"scope"`
			Path    string `json:"path"`
			Confirm bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else {
				data = gitidentity.UnsetIdentity(ctx, gitStore(), p.Scope, p.Path)
			}
		}
	case "git.scanRepos":
		var p struct {
			Root  string `json:"root"`
			Depth int    `json:"depth"`
		}
		if err = decode(req.Payload, &p); err == nil {
			store, storeErr := gitidentity.Load()
			if storeErr != nil {
				err = storeErr
			} else {
				if p.Root == "" {
					p.Root = store.Settings.ScanRoot
				}
				if p.Root == "" {
					p.Root = "."
				}
				if p.Depth <= 0 {
					p.Depth = store.Settings.ScanDepth
				}
				data, err = gitidentity.ScanRepos(ctx, store, p.Root, p.Depth)
			}
		}
	case "git.saveScanRoot":
		var p struct {
			Root    string `json:"root"`
			Depth   int    `json:"depth"`
			Confirm bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else {
				store, storeErr := gitidentity.Load()
				if storeErr != nil {
					err = storeErr
				} else {
					err = gitidentity.SaveScanSettings(store, p.Root, p.Depth)
					data = store.Settings
				}
			}
		}
	case "git.importScanFindings":
		var p struct {
			Root    string   `json:"root"`
			Depth   int      `json:"depth"`
			Paths   []string `json:"paths"`
			DryRun  bool     `json:"dryRun"`
			Confirm bool     `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.DryRun && !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else {
				store, storeErr := gitidentity.Load()
				if storeErr != nil {
					err = storeErr
				} else if p.Root == "" {
					err = errors.New("GIT_SCAN_ROOT_INVALID: root required")
				} else {
					data, err = gitidentity.ImportScanManuals(ctx, store, p.Root, p.Depth, p.Paths, p.DryRun)
				}
			}
		}
	case "git.auditRepo":
		var p struct {
			Repo              string `json:"repo"`
			ExpectedProfileID string `json:"expectedProfileId"`
			Strict            bool   `json:"strict"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = gitidentity.AuditRepo(ctx, gitStore(), p.Repo, p.ExpectedProfileID)
		}
	case "git.gitDoctor":
		data, err = gitidentity.GitDoctor(ctx, gitStore())
	case "git.mirrorList":
		var p struct {
			Repo string `json:"repo"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data = gitidentity.MirrorList(ctx, p.Repo)
		}
	case "git.mirrorSet":
		var p struct {
			Repo      string `json:"repo"`
			Remote    string `json:"remote"`
			MirrorURL string `json:"mirrorUrl"`
			Confirm   bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else if p.Remote == "" {
				p.Remote = "origin"
			} else if p.Repo == "" {
				p.Repo = "."
			}
			if err == nil {
				var urls []string
				urls, err = gitidentity.MirrorSet(ctx, p.Repo, p.Remote, p.MirrorURL)
				data = map[string]any{"remote": p.Remote, "pushUrls": urls}
			}
		}
	case "git.mirrorUnset":
		var p struct {
			Repo    string `json:"repo"`
			Remote  string `json:"remote"`
			Confirm bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else {
				if p.Remote == "" {
					p.Remote = "origin"
				}
				if p.Repo == "" {
					p.Repo = "."
				}
				var removed []string
				removed, err = gitidentity.MirrorUnset(ctx, p.Repo, p.Remote)
				data = map[string]any{"remote": p.Remote, "removed": removed}
			}
		}
	case "git.credentialList":
		var p struct {
			Host string `json:"host"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data = gitidentity.CredentialList(ctx, gitStore(), p.Host)
		}
	case "git.credentialLogin":
		var p struct {
			Host     string `json:"host"`
			Username string `json:"username"`
		}
		if err = decode(req.Payload, &p); err == nil {
			var username string
			username, err = gitidentity.CredentialLogin(ctx, gitStore(), p.Host, p.Username)
			data = map[string]string{"host": p.Host, "username": username}
		}
	case "git.credentialSet":
		var p struct {
			Host     string `json:"host"`
			Username string `json:"username"`
			Token    string `json:"token"`
			Confirm  bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else {
				// one-shot channel: the token is used in this frame only
				token := p.Token
				err = gitidentity.CredentialStore(ctx, gitStore(), p.Host, p.Username, token)
				p.Token = ""
				token = ""
			}
		}
	case "git.credentialRemove":
		var p struct {
			Host     string `json:"host"`
			Username string `json:"username"`
			Confirm  bool   `json:"confirm"`
		}
		if err = decode(req.Payload, &p); err == nil {
			if !p.Confirm {
				err = errors.New("CONFIRMATION_REQUIRED")
			} else {
				err = gitidentity.CredentialErase(ctx, gitStore(), p.Host, p.Username)
				data = map[string]string{"host": p.Host}
			}
		}
	case "git.legacyPreview":
		previewData, previewErr := gitidentity.PreviewLegacy(ctx, gitStore())
		data, err = previewData, previewErr
	case "git.legacyImport":
		var p struct {
			SourceDigest  string   `json:"sourceDigest"`
			ProfileIDs    []string `json:"profileIds"`
			ConfirmBackup bool     `json:"confirmBackup"`
			ConfirmImport bool     `json:"confirmImport"`
		}
		if err = decode(req.Payload, &p); err == nil {
			data, err = gitidentity.LegacyImport(ctx, gitStore(), p.SourceDigest, p.ProfileIDs, p.ConfirmBackup, p.ConfirmImport)
		}
	default:
		err = errors.New("OPERATION_NOT_ALLOWED")
	}
	if err != nil {
		return failed(req.RequestID, err)
	}
	return response{RequestID: req.RequestID, OK: true, Data: data}
}

func decode(raw map[string]json.RawMessage, target any) error {
	data, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func failed(id string, err error) response {
	code := strings.Fields(err.Error())[0]
	if code == "" {
		code = "OPERATION_FAILED"
	}
	return response{RequestID: id, OK: false, Error: &fault{Code: code, Message: fmt.Sprint(err)}}
}
