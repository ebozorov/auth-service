package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"auth-service/internal"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// currentInterval limits how often current_ip and current_at are written.
const currentInterval = time.Minute

var (
	ErrNotFound     = errors.New("session not found")
	ErrInvalidLink  = errors.New("invalid or expired link")
	ErrInvalidToken = errors.New("invalid or expired refresh token")
)

// Service owns the sessions table and both moves around it:
// signin_requests → sessions → ended_sessions (docs/DECISION.md #16).
type Service struct {
	db  *gorm.DB
	ttl time.Duration
}

func NewService(db *gorm.DB, ttl time.Duration) *Service {
	return &Service{db: db, ttl: ttl}
}

// Start opens a session from a signin request: the request is deleted and its id
// and device move to sessions, in one statement, so a link works only once.
// The link needs the secret of the device that asked for it (docs/DECISION.md #4),
// and the user must still be enabled.
func (s *Service) Start(tokenHash, secretHash, ip string) (*Session, string, error) {
	token, hash := internal.NewToken()

	var sess Session
	result := s.db.Raw(`
		WITH used AS (
			DELETE FROM signin_requests r
			USING users u
			WHERE u.id = r.user_id AND u.enabled
			  AND r.token_hash = ? AND r.secret_hash = ? AND r.expires_at > now()
			RETURNING r.id, r.user_id, r.device
		)
		INSERT INTO sessions (id, user_id, token_hash, device, started_ip, started_at, current_ip, current_at, expires_at)
		SELECT id, user_id, ?, device, ?, now(), ?, now(), ?
		FROM used
		RETURNING *`,
		tokenHash, secretHash, hash, ip, ip, time.Now().Add(s.ttl),
	).Scan(&sess)
	if result.Error != nil {
		return nil, "", result.Error
	}
	if result.RowsAffected == 0 {
		return nil, "", ErrInvalidLink
	}
	return &sess, token, nil
}

// Rotate swaps a refresh token for a new one. It is one UPDATE ... RETURNING,
// so each refresh token works only once.
func (s *Service) Rotate(refreshToken, ip string) (*Session, string, error) {
	token, hash := internal.NewToken()
	now := time.Now()

	var sess Session
	result := s.db.Model(&sess).
		Clauses(clause.Returning{}).
		Where("token_hash = ? AND expires_at > ?", internal.HashToken(refreshToken), now).
		Updates(map[string]any{
			"token_hash": hash,
			"current_ip": ip,
			"current_at": now,
			"expires_at": now.Add(s.ttl),
		})
	if result.Error != nil {
		return nil, "", result.Error
	}
	if result.RowsAffected == 0 {
		return nil, "", ErrInvalidToken
	}
	return &sess, token, nil
}

// FindActive returns the user's session if it has not expired.
func (s *Service) FindActive(id, userID int64) (*Session, error) {
	var sess Session
	err := s.db.Where("id = ? AND user_id = ? AND expires_at > now()", id, userID).First(&sess).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

// Touch updates current_ip and current_at together, at most once per currentInterval.
func (s *Service) Touch(sess *Session, ip string) {
	if time.Since(sess.CurrentAt) > currentInterval {
		s.db.Model(sess).Updates(map[string]any{"current_ip": ip, "current_at": time.Now()})
	}
}

// List returns the user's active sessions, most recently used first.
func (s *Service) List(userID int64) ([]Session, error) {
	var sessions []Session
	err := s.db.Where("user_id = ? AND expires_at > now()", userID).
		Order("current_at DESC").
		Find(&sessions).Error
	return sessions, err
}

// End ends one session of the user.
func (s *Service) End(id, userID int64, ended Ended) error {
	n, err := s.archive(ended, "id = ? AND user_id = ?", id, userID)
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// EndOthers ends all sessions of the user except keepID.
func (s *Service) EndOthers(userID, keepID int64, ended Ended) (int64, error) {
	return s.archive(ended, "user_id = ? AND id <> ?", userID, keepID)
}

// EndAll ends every session of the user, e.g. when an admin disables them.
func (s *Service) EndAll(userID int64, ended Ended) (int64, error) {
	return s.archive(ended, "user_id = ?", userID)
}

// EndExpired archives expired sessions. Called periodically.
func (s *Service) EndExpired() (int64, error) {
	return s.archive(Ended{At: time.Now(), Reason: ReasonExpired}, "expires_at <= now()")
}

// archive moves the matching sessions to ended_sessions in one statement.
// Columns keep their names; token_hash is not archived.
// where is always a constant from this file, never user input.
func (s *Service) archive(ended Ended, where string, args ...any) (int64, error) {
	endedJSON, err := json.Marshal(ended)
	if err != nil {
		return 0, err
	}
	query := fmt.Sprintf(`
		WITH ended AS (
			DELETE FROM sessions WHERE %s RETURNING *
		)
		INSERT INTO ended_sessions
			(id, user_id, device, started_ip, started_at, current_ip, current_at, expires_at, ended)
		SELECT id, user_id, device, started_ip, started_at, current_ip, current_at, expires_at, ?::jsonb
		FROM ended`, where)

	result := s.db.Exec(query, append(args, string(endedJSON))...)
	return result.RowsAffected, result.Error
}
