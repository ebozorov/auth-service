package internal

import (
	"fmt"
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
