// Package validate provides small, dependency-free field validators.
package validate

import (
	"net/url"
	"regexp"
	"strings"
)

// FieldErrors maps a field name to its error message, for re-rendering
// forms with inline validation feedback.
type FieldErrors map[string]string

func (fe FieldErrors) Add(field, message string) {
	if _, exists := fe[field]; !exists {
		fe[field] = message
	}
}

func (fe FieldErrors) HasErrors() bool {
	return len(fe) > 0
}

var (
	emailRE = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	zipRE   = regexp.MustCompile(`^\d{5}(-\d{4})?$`)
)

func Required(value, field string, errs FieldErrors) {
	if strings.TrimSpace(value) == "" {
		errs.Add(field, "This field is required.")
	}
}

func MaxLen(value, field string, max int, errs FieldErrors) {
	if len(value) > max {
		errs.Add(field, "This field is too long.")
	}
}

func Email(value, field string, errs FieldErrors) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if !emailRE.MatchString(value) {
		errs.Add(field, "Enter a valid email address.")
	}
}

func Zipcode(value, field string, errs FieldErrors) {
	if strings.TrimSpace(value) == "" {
		return
	}
	if !zipRE.MatchString(value) {
		errs.Add(field, "Enter a valid US zip code.")
	}
}

func URL(value, field string, errs FieldErrors) {
	if strings.TrimSpace(value) == "" {
		return
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		errs.Add(field, "Enter a valid URL starting with http:// or https://.")
	}
}
