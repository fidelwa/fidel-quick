// Package email provides a minimal email-sending abstraction with pluggable
// providers. The provider is selected via the EMAIL_PROVIDER env var.
//
// Only the "stdout" provider is implemented today (default for dev): it logs
// the message — including any action link — to the app logger instead of
// delivering a real email. Wiring a real transactional provider (SES,
// Resend, etc.) is tracked separately in FID-17.
package email

import (
	"context"
	"log/slog"
)

// Message is a single outbound email.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender delivers an email message. Implementations must be safe for
// concurrent use.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// NewSender builds a Sender for the given provider name. Unknown providers
// (including the empty string) fall back to the stdout provider so that a
// misconfigured env never breaks the boot — the operator sees the link in the
// logs and password reset keeps working in dev.
func NewSender(provider, from string, log *slog.Logger) Sender {
	switch provider {
	case "stdout", "":
		return NewStdoutSender(from, log)
	default:
		log.Warn("unknown EMAIL_PROVIDER, falling back to stdout", "provider", provider)
		return NewStdoutSender(from, log)
	}
}

// StdoutSender "sends" email by logging it. Useful for local dev and tests:
// the reset link shows up in the server logs.
type StdoutSender struct {
	from string
	log  *slog.Logger
}

// NewStdoutSender returns a StdoutSender. A nil logger is replaced with the
// default slog logger so callers never have to guard against it.
func NewStdoutSender(from string, log *slog.Logger) *StdoutSender {
	if log == nil {
		log = slog.Default()
	}
	return &StdoutSender{from: from, log: log}
}

func (s *StdoutSender) Send(_ context.Context, msg Message) error {
	s.log.Info("email (stdout provider)",
		"from", s.from,
		"to", msg.To,
		"subject", msg.Subject,
		"body", msg.Body,
	)
	return nil
}
