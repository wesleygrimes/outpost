package cmd

import (
	"strings"
	"testing"
)

func TestTruncateBeforeHandoff(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		lines     []string
		wantLines int // number of lines in output
	}{
		{
			name: "removes last human turn",
			lines: []string{
				`{"type":"user","message":{"role":"user","content":"first question"}}`,
				`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"answer"}]}}`,
				`{"type":"user","message":{"role":"user","content":"handoff again"}}`,
				`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Skill"}]}}`,
			},
			wantLines: 2,
		},
		{
			name: "keeps tool results that follow assistant",
			lines: []string{
				`{"type":"user","message":{"role":"user","content":"do something"}}`,
				`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash"}]}}`,
				`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"ok"}]}}`,
				`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"done"}]}}`,
				`{"type":"user","message":{"role":"user","content":"handoff"}}`,
			},
			wantLines: 4,
		},
		{
			name: "no human messages returns original",
			lines: []string{
				`{"type":"file-history-snapshot"}`,
				`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}`,
			},
			wantLines: 2,
		},
		{
			name: "single human message returns original",
			lines: []string{
				`{"type":"user","message":{"role":"user","content":"hello"}}`,
			},
			wantLines: 1,
		},
		{
			name: "keeps file-history-snapshot before handoff message",
			lines: []string{
				`{"type":"user","message":{"role":"user","content":"first"}}`,
				`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"ok"}]}}`,
				`{"type":"file-history-snapshot"}`,
				`{"type":"user","message":{"role":"user","content":"handoff"}}`,
			},
			wantLines: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			input := []byte(strings.Join(tt.lines, "\n") + "\n")
			got := truncateBeforeHandoff(input)
			gotLines := strings.Split(strings.TrimRight(string(got), "\n"), "\n")

			if len(gotLines) != tt.wantLines {
				t.Errorf("got %d lines, want %d\noutput:\n%s", len(gotLines), tt.wantLines, got)
			}
		})
	}
}

func TestIsHumanMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "human message with string content",
			line: `{"type":"user","message":{"role":"user","content":"hello world"}}`,
			want: true,
		},
		{
			name: "tool result with array content",
			line: `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"t1","content":"ok"}]}}`,
			want: false,
		},
		{
			name: "assistant message",
			line: `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hi"}]}}`,
			want: false,
		},
		{
			name: "file-history-snapshot",
			line: `{"type":"file-history-snapshot"}`,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isHumanMessage([]byte(tt.line)); got != tt.want {
				t.Errorf("isHumanMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
