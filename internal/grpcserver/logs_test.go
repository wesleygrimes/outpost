package grpcserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadLogTail(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "output.log")

	lines := make([]string, 0, 100)
	for i := range 100 {
		lines = append(lines, "line "+string(rune('0'+i/10))+string(rune('0'+i%10)))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := readLogTail(path)
	gotLines := strings.Split(got, "\n")
	if len(gotLines) != 80 {
		t.Errorf("got %d lines, want 80", len(gotLines))
	}
}

func TestReadLogTail_FewerLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "output.log")

	if err := os.WriteFile(path, []byte("line1\nline2\nline3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := readLogTail(path)
	gotLines := strings.Split(got, "\n")
	if len(gotLines) != 3 {
		t.Errorf("got %d lines, want 3", len(gotLines))
	}
}

func TestReadLogTail_MissingFile(t *testing.T) {
	t.Parallel()
	got := readLogTail("/nonexistent/output.log")
	if got != "" {
		t.Errorf("got %q, want empty string for missing file", got)
	}
}

func TestReadLogTail_EmptyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "output.log")

	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	got := readLogTail(path)
	if got != "" {
		t.Errorf("got %q, want empty string for empty file", got)
	}
}
