package validate

import "testing"

func TestRequired(t *testing.T) {
	errs := FieldErrors{}
	Required("", "name", errs)
	if !errs.HasErrors() {
		t.Error("expected error for empty required field")
	}

	errs = FieldErrors{}
	Required("Jane", "name", errs)
	if errs.HasErrors() {
		t.Error("expected no error for non-empty field")
	}
}

func TestEmail(t *testing.T) {
	cases := []struct {
		in    string
		valid bool
	}{
		{"jane@example.com", true},
		{"not-an-email", false},
		{"", true}, // Email() only validates non-empty; pair with Required for mandatory fields
	}
	for _, c := range cases {
		errs := FieldErrors{}
		Email(c.in, "email", errs)
		if got := !errs.HasErrors(); got != c.valid {
			t.Errorf("Email(%q): valid=%v, want %v", c.in, got, c.valid)
		}
	}
}

func TestZipcode(t *testing.T) {
	cases := []struct {
		in    string
		valid bool
	}{
		{"12345", true},
		{"12345-6789", true},
		{"abcde", false},
		{"123", false},
	}
	for _, c := range cases {
		errs := FieldErrors{}
		Zipcode(c.in, "zipcode", errs)
		if got := !errs.HasErrors(); got != c.valid {
			t.Errorf("Zipcode(%q): valid=%v, want %v", c.in, got, c.valid)
		}
	}
}

func TestFieldErrorsAddDoesNotOverwrite(t *testing.T) {
	errs := FieldErrors{}
	errs.Add("email", "first error")
	errs.Add("email", "second error")
	if errs["email"] != "first error" {
		t.Errorf("expected first error to win, got %q", errs["email"])
	}
}
