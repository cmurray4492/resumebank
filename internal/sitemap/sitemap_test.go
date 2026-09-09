package sitemap

import (
	"strings"
	"testing"
	"time"
)

func TestBuildXML_EmptyProducesValidUrlset(t *testing.T) {
	out := string(BuildXML(nil))
	if !strings.Contains(out, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`) {
		t.Errorf("expected a urlset with the sitemap namespace, got: %s", out)
	}
	if strings.Contains(out, "<url>") {
		t.Errorf("expected no <url> entries for an empty input, got: %s", out)
	}
}

func TestBuildXML_IncludesLocAndLastMod(t *testing.T) {
	lastMod := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	out := string(BuildXML([]URL{
		{Loc: "https://resumebank.biz/candidates/jane-doe", LastMod: lastMod},
	}))
	if !strings.Contains(out, "<loc>https://resumebank.biz/candidates/jane-doe</loc>") {
		t.Errorf("expected loc to be present, got: %s", out)
	}
	if !strings.Contains(out, "<lastmod>2026-03-15</lastmod>") {
		t.Errorf("expected lastmod formatted as YYYY-MM-DD, got: %s", out)
	}
}

func TestBuildXML_OmitsLastModWhenZero(t *testing.T) {
	out := string(BuildXML([]URL{{Loc: "https://resumebank.biz/"}}))
	if strings.Contains(out, "<lastmod>") {
		t.Errorf("expected no lastmod for a zero-value time, got: %s", out)
	}
}

func TestCache_DefaultsToValidEmptySitemap(t *testing.T) {
	c := NewCache()
	data, generatedAt := c.Get()
	if !strings.Contains(string(data), "<urlset") {
		t.Errorf("expected a default sitemap before any Set, got: %s", data)
	}
	if !generatedAt.IsZero() {
		t.Errorf("expected zero generatedAt before any Set, got: %v", generatedAt)
	}
}

func TestCache_SetThenGet(t *testing.T) {
	c := NewCache()
	before := time.Now()
	c.Set(BuildXML([]URL{{Loc: "https://resumebank.biz/"}}))
	data, generatedAt := c.Get()
	if !strings.Contains(string(data), "resumebank.biz") {
		t.Errorf("expected the set data to be returned, got: %s", data)
	}
	if generatedAt.Before(before) {
		t.Errorf("expected generatedAt to be set to roughly now")
	}
}
