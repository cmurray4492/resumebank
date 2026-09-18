package render

import "testing"

// New parses every page template eagerly at startup, so a bad template
// (e.g. a variable referenced outside the {{define}} block that declared
// it) fails the whole app's boot, not just requests to that page. This
// needs no DB, so it catches that class of bug in CI/build without a
// Postgres instance.
func TestAllTemplatesParse(t *testing.T) {
	if _, err := New(false, ""); err != nil {
		t.Fatal(err)
	}
}
