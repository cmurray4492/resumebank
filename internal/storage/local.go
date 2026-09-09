package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalStorage struct {
	Root string
}

func NewLocalStorage(root string) *LocalStorage {
	return &LocalStorage{Root: root}
}

// safeJoin resolves key against root, rejecting any key containing ".."
// path segments or that otherwise resolves outside of root.
func (l *LocalStorage) safeJoin(key string) (string, error) {
	if strings.Contains(key, "..") {
		return "", fmt.Errorf("invalid storage key: %q must not contain \"..\"", key)
	}

	absRoot, err := filepath.Abs(l.Root)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(filepath.Join(absRoot, key))
	if err != nil {
		return "", err
	}
	if absPath != absRoot && !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid storage key: %q escapes storage root", key)
	}
	return absPath, nil
}

func (l *LocalStorage) Save(key string, content io.Reader) error {
	path, err := l.safeJoin(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, content)
	return err
}

func (l *LocalStorage) Open(key string) (io.ReadCloser, error) {
	path, err := l.safeJoin(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (l *LocalStorage) Delete(key string) error {
	path, err := l.safeJoin(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
