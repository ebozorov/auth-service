package auth

import (
	"time"

	"auth-service/internal/session"
)

// signinRequest is one "enter email" step; a magic link is emailed for it (docs/DECISION.md #3).
// When the link is used, session.Service.Start moves it to sessions.
type signinRequest struct {
	ID          int64
	UserID      int64
	TokenHash   string
	SecretHash  string
	Device      session.Device `gorm:"serializer:json"`
	RequestedIP string
	RequestedAt time.Time
	ExpiresAt   time.Time
}

func (signinRequest) TableName() string { return "signin_requests" }

type signInBody struct {
	Email  string         `json:"email"`
	Device session.Device `json:"device"`
}

type signInResponse struct {
	DeviceSecret string `json:"device_secret"`
	ExpiresIn    int    `json:"expires_in"`
}

type linkBody struct {
	Token        string `json:"token"`
	DeviceSecret string `json:"device_secret"`
}

type refreshBody struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	// Empty for browsers: their refresh token lives in a cookie.
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}
