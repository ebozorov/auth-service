package internal

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	log, err := NewLogger(&buf, LogFormatJSON, "WARN")
	if err != nil {
		t.Fatal(err)
	}
	log.Info("hidden")
	log.Warn("shown", "key", "value")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("not one JSON line: %q", buf.String())
	}
	if rec["msg"] != "shown" || rec["level"] != "WARN" || rec["key"] != "value" {
		t.Errorf("record %v", rec)
	}

	buf.Reset()
	log, err = NewLogger(&buf, LogFormatText, "debug")
	if err != nil {
		t.Fatal(err)
	}
	log.Debug("hello", "n", 1)
	if got := buf.String(); !strings.Contains(got, "level=DEBUG") || !strings.Contains(got, "n=1") {
		t.Errorf("text line %q", got)
	}
}

func TestNewLoggerRejects(t *testing.T) {
	if _, err := NewLogger(nil, "yaml", "info"); err == nil {
		t.Error("unknown format accepted")
	}
	if _, err := NewLogger(nil, LogFormatJSON, "verbose"); err == nil {
		t.Error("unknown level accepted")
	}
}

func TestMaskEmail(t *testing.T) {
	for in, want := range map[string]string{
		"bob@example.com": "b***@example.com",
		"a@localhost":     "a***@localhost",
		"@example.com":    "***",
		"not-an-email":    "***",
		"":                "***",
	} {
		if got := MaskEmail(in); got != want {
			t.Errorf("MaskEmail(%q) = %q, want %q", in, got, want)
		}
	}
}
