-- Employer <-> candidate messaging. A plain sender/recipient model;
-- "conversations" are derived at query time from (sender_id, recipient_id)
-- pairs rather than a separate conversations table, since a two-party
-- direct-message model doesn't need one.
CREATE TABLE messages (
    id           BIGSERIAL PRIMARY KEY,
    sender_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body         TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at      TIMESTAMPTZ,
    CHECK (sender_id <> recipient_id)
);

CREATE INDEX idx_messages_recipient ON messages(recipient_id, created_at DESC);
CREATE INDEX idx_messages_sender ON messages(sender_id, created_at DESC);
CREATE INDEX idx_messages_conversation ON messages(least(sender_id, recipient_id), greatest(sender_id, recipient_id), created_at);
