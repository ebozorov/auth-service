package session

import "time"

// Session is a signed-in device: one session = one device (docs/DECISION.md #5).
type Session struct {
	ID        int64
	UserID    int64
	TokenHash string
	Device    Device `gorm:"serializer:json"`
	StartedIP string
	StartedAt time.Time
	CurrentIP string
	CurrentAt time.Time
	ExpiresAt time.Time
}

// Why a session ended; stored in ended_sessions.ended.reason.
const (
	ReasonLogout     = "logout"
	ReasonTerminated = "terminated"
	ReasonExpired    = "expired"
	ReasonDisabled   = "disabled"
)

// Ended describes how a session ended; stored as JSON in ended_sessions.ended.
// By and BySession are nil when the system ended it (e.g. expired).
type Ended struct {
	At        time.Time `json:"at"`
	Reason    string    `json:"reason"`
	By        *int64    `json:"by"`
	BySession *int64    `json:"by_session"`
}

// EndedBy is an Ended for a session ended by a user from one of their sessions.
func EndedBy(reason string, userID, sessionID int64) Ended {
	return Ended{At: time.Now(), Reason: reason, By: &userID, BySession: &sessionID}
}

// Response is a session as shown in the device list.
type Response struct {
	ID        int64     `json:"id,string"`
	Device    Device    `json:"device"`
	StartedIP string    `json:"started_ip"`
	StartedAt time.Time `json:"started_at"`
	CurrentIP string    `json:"current_ip"`
	CurrentAt time.Time `json:"current_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Current   bool      `json:"current"`
}

// NewResponses converts sessions for the device list; currentID marks the caller's own session.
func NewResponses(sessions []Session, currentID int64) []Response {
	resp := make([]Response, 0, len(sessions))
	for _, s := range sessions {
		resp = append(resp, Response{
			ID:        s.ID,
			Device:    s.Device,
			StartedIP: s.StartedIP,
			StartedAt: s.StartedAt,
			CurrentIP: s.CurrentIP,
			CurrentAt: s.CurrentAt,
			ExpiresAt: s.ExpiresAt,
			Current:   s.ID == currentID,
		})
	}
	return resp
}
