package model

import (
	"time"

	"gorm.io/gorm"
)

// MerchantUser represents the association between merchants and users
// This allows users to be associated with multiple merchants (tenants)
type MerchantUser struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	MerchantID uint           `json:"merchant_id" gorm:"index;not null"`
	UserID     uint           `json:"user_id" gorm:"index;not null"`
	Role       string         `json:"role" gorm:"type:varchar(50);not null;default:'member'"` // Role within merchant: 'owner', 'admin', 'member', etc.
	Active     bool           `json:"active" gorm:"default:true"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations (optional for GORM to preload)
	Merchant Merchant `json:"merchant,omitempty" gorm:"foreignKey:MerchantID"`
}
