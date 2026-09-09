package sanitize

import (
	"strings"
	"testing"
)

func TestSanitizeRichText_StripsScripts(t *testing.T) {
	out := SanitizeRichText(`<p>Hello</p><script>alert('xss')</script>`)
	if strings.Contains(out, "<script") {
		t.Errorf("expected script tag to be stripped, got: %s", out)
	}
	if !strings.Contains(out, "<p>Hello</p>") {
		t.Errorf("expected paragraph to be preserved, got: %s", out)
	}
}

func TestSanitizeRichText_StripsEventHandlers(t *testing.T) {
	out := SanitizeRichText(`<p onclick="alert('xss')">Hi</p>`)
	if strings.Contains(out, "onclick") {
		t.Errorf("expected onclick attribute to be stripped, got: %s", out)
	}
}

func TestSanitizeRichText_StripsJavascriptHref(t *testing.T) {
	out := SanitizeRichText(`<a href="javascript:alert(1)">click</a>`)
	if strings.Contains(out, "javascript:") {
		t.Errorf("expected javascript: URL to be stripped, got: %s", out)
	}
}

func TestSanitizeRichText_KeepsQuillClasses(t *testing.T) {
	out := SanitizeRichText(`<p class="ql-align-center">Centered</p>`)
	if !strings.Contains(out, "ql-align-center") {
		t.Errorf("expected ql-* class to be preserved, got: %s", out)
	}
}

func TestPlainText_StripsTags(t *testing.T) {
	out := PlainText(`<p>Hello <strong>World</strong></p>`)
	if out != "Hello World" {
		t.Errorf("expected 'Hello World', got: %q", out)
	}
}
