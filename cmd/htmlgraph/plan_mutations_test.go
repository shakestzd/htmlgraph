package main

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Error message length?", "error-message-length"},
		{"Simple", "simple"},
		{"Multiple   Spaces   Here", "multiple-spaces-here"},
		{"", "untitled"},
		{"UPPER CASE", "upper-case"},
		{"special!@#chars", "special-chars"},
	}
	for _, tc := range cases {
		got := slugify(tc.input)
		if got != tc.want {
			t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSlugifyTruncation(t *testing.T) {
	long := "this is a really long title that should be truncated at a word boundary properly"
	got := slugify(long)
	if len(got) > 40 {
		t.Errorf("slugify(long) = %q (len %d), want <= 40 chars", got, len(got))
	}
}
