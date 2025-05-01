package model

import (
	"time"

	"gorm.io/gorm"
)

// Merchant represents the merchant model stored in the database
type Merchant struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"type:varchar(100);not null"`
	Description string         `json:"description" gorm:"type:text"`
	OwnerID     uint           `json:"owner_id" gorm:"index;not null"` // Reference to the User ID who created this merchant
	Active      bool           `json:"active" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	MerchantUsers []MerchantUser `json:"merchant_users,omitempty" gorm:"foreignKey:MerchantID"`
}
