// Package mailer sends transactional email (currently just password reset
// links). It's deliberately vendor-neutral: SMTPMailer works with any
// provider that offers an SMTP endpoint (which is virtually all of
// them - Gmail, SendGrid, Mailgun, Postmark, AWS SES, Resend, ...), so
// there's no vendor-specific SDK/API-key format to commit to.
package mailer

import (
	"context"
	"fmt"
	"log"
	"net/mail"
	"net/smtp"
)

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// SMTPMailer sends real email via net/smtp. It relies on SendMail's
// automatic STARTTLS negotiation (used whenever the server advertises the
// STARTTLS extension, which covers port 587 on effectively every modern
// provider), so no separate TLS configuration is needed here.
type SMTPMailer struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func (m *SMTPMailer) Send(ctx context.Context, to, subject, body string) error {
	addr := m.Host + ":" + m.Port
	auth := smtp.PlainAuth("", m.Username, m.Password, m.Host)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		m.From, to, subject, body)

	errCh := make(chan error, 1)
	go func() {
		errCh <- smtp.SendMail(addr, auth, envelopeAddress(m.From), []string{to}, []byte(msg))
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// envelopeAddress extracts the bare email address from a
// "Display Name <addr>" (or plain "addr") string, for use as the SMTP
// envelope's "MAIL FROM:<...>" command - unlike a message's "From:" header,
// the envelope address must be bare, or servers reject it as a syntax
// error. Falls back to the input unchanged if it doesn't parse (better to
// let the SMTP server reject a genuinely malformed address than to send
// nothing).
func envelopeAddress(from string) string {
	if parsed, err := mail.ParseAddress(from); err == nil {
		return parsed.Address
	}
	return from
}

// LogMailer is the fallback used when SMTP isn't configured (e.g. local
// dev): it logs the email instead of sending it, so password reset and
// similar flows stay testable without a real mailbox. It must never be
// used to silently swallow email in production - see config.go, which
// only constructs this when SMTP_HOST is unset and logs a startup warning
// in that case if ENV=production.
type LogMailer struct{}

func (LogMailer) Send(ctx context.Context, to, subject, body string) error {
	log.Printf("=== EMAIL NOT SENT (SMTP not configured) ===\nTo: %s\nSubject: %s\n\n%s\n=== END EMAIL ===",
		to, subject, body)
	return nil
}
