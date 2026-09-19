package resumeextract

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

type pdfSeg struct {
	text string
	font string // F1 regular, F2 bold, F3 oblique
	size float64
	x, y float64
}

// buildStyledTestPDF is like buildTestPDF but places each segment with an
// absolute position, font and size, and gives the fonts explicit /Widths
// (the pdf library only advances X per glyph when widths are present, as they
// are in real word-processor output).
func buildStyledTestPDF(t *testing.T, segs []pdfSeg) []byte {
	t.Helper()

	var content strings.Builder
	escaper := strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`)
	for _, s := range segs {
		fmt.Fprintf(&content, "BT /%s %g Tf 1 0 0 1 %g %g Tm (%s) Tj ET\n", s.font, s.size, s.x, s.y, escaper.Replace(s.text))
	}
	contentBytes := []byte(content.String())

	widths := "/FirstChar 32 /LastChar 126 /Widths [" + strings.Repeat("500 ", 95) + "]"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /Resources << /Font << /F1 4 0 R /F2 5 0 R /F3 6 0 R >> >> /MediaBox [0 0 612 792] /Contents 7 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica " + widths + " >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold " + widths + " >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Oblique " + widths + " >>",
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	var offsets []int
	for i, o := range objs {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	offsets = append(offsets, buf.Len())
	fmt.Fprintf(&buf, "7 0 obj\n<< /Length %d >>\nstream\n", len(contentBytes))
	buf.Write(contentBytes)
	buf.WriteString("endstream\nendobj\n")

	xrefOffset := buf.Len()
	total := len(offsets) + 1
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", total)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", total, xrefOffset)
	return buf.Bytes()
}

func TestExtractPDFHTML_PreservesFormatting(t *testing.T) {
	longLine := strings.TrimSpace(strings.Repeat("abcd ", 16)) // 79 chars ≈ full width
	data := buildStyledTestPDF(t, []pdfSeg{
		{"Jane Doe", "F2", 24, 72, 740},
		{"Software Engineer", "F1", 12, 72, 710},
		{"jane@example.com", "F1", 12, 72, 694},
		{"EXPERIENCE", "F2", 12, 72, 660},
		{"Senior Engineer", "F2", 12, 72, 640},
		{" at Acme, ", "F1", 12, 162, 640},
		{"Remote", "F3", 12, 222, 640},
		{longLine, "F1", 12, 72, 620},
		{"and then continues here.", "F1", 12, 72, 606},
		{"- Built the billing system that", "F1", 12, 72, 580},
		{"handles refunds", "F1", 12, 90, 566},
		{"- Fixed bugs", "F1", 12, 72, 552},
		{"Hello", "F1", 12, 72, 520},
		{"World", "F1", 12, 106, 520}, // 4pt gap, no space glyph
	})

	out, err := extractPDFHTML(data)
	if err != nil {
		t.Fatalf("extractPDFHTML: %v", err)
	}
	t.Logf("html: %s", out)

	for name, want := range map[string]string{
		"name is h1":                  "<h1>Jane Doe</h1>",
		"caps bold header is h2":      "<h2>EXPERIENCE</h2>",
		"short lines stay separate":   "<p>Software Engineer</p><p>jane@example.com</p>",
		"bold and italic runs":        "<p><strong>Senior Engineer</strong> at Acme, <em>Remote</em></p>",
		"wrapped paragraph is joined": "abcd and then continues here.</p>",
		"wrapped bullet is joined":    `<li data-list="bullet">Built the billing system that handles refunds</li>`,
		"next bullet stays separate":  `<li data-list="bullet">Fixed bugs</li>`,
		"positional gap is a space":   "<p>Hello World</p>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("%s: expected html to contain %q", name, want)
		}
	}
}
