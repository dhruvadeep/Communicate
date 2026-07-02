package mail

import (
	"context"
	"fmt"

	gomail "github.com/wneessen/go-mail"
)

// Config holds SMTP connection settings.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
	FromName string
}

// Mailer sends emails via SMTP.
type Mailer struct {
	cfg Config
}

// New returns a Mailer configured with cfg.
func New(cfg Config) *Mailer {
	return &Mailer{cfg: cfg}
}

// Send sends an email. At least one of htmlBody or textBody must be non-empty.
// If both are set the message is sent as multipart/alternative (text primary, HTML alternative).
func (m *Mailer) Send(ctx context.Context, subject, htmlBody, textBody string, to ...string) error {
	msg := gomail.NewMsg()

	if m.cfg.FromName != "" {
		if err := msg.FromFormat(m.cfg.FromName, m.cfg.From); err != nil {
			return fmt.Errorf("mail: from: %w", err)
		}
	} else {
		if err := msg.From(m.cfg.From); err != nil {
			return fmt.Errorf("mail: from: %w", err)
		}
	}

	for _, addr := range to {
		if err := msg.To(addr); err != nil {
			return fmt.Errorf("mail: to: %w", err)
		}
	}

	msg.Subject(subject)

	// Set body — prefer plain text as primary, HTML as alternative
	switch {
	case textBody != "" && htmlBody != "":
		msg.SetBodyString(gomail.TypeTextPlain, textBody)
		msg.AddAlternativeString(gomail.TypeTextHTML, htmlBody)
	case htmlBody != "":
		msg.SetBodyString(gomail.TypeTextHTML, htmlBody)
	case textBody != "":
		msg.SetBodyString(gomail.TypeTextPlain, textBody)
	default:
		return fmt.Errorf("mail: at least one of htmlBody or textBody must be set")
	}

	client, err := gomail.NewClient(m.cfg.Host,
		gomail.WithPort(m.cfg.Port),
		gomail.WithSMTPAuth(gomail.SMTPAuthPlain),
		gomail.WithUsername(m.cfg.User),
		gomail.WithPassword(m.cfg.Password),
		gomail.WithTLSPolicy(gomail.TLSMandatory),
	)
	if err != nil {
		return fmt.Errorf("mail: client: %w", err)
	}

	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("mail: send: %w", err)
	}

	return nil
}
