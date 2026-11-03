package internal

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

const (
	LogFormatText = "text"
	LogFormatJSON = "json"
)

// NewLogger builds the process logger (docs/DECISION.md #26): JSON for log collectors,
// text for reading in a terminal. main sets it as slog's default, which also takes over
// the standard log package.
func NewLogger(w io.Writer, format, level string) (*slog.Logger, error) {
	lvl, err := parseLogLevel(level)
	if err != nil {
		return nil, err
	}
	if err := checkLogFormat(format); err != nil {
		return nil, err
	}
	opts := &slog.HandlerOptions{Level: lvl}
	if format == LogFormatJSON {
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	}
	return slog.New(slog.NewTextHandler(w, opts)), nil
}

func checkLogFormat(format string) error {
	if format != LogFormatText && format != LogFormatJSON {
		return fmt.Errorf("must be %s or %s", LogFormatText, LogFormatJSON)
	}
	return nil
}

// parseLogLevel accepts debug, info, warn and error in any case.
func parseLogLevel(s string) (slog.Level, error) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(s)); err != nil {
		return 0, fmt.Errorf("must be debug, info, warn or error")
	}
	return lvl, nil
}

// MaskEmail keeps the first letter and the domain: "bob@example.com" → "b***@example.com".
// Enough to tell users apart in a log, without writing the address itself.
func MaskEmail(email string) string {
	local, domain, ok := strings.Cut(email, "@")
	if !ok || local == "" {
		return "***"
	}
	return local[:1] + "***@" + domain
}
