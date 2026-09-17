package resumeextract

import (
	"html"
	"regexp"
	"strings"
)

var (
	bulletRE  = regexp.MustCompile(`^[•\-*‣◦]\s+`)
	orderedRE = regexp.MustCompile(`^\d+[.)]\s+`)
)

type lineKind int

const (
	kindParagraph lineKind = iota
	kindBullet
	kindOrdered
)

type extractedLine struct {
	text string
	kind lineKind
}

// classifyLine strips a literal leading bullet/numbering marker (e.g. "- ",
// "1. ") from text and reports what kind of line it looks like. This is the
// only list signal available for PDF text (which has no structure beyond
// characters) and doubles as a fallback for DOCX paragraphs that were typed
// as literal list markers rather than styled with Word's numbering feature.
func classifyLine(text string) extractedLine {
	if loc := bulletRE.FindStringIndex(text); loc != nil {
		return extractedLine{text: strings.TrimSpace(text[loc[1]:]), kind: kindBullet}
	}
	if loc := orderedRE.FindStringIndex(text); loc != nil {
		return extractedLine{text: strings.TrimSpace(text[loc[1]:]), kind: kindOrdered}
	}
	return extractedLine{text: text, kind: kindParagraph}
}

// buildHTML groups consecutive list lines of the same kind into one <ol>
// block — Quill represents both bulleted and numbered lists as <ol> with a
// data-list attribute, never <ul> — and wraps everything else in <p>. Blank
// lines are dropped rather than treated as paragraph/section breaks, since
// neither extractor reliably produces blank lines at real paragraph
// boundaries.
func buildHTML(lines []extractedLine) string {
	var b strings.Builder
	openKind := kindParagraph
	listOpen := false

	closeList := func() {
		if listOpen {
			b.WriteString("</ol>")
			listOpen = false
		}
	}

	for _, line := range lines {
		text := strings.TrimSpace(line.text)
		if text == "" {
			continue
		}
		escaped := html.EscapeString(text)

		if line.kind == kindBullet || line.kind == kindOrdered {
			if !listOpen || openKind != line.kind {
				closeList()
				b.WriteString(`<ol data-list="` + listDataAttr(line.kind) + `">`)
				listOpen = true
				openKind = line.kind
			}
			b.WriteString(`<li data-list="` + listDataAttr(line.kind) + `">` + escaped + `</li>`)
			continue
		}

		closeList()
		b.WriteString("<p>" + escaped + "</p>")
	}
	closeList()
	return b.String()
}

func listDataAttr(k lineKind) string {
	if k == kindOrdered {
		return "ordered"
	}
	return "bullet"
}
