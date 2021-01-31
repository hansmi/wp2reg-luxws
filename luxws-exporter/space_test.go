package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNormalizeSpace(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{
			input: "",
			want:  "",
		},
		{
			input: "    ",
			want:  "",
		},
		{
			input: "\t\n\t",
			want:  "",
		},
		{
			input: "foobar",
			want:  "foobar",
		},
		{
			input: " -   foo   -   bar   - ",
			want:  "- foo - bar -",
		},
	} {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeSpace(tc.input)

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("normalizeSpace(%q) difference (-want +got):\n%s", tc.input, diff)
			}
		})
	}
}
