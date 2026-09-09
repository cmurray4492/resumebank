-- Embedding columns for the candidate/job matching (RAG) features, backed
-- by a local Ollama server running nomic-embed-text (768 dimensions).
-- No ANN index (ivfflat/hnsw) yet: those need representative data present
-- to choose good parameters, and a brute-force `ORDER BY embedding <=> $1`
-- scan is fast enough at the row counts this app has today. Add one when
-- the corpus grows large enough for it to matter.
ALTER TABLE candidates ADD COLUMN embedding vector(768);
ALTER TABLE jobs ADD COLUMN embedding vector(768);
