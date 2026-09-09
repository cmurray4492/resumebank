package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"resumebank/internal/models"
)

type MessageRepo struct {
	pool *pgxpool.Pool
}

func NewMessageRepo(pool *pgxpool.Pool) *MessageRepo {
	return &MessageRepo{pool: pool}
}

func (r *MessageRepo) Create(ctx context.Context, senderID, recipientID int64, body string) (*models.Message, error) {
	m := &models.Message{SenderID: senderID, RecipientID: recipientID, Body: body}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO messages (sender_id, recipient_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		senderID, recipientID, body,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// ListThread returns all messages exchanged between userID and otherUserID,
// oldest first.
func (r *MessageRepo) ListThread(ctx context.Context, userID, otherUserID int64) ([]models.Message, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, sender_id, recipient_id, body, created_at, read_at
		FROM messages
		WHERE (sender_id = $1 AND recipient_id = $2) OR (sender_id = $2 AND recipient_id = $1)
		ORDER BY created_at ASC`, userID, otherUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.SenderID, &m.RecipientID, &m.Body, &m.CreatedAt, &m.ReadAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}

// MarkThreadRead marks every message sent by otherUserID to userID as read.
func (r *MessageRepo) MarkThreadRead(ctx context.Context, userID, otherUserID int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE messages SET read_at = now()
		WHERE recipient_id = $1 AND sender_id = $2 AND read_at IS NULL`, userID, otherUserID)
	return err
}

// UnreadCount returns how many unread messages userID has across all
// conversations, for a navbar badge.
func (r *MessageRepo) UnreadCount(ctx context.Context, userID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM messages WHERE recipient_id = $1 AND read_at IS NULL`, userID,
	).Scan(&count)
	return count, err
}

// Conversation is a lighter view used by the inbox page: who the other
// party is, a preview of the most recent message, and how many of their
// messages to the current user are unread.
type Conversation struct {
	OtherUserID int64
	LastBody    string
	LastAt      time.Time
	UnreadCount int
}

// ListConversations returns one row per distinct conversation partner for
// userID, most-recently-active first.
func (r *MessageRepo) ListConversations(ctx context.Context, userID int64) ([]Conversation, error) {
	rows, err := r.pool.Query(ctx, `
		WITH convo AS (
			SELECT
				CASE WHEN sender_id = $1 THEN recipient_id ELSE sender_id END AS other_id,
				body, created_at,
				(recipient_id = $1 AND read_at IS NULL) AS unread
			FROM messages
			WHERE sender_id = $1 OR recipient_id = $1
		)
		SELECT other_id, body, created_at, unread_count FROM (
			SELECT DISTINCT ON (other_id) other_id, body, created_at,
				(SELECT count(*) FROM convo c2 WHERE c2.other_id = convo.other_id AND c2.unread) AS unread_count
			FROM convo
			ORDER BY other_id, created_at DESC
		) latest
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.OtherUserID, &c.LastBody, &c.LastAt, &c.UnreadCount); err != nil {
			return nil, err
		}
		conversations = append(conversations, c)
	}
	return conversations, rows.Err()
}
