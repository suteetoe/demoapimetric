package model

import (
	"time"

	"gorm.io/gorm"
)

// User represents the user model stored in the database
type User struct {
	ID         uint           `json:"id" gorm:"primaryKey"`
	Email      string         `json:"email" gorm:"type:varchar(100);uniqueIndex"`
	Password   string         `json:"-" gorm:"type:varchar(255)"`
	MerchantID *uint          `json:"merchant_id,omitempty" gorm:"index"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}
