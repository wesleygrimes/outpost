package runner

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/wesgrimes/outpost/internal/store"
)

func TestBuildClaudeCmd_Headless_Plan(t *testing.T) {
	t.Parallel()
	cfg := &SpawnConfig{
		Mode:     store.ModeHeadless,
		MaxTurns: 50,
		PlanPath: "/tmp/plan.md",
	}
	cmd := BuildClaudeCmd(cfg)

	if !strings.Contains(cmd, "--print") {
		t.Error("headless should contain --print")
	}
	if !strings.Contains(cmd, "--permission-mode bypassPermissions") {
		t.Error("headless should contain --permission-mode bypassPermissions")
	}
	if !strings.Contains(cmd, "--max-turns 50") {
		t.Error("should contain --max-turns 50")
	}
	if strings.Contains(cmd, "--resume") {
		t.Error("headless plan mode should NOT contain --resume")
	}
	if !strings.Contains(cmd, `< "/tmp/plan.md"`) {
		t.Errorf("should redirect plan: got %q", cmd)
	}
}

func TestBuildClaudeCmd_Interactive_Plan(t *testing.T) {
	t.Parallel()
	cfg := &SpawnConfig{
		Mode:     store.ModeInteractive,
		MaxTurns: 50,
		PlanPath: "/tmp/plan.md",
	}
	cmd := BuildClaudeCmd(cfg)

	if strings.Contains(cmd, "--print") {
		t.Error("interactive should NOT contain --print")
	}
	if !strings.Contains(cmd, "--permission-mode bypassPermissions") {
		t.Error("interactive should contain --permission-mode bypassPermissions")
	}
	if !strings.Contains(cmd, "--max-turns 50") {
		t.Error("should contain --max-turns 50")
	}
}

func TestBuildClaudeCmd_Headless_Resume(t *testing.T) {
	t.Parallel()
	cfg := &SpawnConfig{
		Mode:      store.ModeHeadless,
		MaxTurns:  50,
		SessionID: "abc-123-def",
	}
	cmd := BuildClaudeCmd(cfg)

	if !strings.Contains(cmd, "--resume abc-123-def") {
		t.Errorf("should contain --resume with session ID: got %q", cmd)
	}
	if !strings.Contains(cmd, "--fork-session") {
		t.Errorf("should contain --fork-session: got %q", cmd)
	}
	if !strings.Contains(cmd, "--continue") {
		t.Errorf("headless resume should have --continue phase: got %q", cmd)
	}
	if !strings.Contains(cmd, "--max-turns 50") {
		t.Errorf("should contain --max-turns 50: got %q", cmd)
	}
	if strings.Contains(cmd, "< ") {
		t.Errorf("resume mode should NOT redirect stdin: got %q", cmd)
	}
}

func TestBuildClaudeCmd_Interactive_Resume(t *testing.T) {
	t.Parallel()
	cfg := &SpawnConfig{
		Mode:      store.ModeInteractive,
		MaxTurns:  30,
		SessionID: "sess-456",
	}
	cmd := BuildClaudeCmd(cfg)

	if !strings.Contains(cmd, "--resume sess-456") {
		t.Errorf("should contain --resume with session ID: got %q", cmd)
	}
	if !strings.Contains(cmd, "--fork-session") {
		t.Errorf("should contain --fork-session: got %q", cmd)
	}
	if !strings.Contains(cmd, "--max-turns 30") {
		t.Errorf("should contain --max-turns 30: got %q", cmd)
	}
	if strings.Contains(cmd, "--print") {
		t.Error("interactive resume should NOT contain --print")
	}
}

func TestBuildClaudeCmd_Headless_Continue(t *testing.T) {
	t.Parallel()
	cfg := &SpawnConfig{
		Mode:     store.ModeHeadless,
		MaxTurns: 50,
		Continue: true,
	}
	cmd := BuildClaudeCmd(cfg)

	if !strings.Contains(cmd, "--continue") {
		t.Errorf("should contain --continue: got %q", cmd)
	}
	if !strings.Contains(cmd, "--max-turns 50") {
		t.Errorf("should contain --max-turns 50: got %q", cmd)
	}
	if strings.Contains(cmd, "--resume") {
		t.Errorf("continue mode should NOT contain --resume: got %q", cmd)
	}
	if strings.Contains(cmd, "< ") {
		t.Errorf("continue mode should NOT redirect stdin: got %q", cmd)
	}
}

func TestBuildClaudeCmd_Interactive_Continue(t *testing.T) {
	t.Parallel()
	cfg := &SpawnConfig{
		Mode:     store.ModeInteractive,
		MaxTurns: 30,
		Continue: true,
	}
	cmd := BuildClaudeCmd(cfg)

	if !strings.Contains(cmd, "--continue") {
		t.Errorf("should contain --continue: got %q", cmd)
	}
	if !strings.Contains(cmd, "--max-turns 30") {
		t.Errorf("should contain --max-turns 30: got %q", cmd)
	}
	if strings.Contains(cmd, "--print") {
		t.Error("interactive continue should NOT contain --print")
	}
	if strings.Contains(cmd, "-p ") {
		t.Error("interactive continue should NOT contain -p (prompt)")
	}
}

func TestBuildClaudeCmd_Continue_TakesPrecedence(t *testing.T) {
	t.Parallel()
	cfg := &SpawnConfig{
		Mode:      store.ModeHeadless,
		MaxTurns:  50,
		SessionID: "should-be-ignored",
		PlanPath:  "/tmp/plan.md",
		Continue:  true,
	}
	cmd := BuildClaudeCmd(cfg)

	if !strings.Contains(cmd, "--continue") {
		t.Errorf("continue should take precedence: got %q", cmd)
	}
	if strings.Contains(cmd, "--resume") {
		t.Errorf("should NOT contain --resume when Continue is set: got %q", cmd)
	}
}

