package storage

import (
	"bytes"
	"io"
	"testing"
)

func TestLocalStorage_SaveOpenDelete(t *testing.T) {
	s := NewLocalStorage(t.TempDir())

	if err := s.Save("candidates/1/resume.pdf", bytes.NewReader([]byte("pdf-bytes"))); err != nil {
		t.Fatalf("Save: %v", err)
	}

	rc, err := s.Open("candidates/1/resume.pdf")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != "pdf-bytes" {
		t.Errorf("got %q, want %q", got, "pdf-bytes")
	}

	if err := s.Delete("candidates/1/resume.pdf"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Open("candidates/1/resume.pdf"); err == nil {
		t.Error("expected error opening deleted file")
	}
}

func TestLocalStorage_RejectsPathTraversal(t *testing.T) {
	s := NewLocalStorage(t.TempDir())

	if err := s.Save("../../etc/passwd", bytes.NewReader([]byte("evil"))); err == nil {
		t.Error("expected path traversal to be rejected")
	}
	if rc, err := s.Open("../../etc/passwd"); err == nil {
		rc.Close()
		t.Error("expected path traversal to be rejected on open")
	}
}

func TestValidateExtension(t *testing.T) {
	if err := ValidateExtension("resume.pdf", true); err != nil {
		t.Errorf("expected .pdf to be valid for resume, got %v", err)
	}
	if err := ValidateExtension("resume.exe", true); err == nil {
		t.Error("expected .exe to be rejected for resume")
	}
	if err := ValidateExtension("photo.png", false); err != nil {
		t.Errorf("expected .png to be valid for additional file, got %v", err)
	}
	if err := ValidateExtension("script.sh", false); err == nil {
		t.Error("expected .sh to be rejected for additional file")
	}
}

func TestSafeFilename(t *testing.T) {
	got := SafeFilename("../../etc/passwd")
	if got == "../../etc/passwd" {
		t.Error("expected directory traversal characters to be stripped")
	}
	if len(got) < len("passwd") {
		t.Errorf("expected filename to retain base name, got %q", got)
	}
}
