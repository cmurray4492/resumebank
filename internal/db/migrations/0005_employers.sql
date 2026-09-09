CREATE TABLE employers (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    slug          TEXT NOT NULL UNIQUE,
    company_name  TEXT NOT NULL,
    industry      TEXT NOT NULL DEFAULT '',
    city          TEXT NOT NULL DEFAULT '',
    state         TEXT NOT NULL DEFAULT '',
    zipcode       TEXT NOT NULL,
    phone         TEXT NOT NULL DEFAULT '',
    email_address TEXT NOT NULL DEFAULT '',
    website       TEXT NOT NULL DEFAULT '',
    description   TEXT NOT NULL DEFAULT '',
    locations     TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
