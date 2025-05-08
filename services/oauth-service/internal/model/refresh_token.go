package model

import (
	"time"

	"gorm.io/gorm"
)

// RefreshToken represents an OAuth2 refresh token
type RefreshToken struct {
	ID            string         `gorm:"primaryKey" json:"id"`
	Token         string         `json:"-"` // Never expose the actual token in JSON responses
	AccessTokenID string         `json:"access_token_id"`
	ClientID      string         `json:"client_id"`
	UserID        *uint          `json:"user_id,omitempty"`
	TenantID      *uint          `json:"tenant_id,omitempty"`
	ExpiresAt     time.Time      `json:"expires_at"`
	Revoked       bool           `json:"revoked" gorm:"default:false"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate hook will be called before creating a new RefreshToken record
func (t *RefreshToken) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = generateSecureID("ref_")
	}
	if t.Token == "" {
		t.Token = generateSecureToken()
	}
	return nil
}

// IsExpired checks if the token is expired
func (t *RefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsValid checks if the token is valid (not expired and not revoked)
func (t *RefreshToken) IsValid() bool {
	return !t.Revoked && !t.IsExpired()
}
