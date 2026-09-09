package web

import (
	"io"
	"net/http"
	"strconv"

	"resumebank/internal/app"
	"resumebank/internal/httpx"
	"resumebank/internal/repo"
)

type FileHandlers struct {
	App *app.App
}

func NewFileHandlers(a *app.App) *FileHandlers { return &FileHandlers{App: a} }

// Download streams a candidate file publicly (all uploaded files are public
// per spec) with correct content type/disposition headers.
func (h *FileHandlers) Download(w http.ResponseWriter, r *http.Request) {
	fileID, err := strconv.ParseInt(r.PathValue("fileID"), 10, 64)
	if err != nil {
		httpBadRequest(w, err)
		return
	}
	record, err := h.App.Files.GetByID(r.Context(), fileID)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
			return
		}
		httpServerError(w, err)
		return
	}

	f, err := h.App.Storage.Open(record.StoredPath)
	if err != nil {
		httpx.NotFound(w, r)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", record.ContentType)
	w.Header().Set("Content-Disposition", `inline; filename="`+record.OriginalFilename+`"`)
	io.Copy(w, f)
}
