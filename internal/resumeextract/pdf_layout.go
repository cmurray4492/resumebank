package resumeextract

import (
	"fmt"
	"html"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jh125486/pdf"
)

var spaceRunRE = regexp.MustCompile(`\s+`)

type styledRun struct {
	text         string
	bold, italic bool
}

// layoutLine is one visual line of a PDF page, rebuilt from positioned glyphs.
type layoutLine struct {
	runs       []styledRun
	page       int
	x, y, endX float64
	size       float64
	maxGap     float64
	allBold    bool
}

type blockKind int

const (
	blockPara blockKind = iota
	blockHeading
	blockListItem
)

type block struct {
	kind    blockKind
	level   int
	list    lineKind
	runs    []styledRun
	last    layoutLine
	markerX float64
}

type pageExtent struct{ minX, maxEnd float64 }

// extractPDFStyledLines reads every page's positioned glyphs. Page.Content
// panics on a malformed content stream, so that is recovered into an error.
func extractPDFStyledLines(r *pdf.Reader) (lines []layoutLine, err error) {
	defer func() {
		if e := recover(); e != nil {
			lines, err = nil, fmt.Errorf("malformed PDF: %v", e)
		}
	}()
	for i := 1; i <= r.NumPage(); i++ {
		lines = append(lines, layoutPage(r.Page(i).Content().Text, i)...)
	}
	return lines, nil
}

func layoutPage(texts []pdf.Text, page int) []layoutLine {
	chars := make([]pdf.Text, 0, len(texts))
	for _, t := range texts {
		if t.S == "" {
			continue
		}
		t.FontSize = math.Abs(t.FontSize)
		if t.FontSize == 0 {
			t.FontSize = 10
		}
		chars = append(chars, t)
	}
	sort.SliceStable(chars, func(i, j int) bool { return chars[i].Y > chars[j].Y })

	var lines []layoutLine
	var cluster []pdf.Text
	flush := func() {
		if l, ok := buildLayoutLine(cluster, page); ok {
			lines = append(lines, l)
		}
		cluster = cluster[:0]
	}
	for _, c := range chars {
		if len(cluster) > 0 && cluster[0].Y-c.Y > 0.4*c.FontSize {
			flush()
		}
		cluster = append(cluster, c)
	}
	flush()
	return lines
}

func buildLayoutLine(cluster []pdf.Text, page int) (layoutLine, bool) {
	cs := append([]pdf.Text(nil), cluster...)
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].X < cs[j].X })

	l := layoutLine{page: page}
	sizeCount := map[float64]int{}
	found := false
	var prev *pdf.Text
	var prevStyle styledRun

	for i := range cs {
		c := cs[i]
		isSpace := strings.TrimSpace(c.S) == ""
		style := fontStyle(c.Font)

		// Word processors often emit no space glyph, only a positional gap.
		if prev != nil && !isSpace && strings.TrimSpace(prev.S) != "" && prev.W > 0 {
			gap := c.X - (prev.X + prev.W)
			if gap > 0.17*c.FontSize {
				l.addText(" ", prevStyle)
			}
			if gap > l.maxGap {
				l.maxGap = gap
			}
		}
		l.addText(c.S, style)
		if !isSpace {
			if !found {
				l.x, l.y, found = c.X, c.Y, true
			}
			l.endX = c.X + c.W
			sizeCount[math.Round(c.FontSize*2)/2]++
		}
		prev, prevStyle = &cs[i], style
	}
	if !found {
		return layoutLine{}, false
	}

	best, bestN := 0.0, 0
	for s, n := range sizeCount {
		if n > bestN || (n == bestN && s < best) {
			best, bestN = s, n
		}
	}
	l.size = best

	l.allBold = true
	for _, r := range l.runs {
		if strings.TrimSpace(r.text) != "" && !r.bold {
			l.allBold = false
		}
	}
	return l, true
}

func (l *layoutLine) addText(s string, style styledRun) {
	if n := len(l.runs); n > 0 && l.runs[n-1].bold == style.bold && l.runs[n-1].italic == style.italic {
		l.runs[n-1].text += s
		return
	}
	l.runs = append(l.runs, styledRun{text: s, bold: style.bold, italic: style.italic})
}

// fontStyle infers bold/italic from a base font name such as "Arial-BoldMT"
// or "Calibri,Italic" (the library already strips the "ABCDEF+" subset tag).
func fontStyle(font string) styledRun {
	f := strings.ToLower(font)
	var s styledRun
	for _, k := range []string{"bold", "black", "heavy", "demi", "-bd", "-bi"} {
		if strings.Contains(f, k) {
			s.bold = true
		}
	}
	for _, k := range []string{"italic", "oblique", "-it", "-bi", ",it"} {
		if strings.Contains(f, k) {
			s.italic = true
		}
	}
	return s
}

func runsText(runs []styledRun) string {
	var b strings.Builder
	for _, r := range runs {
		b.WriteString(r.text)
	}
	return b.String()
}

func trimPrefixRuns(runs []styledRun, n int) []styledRun {
	out := make([]styledRun, 0, len(runs))
	for _, r := range runs {
		if n >= len(r.text) {
			n -= len(r.text)
			continue
		}
		r.text = r.text[n:]
		n = 0
		out = append(out, r)
	}
	return out
}

func bodyFontSize(lines []layoutLine) float64 {
	counts := map[float64]int{}
	for _, l := range lines {
		counts[l.size] += utf8.RuneCountInString(strings.TrimSpace(runsText(l.runs)))
	}
	best, bestN := 10.0, 0
	for s, n := range counts {
		if n > bestN || (n == bestN && s < best) {
			best, bestN = s, n
		}
	}
	return best
}

