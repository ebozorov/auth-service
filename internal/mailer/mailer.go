package mailer

import (
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// Mailer sends plain-text emails over SMTP. auth-service sends email itself,
// without any other service.
type Mailer struct {
	cfg Config
}

func New(cfg Config) *Mailer {
	return &Mailer{cfg: cfg}
}

// Send delivers one email. Without SMTP_HOST the email is only written to the log, so
// sign-in links can be copied from there. The body holds a one-time link, so config
// allows this only in dev (docs/DECISION.md #13, #26).
func (m *Mailer) Send(to, subject, body string) error {
	if m.cfg.Host == "" {
		slog.Warn("email yuborilmadi, faqat logga yozildi (SMTP_HOST bo'sh)", "to", to, "subject", subject, "body", body)
		return nil
	}

	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}
	addr := net.JoinHostPort(m.cfg.Host, m.cfg.Port)
	// SendMail switches to TLS (STARTTLS) when the server supports it.
	return smtp.SendMail(addr, auth, m.cfg.From, []string{to}, m.message(to, subject, body))
}

func (m *Mailer) message(to, subject, body string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", m.cfg.From)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(body, "\n", "\r\n"))
	return []byte(b.String())
}
