package storage

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var resumeExts = map[string]bool{".pdf": true}

var additionalExts = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true,
	".png": true, ".jpg": true, ".jpeg": true, ".txt": true,
}

var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// ValidateExtension checks filename's extension against the allowlist for
// the given file kind ("resume_pdf" or "additional").
func ValidateExtension(filename string, isResume bool) error {
	ext := strings.ToLower(filepath.Ext(filename))
	allowed := additionalExts
	if isResume {
		allowed = resumeExts
	}
	if !allowed[ext] {
		return fmt.Errorf("file type %q is not allowed", ext)
	}
	return nil
}

// SafeFilename strips directory components and unsafe characters, then
// prefixes a UUID to avoid collisions while keeping the name recognizable.
func SafeFilename(original string) string {
	base := filepath.Base(original)
	base = unsafeChars.ReplaceAllString(base, "_")
	if base == "" || base == "." {
		base = "file"
	}
	return fmt.Sprintf("%s-%s", uuid.NewString(), base)
}

// CandidateFileKey builds the storage key for a candidate's uploaded file.
func CandidateFileKey(candidateID int64, safeFilename string) string {
	return fmt.Sprintf("candidates/%d/%s", candidateID, safeFilename)
}