func isAllCaps(s string) bool {
	letters := 0
	for _, r := range s {
		if unicode.IsLower(r) {
			return false
		}
		if unicode.IsUpper(r) {
			letters++
		}
	}
	return letters >= 3
}

func headingLevel(l layoutLine, plain string, body float64) int {
	n := utf8.RuneCountInString(plain)
	ratio := l.size / body
	switch {
	case ratio >= 1.6 && n <= 100:
		return 1
	case ratio >= 1.25 && n <= 100:
		return 2
	case l.allBold && isAllCaps(plain) && n <= 60:
		return 2
	case l.allBold && ratio >= 1.1 && n <= 80:
		return 3
	}
	return 0
}

// canMerge reports whether ln is the wrapped continuation of block b rather
// than a new line of its own. Resumes are full of short standalone lines
// (name, title, contact, "Company    2019-2021"), so a plain paragraph line
// only joins the previous one if that one visibly ran to the right margin.
func canMerge(b *block, ln layoutLine, ext pageExtent) bool {
	if b.kind == blockHeading {
		return false
	}
	prev := b.last
	gap := prev.y - ln.y
	if prev.page != ln.page || gap <= 0 || gap > 1.6*prev.size {
		return false
	}
	if math.Abs(prev.size-ln.size) > 0.6 || prev.allBold != ln.allBold {
		return false
	}
	if b.kind == blockListItem {
		return ln.x >= b.markerX+3
	}
	if math.Abs(ln.x-prev.x) > 3 || prev.maxGap > 2.5*prev.size {
		return false
	}
	full := ext.maxEnd - ext.minX
	return full > 0 && prev.endX-prev.x >= 0.85*full
}

func buildStyledHTML(lines []layoutLine) string {
	body := bodyFontSize(lines)
	extents := map[int]pageExtent{}
	for _, l := range lines {
		e, ok := extents[l.page]
		if !ok {
			e = pageExtent{minX: l.x, maxEnd: l.endX}
		}
		e.minX = math.Min(e.minX, l.x)
		e.maxEnd = math.Max(e.maxEnd, l.endX)
		extents[l.page] = e
	}

	var blocks []*block
	for _, ln := range lines {
		plain := runsText(ln.runs)

		if n, k := listMarker(plain); k != kindParagraph {
			blocks = append(blocks, &block{kind: blockListItem, list: k, runs: trimPrefixRuns(ln.runs, n), last: ln, markerX: ln.x})
			continue
		}
		if lvl := headingLevel(ln, strings.TrimSpace(plain), body); lvl > 0 {
			blocks = append(blocks, &block{kind: blockHeading, level: lvl, runs: ln.runs, last: ln})
			continue
		}
		if n := len(blocks); n > 0 && canMerge(blocks[n-1], ln, extents[ln.page]) {
			b := blocks[n-1]
			b.runs = append(b.runs, styledRun{text: " "})
			b.runs = append(b.runs, ln.runs...)
			b.last = ln
			continue
		}
		blocks = append(blocks, &block{kind: blockPara, runs: ln.runs, last: ln})
	}

	var out strings.Builder
	listOpen := false
	var openKind lineKind
	closeList := func() {
		if listOpen {
			out.WriteString("</ol>")
			listOpen = false
		}
	}
	for _, b := range blocks {
		switch b.kind {
		case blockHeading:
			text := strings.TrimSpace(spaceRunRE.ReplaceAllString(runsText(b.runs), " "))
			if text == "" {
				continue
			}
			closeList()
			fmt.Fprintf(&out, "<h%d>%s</h%d>", b.level, html.EscapeString(text), b.level)
		case blockListItem:
			inner := renderRuns(b.runs)
			if inner == "" {
				continue
			}
			if !listOpen || openKind != b.list {
				closeList()
				out.WriteString(`<ol data-list="` + listDataAttr(b.list) + `">`)
				listOpen, openKind = true, b.list
			}
			out.WriteString(`<li data-list="` + listDataAttr(b.list) + `">` + inner + `</li>`)
		default:
			inner := renderRuns(b.runs)
			if inner == "" {
				continue
			}
			closeList()
			out.WriteString("<p>" + inner + "</p>")
		}
	}
	closeList()
	return out.String()
}

// renderRuns collapses whitespace across run boundaries and emits
// <strong>/<em> around each styled run, keeping edge spaces outside the tags.
func renderRuns(runs []styledRun) string {
	norm := make([]styledRun, 0, len(runs))
	lastSpace := true // drops leading whitespace
	for _, r := range runs {
		t := spaceRunRE.ReplaceAllString(r.text, " ")
		if lastSpace {
			t = strings.TrimLeft(t, " ")
		}
		if t == "" {
			continue
		}
		lastSpace = strings.HasSuffix(t, " ")
		r.text = t
		norm = append(norm, r)
	}
	if n := len(norm); n > 0 {
		norm[n-1].text = strings.TrimRight(norm[n-1].text, " ")
	}

	var b strings.Builder
	for _, r := range norm {
		core := strings.TrimSpace(r.text)
		if core == "" {
			b.WriteString(" ")
			continue
		}
		if strings.HasPrefix(r.text, " ") {
			b.WriteString(" ")
		}
		esc := html.EscapeString(core)
		if r.italic {
			esc = "<em>" + esc + "</em>"
		}
		if r.bold {
			esc = "<strong>" + esc + "</strong>"
		}
		b.WriteString(esc)
		if strings.HasSuffix(r.text, " ") {
			b.WriteString(" ")
		}
	}
	return strings.TrimSpace(b.String())
}
