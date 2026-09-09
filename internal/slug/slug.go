// Package slug generates URL-safe, SEO-friendly slugs.
package slug

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)
	trimDashes  = regexp.MustCompile(`^-+|-+$`)
)

// Make converts input into a lowercase, dash-separated slug.
func Make(input string) string {
	s := strings.ToLower(strings.TrimSpace(input))
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = trimDashes.ReplaceAllString(s, "")
	if s == "" {
		s = "item"
	}
	return s
}

// WithSuffix appends a numeric suffix, e.g. Make("Jane Doe") + WithSuffix(base, 2) => "jane-doe-2".
func WithSuffix(base string, n int) string {
	return fmt.Sprintf("%s-%d", base, n)
}