func TestBuildClaudeCmd_CustomMaxTurns(t *testing.T) {
	t.Parallel()
	cfg := &SpawnConfig{
		Mode:     store.ModeHeadless,
		MaxTurns: 10,
		PlanPath: "/tmp/plan.md",
	}
	cmd := BuildClaudeCmd(cfg)

	if !strings.Contains(cmd, "--max-turns 10") {
		t.Errorf("should contain --max-turns 10, got %q", cmd)
	}
}

func TestBuildClaudeCmd_ZeroMaxTurns(t *testing.T) {
	t.Parallel()
	cfg := &SpawnConfig{
		Mode:     store.ModeHeadless,
		MaxTurns: 0,
		PlanPath: "/tmp/plan.md",
	}
	cmd := BuildClaudeCmd(cfg)

	if !strings.Contains(cmd, "--max-turns 50") {
		t.Errorf("zero max_turns should default to 50, got %q", cmd)
	}
}

var runIDPattern = regexp.MustCompile(`^[a-zA-Z0-9-]+-\d{8}-\d{6}-[0-9a-f]{8}$`)

func TestGenerateRunID_Formats(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		want string // substring expected in the name portion
	}{
		{"my-feature", "my-feature-"},
		{"", "run-"},
	}

	for _, tt := range tests {
		id := GenerateRunID(tt.name)
		if !strings.Contains(id, tt.want) {
			t.Errorf("GenerateRunID(%q) = %q, want to contain %q", tt.name, id, tt.want)
		}
		if !runIDPattern.MatchString(id) {
			t.Errorf("GenerateRunID(%q) = %q, doesn't match expected pattern", tt.name, id)
		}
	}
}

func TestGenerateRunID_Sanitization(t *testing.T) {
	t.Parallel()
	id := GenerateRunID("hello world!@#$%")
	if strings.ContainsAny(id, " !@#$%") {
		t.Errorf("ID contains special chars: %q", id)
	}
}

func TestGenerateRunID_LongName(t *testing.T) {
	t.Parallel()
	longName := strings.Repeat("a", 100)
	id := GenerateRunID(longName)

	// Name portion should be truncated to 30
	parts := strings.SplitN(id, "-20", 2) // split at the timestamp
	if len(parts[0]) > 30 {
		t.Errorf("name portion too long: %d chars in %q", len(parts[0]), id)
	}
}

func TestGenerateRunID_Uniqueness(t *testing.T) {
	t.Parallel()
	id1 := GenerateRunID("test")
	id2 := GenerateRunID("test")
	if id1 == id2 {
		t.Errorf("two calls produced the same ID: %q", id1)
	}
}

func TestRegistry_AddRemoveGet(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()

	if reg.has("test") {
		t.Error("empty registry should not have 'test'")
	}

	if _, ok := reg.get("test"); ok {
		t.Error("get on empty registry should return false")
	}

	// We can't easily create a real *os.Process without spawning,
	// but we can test the map operations with nil (they're pointer operations).
	reg.add("test", nil)

	if !reg.has("test") {
		t.Error("should have 'test' after add")
	}

	_, ok := reg.get("test")
	if !ok {
		t.Error("get should return true after add")
	}

	reg.remove("test")

	if reg.has("test") {
		t.Error("should not have 'test' after remove")
	}
}

func TestComputePathHash(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"/home/user/project", "-home-user-project"},
		{"/tmp/test", "-tmp-test"},
	}

	for _, tt := range tests {
		got := ComputePathHash(tt.input)
		if got != tt.want {
			t.Errorf("ComputePathHash(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFindForkedSession(t *testing.T) {
	// Cannot use t.Parallel with t.Setenv.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	repoDir := "/tmp/test-repo"
	pathHash := ComputePathHash(repoDir)
	projectDir := filepath.Join(tmpHome, ".claude", "projects", pathHash)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	originalID := "original-session"
	forkedID := "forked-session"

	// Write original session file.
	if err := os.WriteFile(filepath.Join(projectDir, originalID+".jsonl"), []byte("original"), 0o644); err != nil {
		t.Fatalf("write original: %v", err)
	}

	// Write forked session file (created after original).
	time.Sleep(10 * time.Millisecond) // ensure different mtime
	if err := os.WriteFile(filepath.Join(projectDir, forkedID+".jsonl"), []byte("forked"), 0o644); err != nil {
		t.Fatalf("write forked: %v", err)
	}

	got, err := FindForkedSession(repoDir, originalID)
	if err != nil {
		t.Fatalf("FindForkedSession: %v", err)
	}
	if got != forkedID {
		t.Errorf("got %q, want %q", got, forkedID)
	}
}

func TestFindForkedSession_NotFound(t *testing.T) {
	// Cannot use t.Parallel with t.Setenv.
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	repoDir := "/tmp/no-fork-repo"
	pathHash := ComputePathHash(repoDir)
	projectDir := filepath.Join(tmpHome, ".claude", "projects", pathHash)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Only write the original session file.
	if err := os.WriteFile(filepath.Join(projectDir, "only-session.jsonl"), []byte("data"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := FindForkedSession(repoDir, "only-session")
	if err == nil {
		t.Fatal("expected error when no forked session exists")
	}
}
