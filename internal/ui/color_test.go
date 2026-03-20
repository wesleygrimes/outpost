package ui

import "testing"

//nolint:paralleltest // tests mutate shared ColorEnabled global
func TestColorEnabled(t *testing.T) {
	orig := ColorEnabled
	defer func() { ColorEnabled = orig }()

	ColorEnabled = true

	tests := []struct {
		name string
		fn   func(string) string
		code string
	}{
		{"Amber", Amber, "\033[33m"},
		{"Green", Green, "\033[32m"},
		{"Cyan", Cyan, "\033[36m"},
		{"Orange", Orange, "\033[93m"},
		{"Red", Red, "\033[31m"},
		{"Purple", Purple, "\033[35m"},
		{"Blue", Blue, "\033[34m"},
		{"White", White, "\033[37m"},
		{"Dim", Dim, "\033[2m"},
		{"Bold", Bold, "\033[1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn("hello")
			want := tt.code + "hello" + "\033[0m"
			if got != want {
				t.Errorf("%s(\"hello\") = %q, want %q", tt.name, got, want)
			}
		})
	}
}

//nolint:paralleltest // tests mutate shared ColorEnabled global
func TestColorDisabled(t *testing.T) {
	ColorEnabled = false
	defer func() { ColorEnabled = true }()

	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"Amber", Amber},
		{"Green", Green},
		{"Cyan", Cyan},
		{"Orange", Orange},
		{"Red", Red},
		{"Purple", Purple},
		{"Blue", Blue},
		{"White", White},
		{"Dim", Dim},
		{"Bold", Bold},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn("hello")
			if got != "hello" {
				t.Errorf("%s(\"hello\") with color off = %q, want %q", tt.name, got, "hello")
			}
		})
	}
}

//nolint:paralleltest // tests mutate shared ColorEnabled global
func TestColorEmptyString(t *testing.T) {
	ColorEnabled = true
	got := Red("")
	want := "\033[31m" + "" + "\033[0m"
	if got != want {
		t.Errorf("Red(\"\") = %q, want %q", got, want)
	}
}
