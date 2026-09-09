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

// TestEnvelopeAddress_StripsDisplayName guards against a real bug shipped
// to production: passing a "Display Name <addr>" SMTP_FROM straight
// through as SendMail's envelope-from argument, which servers (Gmail
// included) reject with a syntax error since the SMTP envelope's
// "MAIL FROM:<...>" command requires a bare address - the display-name
// form is only valid in the message's own "From:" header.
func TestEnvelopeAddress_StripsDisplayName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"resumebank.biz <no-reply@resumebank.biz>", "no-reply@resumebank.biz"},
		{"\"Display, Name\" <addr@example.com>", "addr@example.com"},
		{"bare@example.com", "bare@example.com"},
		{"not a valid address at all", "not a valid address at all"}, // falls back unchanged
	}
	for _, c := range cases {
		if got := envelopeAddress(c.in); got != c.want {
			t.Errorf("envelopeAddress(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
