// Package web embeds the site's templates and static assets so the built
// binary can serve them without depending on the filesystem in production.
// (Named distinctly from internal/web, which holds the HTTP handlers.)
package web

import "embed"

//go:embed templates static
var FS embed.FS
