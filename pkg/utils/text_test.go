package utils

import "testing"

func TestHasText(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "empty", value: "", want: false},
		{name: "spaces only", value: " \t\n ", want: false},
		{name: "text with spaces", value: "  admin  ", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasText(tt.value); got != tt.want {
				t.Fatalf("HasText(%q) = %t, want %t", tt.value, got, tt.want)
			}
		})
	}
}

func TestTrimLower(t *testing.T) {
	got := TrimLower("  Top-Admin  ")
	if got != "top-admin" {
		t.Fatalf("TrimLower() = %q, want %q", got, "top-admin")
	}
}
