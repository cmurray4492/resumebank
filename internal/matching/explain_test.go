package matching

import (
	"reflect"
	"testing"
)

func TestOverlappingSkills(t *testing.T) {
	cases := []struct {
		name      string
		skillsCSV string
		text      string
		want      []string
	}{
		{
			name:      "matches case-insensitively and preserves skill order",
			skillsCSV: "Recruiting, Benefits, Compensation",
			text:      "We need someone with strong RECRUITING and benefits administration experience.",
			want:      []string{"Recruiting", "Benefits"},
		},
		{
			name:      "no overlap",
			skillsCSV: "Welding, Forklift Operation",
			text:      "Looking for a software engineer skilled in Go and PostgreSQL.",
			want:      nil,
		},
		{
			name:      "empty skills",
			skillsCSV: "",
			text:      "Anything at all",
			want:      nil,
		},
		{
			name:      "empty text",
			skillsCSV: "Recruiting",
			text:      "",
			want:      nil,
		},
		{
			name:      "ignores blank entries between commas",
			skillsCSV: "Recruiting, , Benefits",
			text:      "recruiting and benefits",
			want:      []string{"Recruiting", "Benefits"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := OverlappingSkills(c.skillsCSV, c.text)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("OverlappingSkills(%q, %q) = %#v, want %#v", c.skillsCSV, c.text, got, c.want)
			}
		})
	}
}
