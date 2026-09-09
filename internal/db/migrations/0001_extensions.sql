-- Enabled now so later phases (candidate/job embeddings for RAG matching)
-- can add `vector(...)` columns via plain ALTER TABLE statements.
CREATE EXTENSION IF NOT EXISTS vector;
