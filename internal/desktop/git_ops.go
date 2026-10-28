//go:build gui

package desktop

import (
	"dev-toolbox/internal/gitidentity"
)

// gitStore loads the identity store for bridge operations. A corrupt or
// missing store yields an empty in-memory store so read-only operations
// still answer; writes only reach disk after explicit user confirmation.
func gitStore() *gitidentity.Store {
	store, err := gitidentity.Load()
	if err != nil {
		return gitidentity.NewStore()
	}
	return store
}

func gitSaveProfileOp(profile gitidentity.Profile) error {
	store, err := gitidentity.Load()
	if err != nil {
		return err
	}
	if err := store.Upsert(profile); err != nil {
		return err
	}
	return store.Save()
}
