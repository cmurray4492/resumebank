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

// TestSanitizeRichText_PreservesListDataAttribute guards against a real bug:
// Quill renders bullet AND ordered lists as <ol><li data-list="...">
// (it never emits <ul>), distinguishing them only via this attribute. If
// it's stripped, a bare <li> inside <ol> falls back to default browser
// numbering, silently turning every bullet list into a numbered one on
// save.
func TestSanitizeRichText_PreservesListDataAttribute(t *testing.T) {
	for _, value := range []string{"bullet", "ordered", "checked", "unchecked"} {
		in := `<ol><li data-list="` + value + `">Item</li></ol>`
		out := SanitizeRichText(in)
		if !strings.Contains(out, `data-list="`+value+`"`) {
			t.Errorf("expected data-list=%q to survive sanitization, got: %s", value, out)
		}
	}
}

func TestSanitizeRichText_RejectsInvalidListDataValue(t *testing.T) {
	out := SanitizeRichText(`<ol><li data-list="javascript:alert(1)">Item</li></ol>`)
	if strings.Contains(out, "data-list") {
		t.Errorf("expected an invalid data-list value to be stripped, got: %s", out)
	}
}

func TestSanitizeRichText_DataListOnlyAllowedOnLi(t *testing.T) {
	out := SanitizeRichText(`<p data-list="bullet">Not a list item</p>`)
	if strings.Contains(out, "data-list") {
		t.Errorf("expected data-list to be stripped on a non-li element, got: %s", out)
	}
}

func TestPlainText_StripsTags(t *testing.T) {
	out := PlainText(`<p>Hello <strong>World</strong></p>`)
	if out != "Hello World" {
		t.Errorf("expected 'Hello World', got: %q", out)
	}
}
