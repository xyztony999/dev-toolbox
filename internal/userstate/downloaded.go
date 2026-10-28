package userstate

import (
	"errors"
	"path/filepath"
)

// RegisterDownloadedInstallation records one completed provider install into
// the user catalog. It appends only; global defaults stay untouched, and
// re-registering the same runtime home is a no-op.
func RegisterDownloadedInstallation(item Installation) (State, error) {
	path, err := StatePath()
	if err != nil {
		return State{}, err
	}
	unlock, err := lock(path)
	if err != nil {
		return State{}, err
	}
	defer unlock()
	state, path, err := readState()
	if err != nil {
		return State{}, err
	}
	home, err := filepath.Abs(filepath.Clean(item.LocalHome))
	if err != nil || item.Runtime == "" {
		return State{}, errors.New("RUNTIME_TARGET_INVALID")
	}
	item.LocalHome = home
	item.ID = installationID(item.Runtime, home)
	if item.Source == "" {
		item.Source = "downloaded"
	}
	item.Available = isDirectory(home)
	if findInstallation(state.Installations, item.ID) != nil {
		return state, nil
	}
	state.Installations = append(state.Installations, item)
	state.Revision++
	if err := writeState(path, state); err != nil {
		return State{}, err
	}
	return state, nil
}
