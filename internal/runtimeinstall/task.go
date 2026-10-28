package runtimeinstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"dev-toolbox/internal/userstate"
)

// Manager implements the runtime.install.* application service contract.
// CLI and WebView share one process-wide instance; plans live in memory and
// expire, so a start must follow its preview in the same session.
type Manager struct {
	mu     sync.Mutex
	plans  map[string]InstallPlan
	tasks  map[string]*taskRun
	order  []string // finished task ids, oldest first, for pruning
	events []Event
	seq    uint64
	sink   func(Event)

	nextTaskID uint64
}

type taskRun struct {
	task     InstallTask
	cancel   context.CancelFunc
	finished bool
	lastEmit time.Time
}

func NewManager() *Manager {
	return &Manager{plans: map[string]InstallPlan{}, tasks: map[string]*taskRun{}}
}

// Default is the shared manager for CLI and desktop host.
var Default = NewManager()

// SetSink registers a host push callback for install events (the WebView
// bridge forwards them into the page). The callback must be quick and
// non-blocking.
func (m *Manager) SetSink(sink func(Event)) {
	m.mu.Lock()
	m.sink = sink
	m.mu.Unlock()
}

func (m *Manager) Options(tool string) ([]InstallOption, error) {
	return installOptions(tool)
}

// Preview resolves a concrete install plan. Network metadata queries are
// allowed; nothing is downloaded and no install directory is touched.
func (m *Manager) Preview(ctx context.Context, tool, selector, mirror string, expectedRevision uint64) (*InstallPlan, error) {
	p := providers[tool]
	if p == nil {
		return nil, fmt.Errorf("RUNTIME_PROVIDER_UNSUPPORTED: unknown tool %q", tool)
	}
	if expectedRevision != 0 {
		state, err := userstate.Get()
		if err != nil {
			return nil, err
		}
		if state.Revision != expectedRevision {
			return nil, errors.New("RUNTIME_REVISION_CONFLICT")
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	planCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	plan, err := p.preview(planCtx, fetcherFor(), selector, mirror)
	if err != nil {
		return nil, err
	}
	if _, err := os.Lstat(plan.TargetHome); err == nil {
		plan.TargetExists = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	m.mu.Lock()
	m.plans[plan.PlanID] = *plan
	m.mu.Unlock()
	return plan, nil
}

// Start confirms a previewed plan. Inside the registration lock it re-checks
// digest, expiry and user-state revision; resubmitting an active plan
// returns the existing task.
func (m *Manager) Start(planID, planDigest string, expectedRevision uint64, confirm bool) (InstallTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	plan, ok := m.plans[planID]
	if !ok {
		return InstallTask{}, errors.New("RUNTIME_PLAN_EXPIRED: unknown or expired plan")
	}
	if time.Now().After(plan.ExpiresAt) {
		delete(m.plans, planID)
		return InstallTask{}, errors.New("RUNTIME_PLAN_EXPIRED")
	}
	if computePlanDigest(plan) != planDigest {
		return InstallTask{}, errors.New("RUNTIME_PLAN_STALE")
	}
	if !confirm {
		return InstallTask{}, errors.New("RUNTIME_CONFIRMATION_REQUIRED")
	}
	if expectedRevision != 0 {
		state, err := userstate.Get()
		if err != nil {
			return InstallTask{}, err
		}
		if state.Revision != expectedRevision {
			return InstallTask{}, errors.New("RUNTIME_REVISION_CONFLICT")
		}
	}
	for _, run := range m.tasks {
		if run.task.PlanID == planID && !run.finished {
			return run.task, nil
		}
	}
	p := providers[plan.Tool]
	if p == nil {
		return InstallTask{}, fmt.Errorf("RUNTIME_PROVIDER_UNSUPPORTED: unknown tool %q", plan.Tool)
	}
	if _, err := os.Lstat(plan.TargetHome); err == nil {
		if err := p.checkTarget(&plan); err != nil {
			return InstallTask{}, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return InstallTask{}, err
	}
	if plan.EstimatedBytes > 0 {
		if root, err := RuntimeRoot(); err == nil {
			if free, err := freeBytes(root); err == nil && free < uint64(plan.EstimatedBytes) {
				return InstallTask{}, errors.New("RUNTIME_DISK_SPACE_INSUFFICIENT")
			}
		}
	}
	m.nextTaskID++
	task := InstallTask{
		TaskID:     fmt.Sprintf("install-task-%d", m.nextTaskID),
		PlanID:     planID,
		PlanDigest: planDigest,
		State:      StateQueued,
		Packages:   []PackageResult{},
	}
	for _, pkg := range plan.Packages {
		task.Packages = append(task.Packages, PackageResult{Name: pkg.Name, FileName: pkg.FileName, State: "pending"})
	}
	ctx, cancel := context.WithCancel(context.Background())
	run := &taskRun{task: task, cancel: cancel}
	m.tasks[task.TaskID] = run
	m.emitLocked(run)
	go m.execute(p, plan, run, ctx)
	return run.task, nil
}

func (m *Manager) execute(p provider, plan InstallPlan, run *taskRun, ctx context.Context) {
	report := func(progress InstallProgress) {
		m.mu.Lock()
		if !run.finished {
			if progress.Phase != "" && progress.Phase != run.task.State && isTaskPhase(progress.Phase) {
				run.task.State = progress.Phase
			}
			run.task.Progress = progress
			if progress.Phase != StateDownloading || time.Since(run.lastEmit) > 250*time.Millisecond {
				run.lastEmit = time.Now()
				m.emitLocked(run)
			}
		}
		m.mu.Unlock()
	}
	mark := func(name, state string, terr *TaskError) {
		m.mu.Lock()
		if !run.finished {
			for i := range run.task.Packages {
				if run.task.Packages[i].Name == name {
					run.task.Packages[i].State = state
					run.task.Packages[i].Error = terr
				}
			}
			run.task.Progress = InstallProgress{Phase: run.task.State, Package: name}
			m.emitLocked(run)
		}
		m.mu.Unlock()
	}
	installation, err := p.install(ctx, fetcherFor(), &plan, &installRun{report: report, mark: mark})

	m.mu.Lock()
	defer m.mu.Unlock()
	run.finished = true
	m.order = append(m.order, run.task.TaskID)
	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		run.task.State = StateCancelled
		run.task.Error = &TaskError{Code: "RUNTIME_INSTALL_CANCELLED", Message: "install cancelled by user; staged files were cleaned up"}
	case err != nil:
		run.task.Error = taskError(err)
		run.task.State = StateFailed
		done, failed := 0, 0
		for _, pkg := range run.task.Packages {
			switch pkg.State {
			case "done", "reused":
				done++
			case "failed":
				failed++
			}
		}
		if done > 0 && failed > 0 {
			run.task.State = StatePartial
		}
	default:
		run.task.Progress = InstallProgress{Phase: StateRegistering}
		state, regErr := userstate.RegisterDownloadedInstallation(installation)
		if regErr != nil {
			run.task.State = StateFailed
			run.task.Error = &TaskError{Code: "RUNTIME_INSTALL_FAILED", Message: "catalog registration failed: " + regErr.Error()}
		} else {
			for _, item := range state.Installations {
				if item.Runtime == installation.Runtime && item.LocalHome == installation.LocalHome {
					run.task.InstallationID = item.ID
					break
				}
			}
			run.task.State = StateComplete
		}
	}
	run.task.Progress = InstallProgress{Phase: run.task.State}
	m.emitLocked(run)
	m.pruneLocked()
}

func (m *Manager) emitLocked(run *taskRun) {
	m.seq++
	event := Event{Sequence: m.seq, TaskID: run.task.TaskID, Task: run.task}
	m.events = append(m.events, event)
	if len(m.events) > 1000 {
		m.events = m.events[len(m.events)-500:]
	}
	if m.sink != nil {
		m.sink(event)
	}
}

func (m *Manager) pruneLocked() {
	for len(m.order) > 32 {
		oldest := m.order[0]
		m.order = m.order[1:]
		delete(m.tasks, oldest)
	}
}

// Cancel cooperatively cancels a task. Once the commit phase started, the
// returned snapshot shows the current state instead of claiming a cancel.
func (m *Manager) Cancel(taskID string) (InstallTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok := m.tasks[taskID]
	if !ok {
		return InstallTask{}, errors.New("RUNTIME_TASK_NOT_FOUND")
	}
	if !run.finished {
		switch run.task.State {
		case StateRegistering, StateComplete, StateFailed, StateCancelled, StatePartial:
			// too late for a clean cancel; report the current state
		default:
			run.cancel()
		}
	}
	return run.task, nil
}

func (m *Manager) Status(taskID string) (InstallTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	run, ok := m.tasks[taskID]
	if !ok {
		return InstallTask{}, errors.New("RUNTIME_TASK_NOT_FOUND")
	}
	return run.task, nil
}

// Events returns install events newer than the given sequence so consumers
// can recover after a reconnect or drop stale duplicates.
func (m *Manager) Events(sinceSequence uint64) []Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []Event{}
	for _, event := range m.events {
		if event.Sequence > sinceSequence {
			out = append(out, event)
		}
	}
	return out
}

// WaitTerminal blocks until the task reaches a terminal state; used by the
// CLI foreground install flow.
func (m *Manager) WaitTerminal(taskID string, pollInterval time.Duration) (InstallTask, error) {
	for {
		task, err := m.Status(taskID)
		if err != nil {
			return InstallTask{}, err
		}
		switch task.State {
		case StateComplete, StateFailed, StateCancelled, StatePartial:
			return task, nil
		}
		time.Sleep(pollInterval)
	}
}
