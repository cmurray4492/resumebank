package repo

import "time"

// SitemapEntry is the minimal data needed to emit one <url> in sitemap.xml.
type SitemapEntry struct {
	Slug      string
	UpdatedAt time.Time
}
