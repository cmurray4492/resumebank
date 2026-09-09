package repo

// EmbeddingTarget is a row awaiting an embedding computation, returned by
// the MissingEmbeddings backfill queries.
type EmbeddingTarget struct {
	ID   int64
	Text string
}
