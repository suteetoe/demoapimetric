package model

import (
	"time"

	"gorm.io/gorm"
)

// Client represents an OAuth client application
type Client struct {
	ID           string         `gorm:"primaryKey" json:"id"`
	Secret       string         `json:"-"` // Never expose the secret in JSON responses
	Name         string         `json:"name"`
	RedirectURIs string         `json:"redirect_uris"` // Comma-separated list of allowed redirect URIs
	Grants       string         `json:"grants"`        // Comma-separated list of allowed grant types
	Scopes       string         `json:"scopes"`        // Comma-separated list of allowed scopes
	UserID       *uint          `json:"user_id,omitempty"`
	TenantID     *uint          `json:"tenant_id,omitempty"`
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate hook will be called before creating a new Client record
func (c *Client) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == "" {
		c.ID = generateSecureID("cli_")
	}
	return nil
}
