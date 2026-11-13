// freemagpie // The user's own offer states (API.md §4): saved,
// claimed, dismissed. They live in freemagpie.json beside magpie's
// other files, so they are there before the user signs in and after
// they sign out (§1.2.1: not signing in still records, locally); the
// signed-in sync is a later step that pushes and pulls these rows.
package freemagpie

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/filelock"
)

// stateMu keeps two magpies (the app and a CLI) and this process's own
// calls from reading, changing and writing the file over each other;
// the lock is taken on the file as well (filelock.Mutex).
var stateMu = filelock.Mutex{
	Path:   func() string { return stateFile() + ".lock" },
	Locked: func() { states.readDisk() },
}

// stateFile is the states' file, its own, never settings.json.
func stateFile() string { return appdir.Join(appdir.Config(), "freemagpie.json") }

// states is every row the file holds, by offer slug. A read that fails
// is unknown (the error is kept and answered), never an empty set: a
// guard written over unknown data must not see "nothing saved".
type stateStore struct {
	mu     sync.Mutex
	rows   map[string]OfferState
	read   error
	readAt time.Time
}

// States is the offer states of this user, read and written through
// SetState.
var states stateStore

// readDisk loads the file once, or keeps the error. Unreadable is not
// empty (code-standards: a failed read means unknown).
func (s *stateStore) readDisk() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rows != nil || s.read != nil {
		return
	}
	b, err := os.ReadFile(stateFile())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.rows, s.read = map[string]OfferState{}, nil
			return
		}
		s.read = err
		return
	}
	var rows []OfferState
	if err := json.Unmarshal(b, &rows); err != nil {
		s.read = err
		return
	}
	m := make(map[string]OfferState, len(rows))
	for _, r := range rows {
		m[r.OfferSlug] = r
	}
	s.rows, s.read, s.readAt = m, nil, time.Now()
}

// writeDisk keeps the rows under the lock, whole-file (temp, rename),
// keeping the file it replaces as .bak (plugins.yml's own pattern).
func (s *stateStore) writeDisk() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := make([]OfferState, 0, len(s.rows))
	for _, r := range s.rows {
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].OfferSlug < rows[j].OfferSlug })
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	tmp := stateFile() + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if _, err := os.Stat(stateFile()); err == nil {
		_ = os.Rename(stateFile(), stateFile()+".bak")
	}
	if err := os.Rename(tmp, stateFile()); err != nil {
		return err
	}
	return nil
}

// States answers the user's offer states, and an error when the file
// can't be read (unknown, not empty).
func States() (map[string]OfferState, error) {
	stateMu.Lock()
	defer stateMu.Unlock()
	states.mu.Lock()
	defer states.mu.Unlock()
	if states.read != nil {
		return nil, states.read
	}
	out := make(map[string]OfferState, len(states.rows))
	for k, v := range states.rows {
		out[k] = v
	}
	return out, nil
}

// SetState sets one offer's state of the user's (API.md §4's PUT),
// keeping it on disk. A state of "" takes the row away (un-save).
// When the file can't be read, nothing is written: unknown is not
// written over.
func SetState(slug, state string, claimedAt *time.Time) error {
	stateMu.Lock()
	defer stateMu.Unlock()
	states.mu.Lock()
	if states.read != nil {
		states.mu.Unlock()
		return states.read
	}
	if state == "" {
		delete(states.rows, slug)
	} else {
		row := OfferState{OfferSlug: slug, State: state, ClaimedAt: claimedAt, UpdatedAt: time.Now().UTC()}
		states.rows[slug] = row
	}
	states.mu.Unlock()
	return states.writeDisk()
}

// ResetStatesForTest forgets what was read, so a test sees the file as
// a fresh process would. Only for tests. The store itself stays: its
// mutex is unlocked after this, so swapping it whole would leave the
// deferred unlock on a fresh, unlocked one.
func ResetStatesForTest() {
	states.mu.Lock()
	defer states.mu.Unlock()
	states.rows, states.read, states.readAt = nil, nil, time.Time{}
}
