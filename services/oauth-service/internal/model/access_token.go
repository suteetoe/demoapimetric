package model

import (
	"time"

	"gorm.io/gorm"
)

// AccessToken represents an OAuth2 access token
type AccessToken struct {
	ID            string         `gorm:"primaryKey" json:"id"`
	Token         string         `json:"-"` // Never expose the actual token in JSON responses
	ClientID      string         `json:"client_id"`
	UserID        *uint          `json:"user_id,omitempty"`
	TenantID      *uint          `json:"tenant_id,omitempty"`
	Scopes        string         `json:"scopes"`
	ExpiresAt     time.Time      `json:"expires_at"`
	Revoked       bool           `json:"revoked" gorm:"default:false"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:AccessTokenID" json:"-"`
}

// BeforeCreate hook will be called before creating a new AccessToken record
func (t *AccessToken) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == "" {
		t.ID = generateSecureID("tok_")
	}
	if t.Token == "" {
		t.Token = generateSecureToken()
	}
	return nil
}

// IsExpired checks if the token is expired
func (t *AccessToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsValid checks if the token is valid (not expired and not revoked)
func (t *AccessToken) IsValid() bool {
	return !t.Revoked && !t.IsExpired()
}
