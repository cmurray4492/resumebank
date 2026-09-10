// Package sanitize defines the single HTML-sanitization boundary for
// user-submitted rich text (Quill editor output). Handlers must pass
// resume/job-description HTML through SanitizeRichText before it ever
// reaches the repo layer; nothing downstream re-sanitizes it.
package sanitize

import (
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var policy = newPolicy()

func newPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	p.AllowStandardURLs()
	p.RequireNoFollowOnLinks(true)
	p.AllowElements("p", "br", "hr", "strong", "em", "u", "s", "strike",
		"blockquote", "pre", "code", "sub", "sup",
		"ol", "ul", "li", "h1", "h2", "h3", "h4", "h5", "h6", "span", "div", "a")
	p.AllowAttrs("href").OnElements("a")
	// Quill renders bullet AND ordered lists as <ol><li data-list="...">,
	// distinguishing them only via this attribute (no <ul> is ever emitted) -
	// stripping it silently turns every bullet list into a numbered one,
	// since a bare <li> inside <ol> falls back to default browser numbering.
	// See the .rich-text rules in web/static/css/site.css, which render the
	// actual marker (bullet/number/checkbox) from this attribute.
	p.AllowAttrs("data-list").Matching(regexp.MustCompile(`^(bullet|ordered|checked|unchecked)$`)).OnElements("li")
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^ql-[a-zA-Z0-9-]+$`)).Globally()
	p.AllowAttrs("style").Matching(regexp.MustCompile(`^[a-zA-Z-]+:\s*[a-zA-Z0-9%.\s-]+;?$`)).Globally()

	return p
}

// SanitizeRichText strips scripts, event handlers, and non-http(s) URLs from
// Quill-produced HTML while preserving its formatting tags and ql-* classes.
func SanitizeRichText(html string) string {
	return policy.Sanitize(html)
}

var tagRE = regexp.MustCompile(`<[^>]*>`)
var spaceRE = regexp.MustCompile(`\s+`)

// PlainText strips all tags from sanitized HTML, for storage in the
// search-only *_text columns.
func PlainText(html string) string {
	stripped := tagRE.ReplaceAllString(html, " ")
	return strings.TrimSpace(spaceRE.ReplaceAllString(stripped, " "))
}
