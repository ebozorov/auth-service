package mailer

import (
	"io"
	"mime"
	"net/mail"
	"strings"
	"testing"
)

func TestMessage(t *testing.T) {
	m := New(Config{From: "no-reply@example.com"})
	raw := m.message("bob@example.com", "Tizimga kirish — link", "line 1\nline 2\n")

	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	h := msg.Header
	if h.Get("From") != "no-reply@example.com" || h.Get("To") != "bob@example.com" {
		t.Errorf("from %q, to %q", h.Get("From"), h.Get("To"))
	}
	if _, err := h.Date(); err != nil {
		t.Errorf("date: %v", err)
	}
	if h.Get("Content-Type") != "text/plain; charset=utf-8" || h.Get("MIME-Version") != "1.0" {
		t.Errorf("content-type %q, mime-version %q", h.Get("Content-Type"), h.Get("MIME-Version"))
	}

	// A non-ASCII subject must be encoded, and decode back to the original.
	subject := h.Get("Subject")
	if !strings.HasPrefix(subject, "=?utf-8?q?") {
		t.Errorf("subject is not encoded: %q", subject)
	}
	decoded, err := new(mime.WordDecoder).DecodeHeader(subject)
	if err != nil || decoded != "Tizimga kirish — link" {
		t.Errorf("subject decodes to %q, %v", decoded, err)
	}

	// SMTP needs CRLF line endings, with no bare LF left.
	body, err := io.ReadAll(msg.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "line 1\r\nline 2\r\n" {
		t.Errorf("body %q", body)
	}
	if strings.Contains(strings.ReplaceAll(string(raw), "\r\n", ""), "\n") {
		t.Error("message has a bare LF")
	}
}

// Without SMTP_HOST the email is only logged, never sent.
func TestSendWithoutHost(t *testing.T) {
	if err := New(Config{}).Send("bob@example.com", "subject", "body"); err != nil {
		t.Fatal(err)
	}
}
