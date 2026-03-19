// Package store provides an in-memory run store.
package store

import (
	"fmt"
	"regexp"
	"sort"
	"sync"
	"time"
)

// Status represents the state of a run.
type Status string

// Run status constants.
const (
	StatusPending  Status = "pending"
	StatusRunning  Status = "running"
	StatusComplete Status = "complete"
	StatusFailed   Status = "failed"
	StatusDropped  Status = "dropped"
)

// Mode represents the execution mode of a run.
type Mode string

// Run mode constants.
const (
	ModeInteractive Mode = "interactive"
	ModeHeadless    Mode = "headless"
)

// Run holds metadata for a single outpost session.
type Run struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	Mode            Mode       `json:"mode"`
	Status          Status     `json:"status"`
	BaseSHA         string     `json:"base_sha"`
	FinalSHA        string     `json:"final_sha"`
	CreatedAt       time.Time  `json:"created_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	Attach          string     `json:"attach"`
	LogTail         string     `json:"log_tail"`
	PatchReady      bool       `json:"patch_ready"`
	Branch          string     `json:"branch"`
	MaxTurns        int        `json:"max_turns"`
	Subdir          string     `json:"subdir"`
	SessionID       string     `json:"session_id"`
	ForkedSessionID string     `json:"forked_session_id"`
	SessionReady    bool       `json:"session_ready"`
	Dir             string     `json:"-"`
	Converting      bool       `json:"-"` // true during mode conversion; suppresses OnExit finalization
}

// Store is a thread-safe in-memory collection of runs.
type Store struct {
	mu   sync.RWMutex
	runs map[string]*Run
}

// New creates an empty Store.
func New() *Store {
	return &Store{runs: make(map[string]*Run)}
}

// Add inserts a run into the store.
func (s *Store) Add(r *Run) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[r.ID] = r
}

// Get returns a copy of the run with the given ID.
func (s *Store) Get(id string) (*Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[id]
	if !ok {
		return nil, fmt.Errorf("run %q not found", id)
	}
	cp := *r
	return &cp, nil
}

// GetStatus returns just the status of a run (cheaper than full Get).
func (s *Store) GetStatus(id string) (Status, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[id]
	if !ok {
		return "", fmt.Errorf("run %q not found", id)
	}
	return r.Status, nil
}

// List returns all runs sorted by created_at descending.
func (s *Store) List() []*Run {
	s.mu.RLock()
	defer s.mu.RUnlock()
	runs := make([]*Run, 0, len(s.runs))
	for _, r := range s.runs {
		cp := *r
		runs = append(runs, &cp)
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})
	return runs
}

// ActiveRuns returns the count of active runs and their summaries.
func (s *Store) ActiveRuns() (int, []Run) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var active []Run
	for _, r := range s.runs {
		if r.Status == StatusPending || r.Status == StatusRunning {
			active = append(active, *r)
		}
	}
	return len(active), active
}

// Update applies fn to the run in-place.
func (s *Store) Update(id string, fn func(*Run)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[id]
	if !ok {
		return fmt.Errorf("run %q not found", id)
	}
	fn(r)
	return nil
}

// Delete removes a run from the store.
func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.runs, id)
}

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from a string.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}
