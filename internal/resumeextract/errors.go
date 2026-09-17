package resumeextract

import "errors"

var (
	// ErrUnsupportedFormat means the file extension/content isn't a
	// supported PDF or DOCX.
	ErrUnsupportedFormat = errors.New("unsupported file type")
	// ErrCorruptFile means the file matched the expected format but
	// couldn't be parsed (malformed PDF, invalid zip, missing document
	// part, etc).
	ErrCorruptFile = errors.New("file could not be read")
	// ErrNoTextFound means the file parsed fine but contained no
	// extractable text (e.g. a scanned/image-only PDF).
	ErrNoTextFound = errors.New("no text found in file")
)
