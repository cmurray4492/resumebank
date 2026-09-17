package resumeextract

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"

	"github.com/xavier268/mydocx"
)

// docxParagraphIsListItem reports, for each <w:p> paragraph in
// word/document.xml, whether it carries Word's native list-numbering markup
// (<w:pPr><w:numPr>). It walks paragraphs in document order exactly like
// mydocx.ExtractTextBytes does (a flat top-to-bottom scan, including
// paragraphs nested inside tables), so the returned slice lines up
// index-for-index with mydocx's paragraph slice.
//
// This is presence-only detection: it doesn't resolve numbering.xml, so it
// can't distinguish a bulleted list from a numbered one — that's left to
// the caller, which treats every flagged paragraph as a bullet.
func docxParagraphIsListItem(docxBytes []byte) ([]bool, error) {
	zr, err := zip.NewReader(bytes.NewReader(docxBytes), int64(len(docxBytes)))
	if err != nil {
		return nil, err
	}

	var docBytes []byte
	for _, f := range zr.File {
		if f.Name != docxDocumentXML {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		docBytes, err = io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		break
	}
	if docBytes == nil {
		return nil, fmt.Errorf("word/document.xml not found")
	}

	var flags []bool
	dec := xml.NewDecoder(bytes.NewReader(docBytes))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		start, ok := tok.(xml.StartElement)
		if !ok || start.Name.Local != "p" || start.Name.Space != mydocx.NAMESPACE {
			continue
		}
		flags = append(flags, paragraphHasNumPr(dec))
	}
	return flags, nil
}

// paragraphHasNumPr consumes tokens up to and including the matching </w:p>
// end element, reporting whether a <w:numPr> descendant was seen. numPr
// only legally appears inside <w:pPr>, so a simple "seen anywhere in this
// paragraph" check is sufficient without tracking pPr nesting separately.
func paragraphHasNumPr(dec *xml.Decoder) bool {
	hasNumPr := false
	for {
		tok, err := dec.Token()
		if err != nil {
			return hasNumPr
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "numPr" && t.Name.Space == mydocx.NAMESPACE {
				hasNumPr = true
			}
		case xml.EndElement:
			if t.Name.Local == "p" && t.Name.Space == mydocx.NAMESPACE {
				return hasNumPr
			}
		}
	}
}
