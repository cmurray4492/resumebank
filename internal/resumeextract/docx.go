package resumeextract

import (
	"fmt"
	"strings"

	"github.com/xavier268/mydocx"
)

// docxDocumentXML is the key mydocx.ExtractTextBytes uses in its result map
// for the main document body (as opposed to headers/footers).
const docxDocumentXML = "word/document.xml"

// extractDOCXHTML reads paragraph text from a DOCX via mydocx, then
// separately scans the same document.xml for Word-native list numbering
// (<w:numPr>) to flag which paragraphs are list items — mydocx itself has
// no list-detection API, it only returns flat paragraph strings.
func extractDOCXHTML(data []byte) (string, error) {
	result, err := mydocx.ExtractTextBytes(data)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCorruptFile, err)
	}
	paragraphs, ok := result[docxDocumentXML]
	if !ok {
		return "", fmt.Errorf("%w: no document body found", ErrCorruptFile)
	}

	listFlags, err := docxParagraphIsListItem(data)
	if err != nil || len(listFlags) != len(paragraphs) {
		// Degrade gracefully rather than risk misaligning list flags with
		// the wrong paragraph text: treat everything as plain text and
		// fall back to the leading-character heuristic below.
		listFlags = make([]bool, len(paragraphs))
	}

	lines := make([]extractedLine, 0, len(paragraphs))
	for i, para := range paragraphs {
		if strings.TrimSpace(para) == "" {
			continue // Word spacer paragraphs
		}
		if listFlags[i] {
			// Presence-only detection: we don't resolve numbering.xml, so a
			// Word-native numbered list can't be told apart from a bulleted
			// one — every numPr-flagged paragraph renders as a bullet.
			lines = append(lines, extractedLine{text: para, kind: kindBullet})
			continue
		}
		lines = append(lines, classifyLine(para))
	}

	out := buildHTML(lines)
	if out == "" {
		return "", ErrNoTextFound
	}
	return out, nil
}
