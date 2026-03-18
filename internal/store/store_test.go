package store

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestStore_AddAndGet(t *testing.T) {
	t.Parallel()
	s := New()
	r := &Run{
		ID:        "test-1",
		Name:      "test run",
		Mode:      ModeHeadless,
		Status:    StatusRunning,
		BaseSHA:   "abc123",
		CreatedAt: time.Now(),
		MaxTurns:  50,
	}
	s.Add(r)

	got, err := s.Get("test-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "test-1" || got.Name != "test run" || got.Mode != ModeHeadless {
		t.Errorf("fields mismatch: got %+v", got)
	}
}

func TestStore_Get_ReturnsCopy(t *testing.T) {
	t.Parallel()
	s := New()
	s.Add(&Run{ID: "test-1", Status: StatusRunning})

	got, _ := s.Get("test-1")
	got.Status = StatusComplete

	original, _ := s.Get("test-1")
	if original.Status != StatusRunning {
		t.Errorf("Get returned a reference, not a copy")
	}
}

func TestStore_Get_NotFound(t *testing.T) {
	t.Parallel()
	s := New()
	_, err := s.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

func TestStore_GetStatus(t *testing.T) {
	t.Parallel()
	s := New()
	s.Add(&Run{ID: "test-1", Status: StatusRunning})

	st, err := s.GetStatus("test-1")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}
	if st != StatusRunning {
		t.Errorf("got %q, want %q", st, StatusRunning)
	}
}

func TestStore_GetStatus_NotFound(t *testing.T) {
	t.Parallel()
	s := New()
	_, err := s.GetStatus("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

func TestStore_List_SortOrder(t *testing.T) {
	t.Parallel()
	s := New()
	now := time.Now()
	s.Add(&Run{ID: "old", CreatedAt: now.Add(-2 * time.Hour)})
	s.Add(&Run{ID: "mid", CreatedAt: now.Add(-1 * time.Hour)})
	s.Add(&Run{ID: "new", CreatedAt: now})

	runs := s.List()
	if len(runs) != 3 {
		t.Fatalf("got %d runs, want 3", len(runs))
	}
	if runs[0].ID != "new" || runs[1].ID != "mid" || runs[2].ID != "old" {
		t.Errorf("wrong order: %s, %s, %s", runs[0].ID, runs[1].ID, runs[2].ID)
	}
}

func TestStore_List_Empty(t *testing.T) {
	t.Parallel()
	s := New()
	runs := s.List()
	if runs == nil {
		t.Error("List returned nil, want empty slice")
	}
	if len(runs) != 0 {
		t.Errorf("got %d runs, want 0", len(runs))
	}
}

func TestStore_ActiveRuns(t *testing.T) {
	t.Parallel()
	s := New()
	s.Add(&Run{ID: "pending", Status: StatusPending})
	s.Add(&Run{ID: "running", Status: StatusRunning})
	s.Add(&Run{ID: "complete", Status: StatusComplete})
	s.Add(&Run{ID: "failed", Status: StatusFailed})
	s.Add(&Run{ID: "dropped", Status: StatusDropped})

	count, active := s.ActiveRuns()
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if len(active) != 2 {
		t.Errorf("len(active) = %d, want 2", len(active))
	}

	ids := map[string]bool{}
	for _, r := range active {
		ids[r.ID] = true
	}
	if !ids["pending"] || !ids["running"] {
		t.Errorf("active runs: %v", ids)
	}
}

func TestStore_Update(t *testing.T) {
	t.Parallel()
	s := New()
	s.Add(&Run{ID: "test-1", Status: StatusRunning})

	err := s.Update("test-1", func(r *Run) {
		r.Status = StatusComplete
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := s.Get("test-1")
	if got.Status != StatusComplete {
		t.Errorf("status = %q, want %q", got.Status, StatusComplete)
	}
}

func TestStore_Update_NotFound(t *testing.T) {
	t.Parallel()
	s := New()
	err := s.Update("nonexistent", func(r *Run) {})
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

func TestStore_Delete(t *testing.T) {
	t.Parallel()
	s := New()
	s.Add(&Run{ID: "test-1"})
	s.Delete("test-1")

	_, err := s.Get("test-1")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	s := New()
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("run-%d", n)
			s.Add(&Run{ID: id, Status: StatusRunning, CreatedAt: time.Now()})
			_, _ = s.Get(id)
			_, _ = s.GetStatus(id)
			_ = s.List()
			count, _ := s.ActiveRuns()
			_ = count
			_ = s.Update(id, func(r *Run) { r.Status = StatusComplete })
			s.Delete(id)
		}(i)
	}

	wg.Wait()
}

func TestStripANSI(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"no ansi here", "no ansi here"},
		{"\x1b[31mred\x1b[0m", "red"},
		{"\x1b[1;32mbold green\x1b[0m normal", "bold green normal"},
		{"\x1b[38;5;196mextended\x1b[0m", "extended"},
	}

	for _, tt := range tests {
		got := StripANSI(tt.input)
		if got != tt.want {
			t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
