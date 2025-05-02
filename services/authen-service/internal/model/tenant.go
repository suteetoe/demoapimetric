package model

import (
	"time"

	"gorm.io/gorm"
)

// Tenant represents the tenant model stored in the database
// This is the core of our multi-tenant architecture
type Tenant struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"type:varchar(100);uniqueIndex"`
	Description string         `json:"description" gorm:"type:text"`
	OwnerID     uint           `json:"owner_id" gorm:"index;not null"`
	Active      bool           `json:"active" gorm:"default:true"`
	Settings    string         `json:"settings" gorm:"type:jsonb"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}
