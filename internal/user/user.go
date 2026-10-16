package user

import (
	"errors"
	"strings"
	"time"

	"auth-api/internal"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already exists")
)

// User is a company employee. Users are added in advance by an admin;
// there is no self sign-up (docs/DECISION.md #1).
type User struct {
	ID        int64      `json:"id"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	Enabled   bool       `json:"enabled"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"autoUpdateTime:false"`
}

func IsValidRole(role string) bool {
	return role == RoleAdmin || role == RoleUser
}

// NormalizeEmail returns the form stored in users.email: trimmed and lowercase.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Get(id int64) (*User, error) {
	return s.first("id = ?", id)
}

func (s *Service) GetByEmail(email string) (*User, error) {
	return s.first("email = ?", NormalizeEmail(email))
}

func (s *Service) List() ([]User, error) {
	var users []User
	err := s.db.Order("id").Find(&users).Error
	return users, err
}

func (s *Service) Create(email, name, role string) (*User, error) {
	u := &User{
		ID:      internal.NewID(),
		Email:   NormalizeEmail(email),
		Name:    strings.TrimSpace(name),
		Role:    role,
		Enabled: true,
	}
	err := s.db.Create(u).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return nil, ErrEmailTaken
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// Save writes name, role and enabled of an existing user.
func (s *Service) Save(u *User) error {
	now := time.Now()
	u.UpdatedAt = &now
	return s.db.Model(u).Select("name", "role", "enabled", "updated_at").Updates(u).Error
}

// EnsureAdmin creates the first admin when there is none yet (docs/DECISION.md #9).
// If the email already belongs to a user, that user becomes an enabled admin.
// It returns false when an admin already exists and nothing was done.
func (s *Service) EnsureAdmin(email, name string) (bool, error) {
	var admins int64
	if err := s.db.Model(&User{}).Where("role = ?", RoleAdmin).Count(&admins).Error; err != nil {
		return false, err
	}
	if admins > 0 {
		return false, nil
	}

	admin := &User{
		ID:      internal.NewID(),
		Email:   NormalizeEmail(email),
		Name:    strings.TrimSpace(name),
		Role:    RoleAdmin,
		Enabled: true,
	}
	err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "email"}},
		DoUpdates: clause.Assignments(map[string]any{"role": RoleAdmin, "enabled": true, "updated_at": time.Now()}),
	}).Create(admin).Error
	return err == nil, err
}

func (s *Service) first(query string, arg any) (*User, error) {
	var u User
	err := s.db.Where(query, arg).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
