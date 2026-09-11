// Package matching provides lightweight, deterministic explanations for why
// a semantic (embedding-based) match was suggested - a cheap complement to
// the AI similarity score, not a replacement for it. No AI model call
// involved: just keyword overlap against the candidate's own declared
// Skills field.
package matching

import "strings"

// OverlappingSkills returns which of the comma-separated skills in
// skillsCSV appear (case-insensitively, as a substring) in text, in the
// order they appear in skillsCSV. Used to show why a match was suggested -
// e.g. "Matched on: Recruiting, Benefits" - without another AI call.
func OverlappingSkills(skillsCSV, text string) []string {
	if skillsCSV == "" || text == "" {
		return nil
	}
	lowerText := strings.ToLower(text)

	var found []string
	for _, skill := range strings.Split(skillsCSV, ",") {
		skill = strings.TrimSpace(skill)
		if skill == "" {
			continue
		}
		if strings.Contains(lowerText, strings.ToLower(skill)) {
			found = append(found, skill)
		}
	}
	return found
}
