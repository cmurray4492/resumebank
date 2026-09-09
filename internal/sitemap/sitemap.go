// Package sitemap builds sitemap.xml content per the sitemaps.org protocol
// and holds the most recently generated copy for the server to serve.
package sitemap

import (
	"bytes"
	"encoding/xml"
	"sync"
	"time"
)

// URL is one <url> entry.
type URL struct {
	Loc     string
	LastMod time.Time
}

type xmlURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlSet struct {
	XMLName xml.Name `xml:"urlset"`
	Xmlns   string   `xml:"xmlns,attr"`
	URLs    []xmlURL `xml:"url"`
}

// BuildXML renders urls as a complete sitemap.xml document.
func BuildXML(urls []URL) []byte {
	set := urlSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, u := range urls {
		entry := xmlURL{Loc: u.Loc}
		if !u.LastMod.IsZero() {
			entry.LastMod = u.LastMod.UTC().Format("2006-01-02")
		}
		set.URLs = append(set.URLs, entry)
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	_ = enc.Encode(set) // urlSet has no cyclic/unsupported types, so Encode cannot fail here
	buf.WriteByte('\n')
	return buf.Bytes()
}

// Cache holds the most recently generated sitemap so requests are served
// instantly instead of hitting the database on every request.
type Cache struct {
	mu          sync.RWMutex
	data        []byte
	generatedAt time.Time
}

func NewCache() *Cache {
	return &Cache{data: BuildXML(nil)}
}

func (c *Cache) Set(data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = data
	c.generatedAt = time.Now()
}

func (c *Cache) Get() ([]byte, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data, c.generatedAt
}
