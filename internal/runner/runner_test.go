package runner

import (
	"regexp"
	"strings"
	"testing"

	"github.com/wesgrimes/outpost/internal/store"
)

func TestBuildClaudeCmd_Headless(t *testing.T) {
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
		t.Error("headless should NOT contain --resume")
	}
	if !strings.Contains(cmd, `< "/tmp/plan.md"`) {
		t.Errorf("should redirect plan: got %q", cmd)
	}
}

func TestBuildClaudeCmd_Interactive(t *testing.T) {
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
