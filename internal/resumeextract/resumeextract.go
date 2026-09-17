// Package resumeextract pulls readable text out of uploaded PDF and DOCX
// resumes and renders it as Quill-compatible HTML (paragraphs plus
// bullet/numbered lists — no bold/italic). Callers are responsible for
// running the result through sanitize.SanitizeRichText before persisting or
// returning it; this package only builds HTML, it doesn't sanitize it.
package resumeextract

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
)

// Format identifies which extractor a file should be routed through.
type Format int

const (
	FormatUnknown Format = iota
	FormatPDF
	FormatDOCX
)

// DetectFormat validates filename's extension against the supported set
// (.pdf, .docx only — legacy .doc is unsupported since the DOCX extractor
// only reads the OOXML zip format) and cross-checks the file's magic bytes
// against that extension, so a mislabeled or corrupted upload fails with a
// clear error before it ever reaches the parsing libraries.
func DetectFormat(filename string, data []byte) (Format, error) {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".pdf":
		if !bytes.HasPrefix(data, []byte("%PDF-")) {
			return FormatUnknown, fmt.Errorf("%w: that file doesn't look like a valid PDF", ErrUnsupportedFormat)
		}
		return FormatPDF, nil
	case ".docx":
		if !bytes.HasPrefix(data, []byte("PK\x03\x04")) {
			return FormatUnknown, fmt.Errorf("%w: that file doesn't look like a valid DOCX", ErrUnsupportedFormat)
		}
		return FormatDOCX, nil
	case ".doc":
		return FormatUnknown, fmt.Errorf("%w: legacy .doc files aren't supported — please save as .docx or .pdf", ErrUnsupportedFormat)
	default:
		return FormatUnknown, fmt.Errorf("%w: only PDF and DOCX files are supported", ErrUnsupportedFormat)
	}
}

// ExtractHTML dispatches to the PDF or DOCX extractor for the given format.
func ExtractHTML(format Format, data []byte) (string, error) {
	switch format {
	case FormatPDF:
		return extractPDFHTML(data)
	case FormatDOCX:
		return extractDOCXHTML(data)
	default:
		return "", ErrUnsupportedFormat
	}
}
