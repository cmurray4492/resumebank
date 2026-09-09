package mailer

import (
	"context"
	"testing"
)

func TestLogMailer_NeverErrors(t *testing.T) {
	var m LogMailer
	if err := m.Send(context.Background(), "someone@example.com", "Subject", "Body"); err != nil {
		t.Errorf("expected LogMailer.Send to never error, got %v", err)
	}
}
