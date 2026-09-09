// Package storage abstracts file persistence so the local-disk
// implementation used in Phase 1 can later be swapped for object storage
// (e.g. when deployed on Railway, which does not guarantee a persistent
// local filesystem across deploys) without changing callers.
package storage

import "io"

type Storage interface {
	// Save stores content under the given relative key and returns nothing;
	// callers persist the key themselves (e.g. in candidate_files.stored_path).
	Save(key string, content io.Reader) error
	Open(key string) (io.ReadCloser, error)
	Delete(key string) error
}
