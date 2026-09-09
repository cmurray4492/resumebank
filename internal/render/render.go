// Package render wraps html/template loading and execution.
package render

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"sync"

	webassets "resumebank/web"
)

type Renderer struct {
	dev       bool
	diskRoot  string
	mu        sync.RWMutex
	templates map[string]*template.Template
	funcs     template.FuncMap
}

// New creates a Renderer. When dev is true, templates are re-parsed from
// diskRoot on every request (hot reload); otherwise they're parsed once
// from the embedded filesystem.
func New(dev bool, diskRoot string) (*Renderer, error) {
	r := &Renderer{dev: dev, diskRoot: diskRoot, funcs: FuncMap()}
	if !dev {
		if err := r.loadAll(); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Renderer) loadAll() error {
	pagesFS, err := fs.Sub(webassets.FS, "templates/pages")
	if err != nil {
		return err
	}
	entries, err := fs.ReadDir(pagesFS, ".")
	if err != nil {
		return err
	}

	templates := make(map[string]*template.Template)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		tmpl := template.New(name).Funcs(r.funcs)
		tmpl, err := tmpl.ParseFS(webassets.FS,
			"templates/layout/*.html.tmpl",
			"templates/partials/*.html.tmpl",
			"templates/pages/"+name,
		)
		if err != nil {
			return fmt.Errorf("parsing template %s: %w", name, err)
		}
		templates[name] = tmpl
	}

	r.mu.Lock()
	r.templates = templates
	r.mu.Unlock()
	return nil
}

func (r *Renderer) getTemplate(page string) (*template.Template, error) {
	if r.dev {
		tmpl := template.New(page).Funcs(r.funcs)
		tmpl, err := tmpl.ParseGlob(filepath.Join(r.diskRoot, "layout", "*.html.tmpl"))
		if err != nil {
			return nil, err
		}
		tmpl, err = tmpl.ParseGlob(filepath.Join(r.diskRoot, "partials", "*.html.tmpl"))
		if err != nil {
			return nil, err
		}
		tmpl, err = tmpl.ParseFiles(filepath.Join(r.diskRoot, "pages", page))
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", page, err)
		}
		return tmpl, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	tmpl, ok := r.templates[page]
	if !ok {
		return nil, fmt.Errorf("template %s not found", page)
	}
	return tmpl, nil
}

// Render executes the named page template (e.g. "home.html.tmpl") using
// "base.html.tmpl" as the entrypoint.
func (r *Renderer) Render(w http.ResponseWriter, status int, page string, data any) {
	tmpl, err := r.getTemplate(page)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "base.html.tmpl", data); err != nil {
		// Headers/status are already written; log-style fallback only.
		fmt.Fprintf(w, "<!-- render error: %v -->", err)
	}
}

// StaticFileSystem returns the filesystem to serve /static/* from: the OS
// filesystem in dev (for hot reload), the embedded FS in production.
func StaticFileSystem(dev bool, diskStaticRoot string) http.FileSystem {
	if dev {
		return http.Dir(diskStaticRoot)
	}
	sub, err := fs.Sub(webassets.FS, "static")
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}
