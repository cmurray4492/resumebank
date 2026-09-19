package resumeextract

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/jh125486/pdf"
)

// extractPDFHTML reads plain text from a PDF and renders it as HTML.
//
// pdf.Reader.GetPlainText inserts one "\n" per text-positioning operator in
// the PDF's content stream (BT/T*), which in practice is roughly one "\n"
// per visual line already — not per wrapped sub-line the way a plain-text
// word-wrap would. So unlike a word-wrapped text file, lines here are NOT
// joined back into paragraphs: each non-blank line becomes its own <p> (or
// list item), since merging would frequently glue unrelated resume lines
// (name, title, section headers, job titles) into one garbled paragraph.
func extractPDFHTML(data []byte) (string, error) {
	r, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCorruptFile, err)
	}

	// Preferred path: rebuild headings, bold/italic and wrapped paragraphs
	// from glyph fonts/sizes/positions. Falls back to plain text below if
	// the content stream can't be read that way or yields nothing.
	if lines, err := extractPDFStyledLines(r); err == nil && len(lines) > 0 {
		if out := buildStyledHTML(lines); out != "" {
			return out, nil
		}
	}

	textReader, err := r.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCorruptFile, err)
	}
	raw, err := io.ReadAll(textReader)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCorruptFile, err)
	}

	var lines []extractedLine
	for _, rawLine := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(rawLine) == "" {
			continue
		}
		lines = append(lines, classifyLine(rawLine))
	}

	out := buildHTML(lines)
	if out == "" {
		return "", ErrNoTextFound
	}
	return out, nil
}
