package internal

import (
	"fmt"
	"strconv"
	"time"

	"github.com/sony/sonyflake"
)

// idStartTime is the zero point of the time part of generated ids.
var idStartTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

var flake *sonyflake.Sonyflake

// InitID prepares the id generator. Call it once at startup (docs/DECISION.md #14).
// The machine id is the lower 16 bits of the pod's private IP, so ids are unique
// across the whole cluster.
func InitID() error {
	sf, err := sonyflake.New(sonyflake.Settings{StartTime: idStartTime})
	if err != nil {
		return fmt.Errorf("sonyflake: %w", err)
	}
	flake = sf
	return nil
}

// NewID returns a new unique, time-ordered id.
func NewID() int64 {
	id, err := flake.NextID()
	if err != nil {
		// Only happens after ~174 years of running; nothing sensible to do.
		panic(fmt.Sprintf("sonyflake: %v", err))
	}
	return int64(id)
}

// ID is an id read from JSON that accepts both "123" and 123, and is written as a string:
// ids exceed 2^53, so JavaScript would round a JSON number (docs/DECISION.md #21, #22).
// Response models that are only written use int64 with the `json:",string"` tag instead;
// it encodes without an allocation per id.
type ID int64

func (id ID) MarshalJSON() ([]byte, error) {
	return strconv.AppendQuote(nil, strconv.FormatInt(int64(id), 10)), nil
}

func (id *ID) UnmarshalJSON(b []byte) error {
	s := string(b)
	if s == "null" {
		return nil
	}
	if unquoted, err := strconv.Unquote(s); err == nil {
		s = unquoted
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("id: %q is not an integer", b)
	}
	*id = ID(v)
	return nil
}
