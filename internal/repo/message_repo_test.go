package repo_test

import (
	"context"
	"testing"

	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/testutil"
)

func TestMessageRepo_CreateListThreadMarkRead(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	messages := repo.NewMessageRepo(pool)
	ctx := context.Background()

	candidateUser := newTestUser(t, users, "candidate@example.com", models.RoleCandidate)
	employerUser := newTestUser(t, users, "employer@example.com", models.RoleEmployer)

	if _, err := messages.Create(ctx, employerUser.ID, candidateUser.ID, "Interested in your profile"); err != nil {
		t.Fatalf("Create (employer->candidate): %v", err)
	}
	if _, err := messages.Create(ctx, candidateUser.ID, employerUser.ID, "Thanks, tell me more"); err != nil {
		t.Fatalf("Create (candidate->employer): %v", err)
	}

	thread, err := messages.ListThread(ctx, candidateUser.ID, employerUser.ID)
	if err != nil {
		t.Fatalf("ListThread: %v", err)
	}
	if len(thread) != 2 {
		t.Fatalf("expected 2 messages in thread, got %d", len(thread))
	}
	if thread[0].Body != "Interested in your profile" {
		t.Errorf("expected oldest message first, got %q", thread[0].Body)
	}

	unread, err := messages.UnreadCount(ctx, candidateUser.ID)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if unread != 1 {
		t.Fatalf("expected 1 unread message for candidate, got %d", unread)
	}

	if err := messages.MarkThreadRead(ctx, candidateUser.ID, employerUser.ID); err != nil {
		t.Fatalf("MarkThreadRead: %v", err)
	}
	unread, err = messages.UnreadCount(ctx, candidateUser.ID)
	if err != nil {
		t.Fatalf("UnreadCount after mark read: %v", err)
	}
	if unread != 0 {
		t.Fatalf("expected 0 unread after marking read, got %d", unread)
	}
}

func TestMessageRepo_ListConversations(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	messages := repo.NewMessageRepo(pool)
	ctx := context.Background()

	candidateUser := newTestUser(t, users, "candidate@example.com", models.RoleCandidate)
	employerA := newTestUser(t, users, "a@example.com", models.RoleEmployer)
	employerB := newTestUser(t, users, "b@example.com", models.RoleEmployer)

	if _, err := messages.Create(ctx, employerA.ID, candidateUser.ID, "Hello from A"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := messages.Create(ctx, employerB.ID, candidateUser.ID, "Hello from B"); err != nil {
		t.Fatalf("Create: %v", err)
	}

	conversations, err := messages.ListConversations(ctx, candidateUser.ID)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(conversations) != 2 {
		t.Fatalf("expected 2 conversations, got %d: %+v", len(conversations), conversations)
	}
	for _, c := range conversations {
		if c.UnreadCount != 1 {
			t.Errorf("expected each conversation to have 1 unread message, got %d for other_id=%d", c.UnreadCount, c.OtherUserID)
		}
	}
}
