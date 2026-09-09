CREATE TABLE blog_posts (
    id            BIGSERIAL PRIMARY KEY,
    slug          TEXT NOT NULL UNIQUE,
    title         TEXT NOT NULL,
    body_html     TEXT NOT NULL,
    body_text     TEXT NOT NULL,
    author_name   TEXT NOT NULL DEFAULT '',
    published     BOOLEAN NOT NULL DEFAULT false,
    published_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_blog_posts_published ON blog_posts(published, published_at DESC);
