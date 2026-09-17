package resumeextract

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// buildTestPDF assembles a minimal, well-formed single-page PDF (standard
// xref table, one Helvetica text run per line) so extraction tests exercise
// the real pdf.Reader parsing path rather than a canned fixture.
func buildTestPDF(t *testing.T, lines []string) []byte {
	t.Helper()

	var content strings.Builder
	y := 750
	escaper := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	for _, line := range lines {
		fmt.Fprintf(&content, "BT /F1 12 Tf 72 %d Td (%s) Tj ET\n", y, escaper.Replace(line))
		y -= 18
	}
	contentBytes := []byte(content.String())

	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /Resources << /Font << /F1 4 0 R >> >> /MediaBox [0 0 612 792] /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	var offsets []int
	for i, o := range objs {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}

	offsets = append(offsets, buf.Len())
	fmt.Fprintf(&buf, "5 0 obj\n<< /Length %d >>\nstream\n", len(contentBytes))
	buf.Write(contentBytes)
	buf.WriteString("endstream\nendobj\n")

	xrefOffset := buf.Len()
	total := len(offsets) + 1 // + the free-list head entry
	fmt.Fprintf(&buf, "xref\n0 %d\n", total)
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", total, xrefOffset)

	return buf.Bytes()
}

const docxNamespace = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"

type docxPara struct {
	text     string
	numbered bool // wraps the run in <w:pPr><w:numPr>...</w:numPr></w:pPr>
}

// buildTestDOCX assembles a minimal, valid .docx zip containing only
// word/document.xml (the only part mydocx and docxParagraphIsListItem read).
func buildTestDOCX(t *testing.T, paras []docxPara) []byte {
	t.Helper()

	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	fmt.Fprintf(&body, `<w:document xmlns:w="%s"><w:body>`, docxNamespace)
	for _, p := range paras {
		body.WriteString("<w:p>")
		if p.numbered {
			body.WriteString(`<w:pPr><w:numPr><w:ilvl w:val="0"/><w:numId w:val="1"/></w:numPr></w:pPr>`)
		}
		body.WriteString("<w:r><w:t>")
		body.WriteString(strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(p.text))
		body.WriteString("</w:t></w:r></w:p>")
	}
	body.WriteString("</w:body></w:document>")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(docxDocumentXML)
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := w.Write([]byte(body.String())); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func TestExtractPDFHTML_ParagraphsAndLists(t *testing.T) {
	data := buildTestPDF(t, []string{
		"Jane Doe",
		"Software Engineer",
		"- Built things",
		"- Fixed bugs",
		"1. Led a project",
		"2. Shipped it",
	})

	html, err := extractPDFHTML(data)
	if err != nil {
		t.Fatalf("extractPDFHTML: %v", err)
	}
	t.Logf("html: %s", html)

	for _, want := range []string{
		"<p>Jane Doe</p>",
		"<p>Software Engineer</p>",
		`<ol data-list="bullet">`,
		`<li data-list="bullet">Built things</li>`,
		`<li data-list="bullet">Fixed bugs</li>`,
		`<ol data-list="ordered">`,
		`<li data-list="ordered">Led a project</li>`,
		`<li data-list="ordered">Shipped it</li>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected html to contain %q, got: %s", want, html)
		}
	}
}

func TestExtractPDFHTML_CorruptFile(t *testing.T) {
	_, err := extractPDFHTML([]byte("%PDF-1.4\nnot really a pdf"))
	if err == nil {
		t.Fatal("expected an error for a corrupt PDF, got nil")
	}
}

func TestExtractPDFHTML_NoText(t *testing.T) {
	data := buildTestPDF(t, nil)
	_, err := extractPDFHTML(data)
	if err != ErrNoTextFound {
		t.Fatalf("expected ErrNoTextFound, got %v", err)
	}
}

func TestExtractDOCXHTML_ParagraphsAndLists(t *testing.T) {
	data := buildTestDOCX(t, []docxPara{
		{text: "Jane Doe"},
		{text: "Software Engineer"},
		{text: "Built things", numbered: true},
		{text: "Fixed bugs", numbered: true},
		{text: "1. Typed as literal text"},
		{text: "   "}, // blank spacer paragraph, should be skipped
	})

	html, err := extractDOCXHTML(data)
	if err != nil {
		t.Fatalf("extractDOCXHTML: %v", err)
	}
	t.Logf("html: %s", html)

	for _, want := range []string{
		"<p>Jane Doe</p>",
		"<p>Software Engineer</p>",
		`<ol data-list="bullet">`,
		`<li data-list="bullet">Built things</li>`,
		`<li data-list="bullet">Fixed bugs</li>`,
		`<li data-list="ordered">Typed as literal text</li>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("expected html to contain %q, got: %s", want, html)
		}
	}
	if strings.Count(html, "<p></p>") != 0 {
		t.Errorf("blank spacer paragraph should have been skipped, got: %s", html)
	}
}

func TestExtractDOCXHTML_CorruptFile(t *testing.T) {
	_, err := extractDOCXHTML([]byte("not a zip"))
	if err == nil {
		t.Fatal("expected an error for a corrupt DOCX, got nil")
	}
}

func TestExtractDOCXHTML_NoText(t *testing.T) {
	data := buildTestDOCX(t, nil)
	_, err := extractDOCXHTML(data)
	if err != ErrNoTextFound {
		t.Fatalf("expected ErrNoTextFound, got %v", err)
	}
}

func TestDetectFormat(t *testing.T) {
	pdfData := buildTestPDF(t, []string{"hi"})
	docxData := buildTestDOCX(t, []docxPara{{text: "hi"}})

	cases := []struct {
		name     string
		filename string
		data     []byte
		want     Format
		wantErr  bool
	}{
		{"valid pdf", "resume.pdf", pdfData, FormatPDF, false},
		{"valid docx", "resume.docx", docxData, FormatDOCX, false},
		{"mislabeled pdf", "resume.pdf", []byte("not a pdf"), FormatUnknown, true},
		{"mislabeled docx", "resume.docx", []byte("not a docx"), FormatUnknown, true},
		{"legacy doc", "resume.doc", []byte("whatever"), FormatUnknown, true},
		{"unsupported ext", "resume.txt", []byte("whatever"), FormatUnknown, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := DetectFormat(tc.filename, tc.data)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("format = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExtractHTML_Dispatch(t *testing.T) {
	if _, err := ExtractHTML(FormatUnknown, nil); err != ErrUnsupportedFormat {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}
