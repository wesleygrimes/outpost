package store

import (
	"sort"
	"sync"
	"time"
)

// Status represents the lifecycle state of a run.
type Status string

// Run status values.
const (
	StatusPending  Status = "pending"
	StatusRunning  Status = "running"
	StatusComplete Status = "complete"
	StatusFailed   Status = "failed"
	StatusKilled   Status = "killed"
)

// Run holds metadata for a single Outpost run.
type Run struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Mode       string     `json:"mode"`
	Status     Status     `json:"status"`
	BaseSHA    string     `json:"base_sha"`
	FinalSHA   string     `json:"final_sha,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	Attach     string     `json:"attach"`
	LogTail    string     `json:"log_tail,omitempty"`
	PatchReady bool       `json:"patch_ready"`
	Branch     string     `json:"branch,omitempty"`
	MaxTurns   int        `json:"max_turns,omitempty"`
	Subdir     string     `json:"subdir,omitempty"`
	Dir        string     `json:"-"`
}

// Store is a concurrency-safe in-memory run store.
type Store struct {
	mu   sync.RWMutex
	runs map[string]*Run
}

// New creates an empty Store.
func New() *Store {
	return &Store{
		runs: make(map[string]*Run),
	}
}

// Add inserts a run into the store.
func (s *Store) Add(r *Run) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runs[r.ID] = r
}

// Get returns a run by ID and whether it was found.
func (s *Store) Get(id string) (*Run, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.runs[id]
	return r, ok
}

// List returns all runs sorted by CreatedAt descending.
func (s *Store) List() []*Run {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Run, 0, len(s.runs))
	for _, r := range s.runs {
		result = append(result, r)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result
}

// Update applies fn to the run with the given ID while holding the write lock.
func (s *Store) Update(id string, fn func(*Run)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if r, ok := s.runs[id]; ok {
		fn(r)
	}
}

// ActiveCount returns the number of runs in pending or running state.
func (s *Store) ActiveCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, r := range s.runs {
		if r.Status == StatusPending || r.Status == StatusRunning {
			count++
		}
	}

	return count
}

// Delete removes a run from the store.
func (s *Store) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.runs, id)
}
