package slug

import "testing"

func TestMake(t *testing.T) {
	cases := map[string]string{
		"Jane Doe":        "jane-doe",
		"  Trim Me  ":     "trim-me",
		"Acme, Inc.":      "acme-inc",
		"Already-Slugged": "already-slugged",
		"":                "item",
		"!!!":             "item",
	}
	for in, want := range cases {
		if got := Make(in); got != want {
			t.Errorf("Make(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWithSuffix(t *testing.T) {
	if got := WithSuffix("jane-doe", 2); got != "jane-doe-2" {
		t.Errorf("WithSuffix = %q, want jane-doe-2", got)
	}
}
