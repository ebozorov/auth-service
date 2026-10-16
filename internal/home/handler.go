package home

import (
	"auth-service/internal/auth"

	"gorm.io/gorm"
)

// Handler serves the public, unauthenticated paths: /api/v1/, /api/v1/openapi.json, /.well-known/* and Swagger UI.
type Handler struct {
	db      *gorm.DB
	tokens  *auth.TokenManager
	openapi []byte
	opts    Options
}

func NewHandler(db *gorm.DB, tokens *auth.TokenManager, openapi []byte, opts Options) *Handler {
	return &Handler{db: db, tokens: tokens, openapi: openapi, opts: opts}
}
