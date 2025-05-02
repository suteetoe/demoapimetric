package model

import (
	"time"

	"gorm.io/gorm"
)

// User represents the user model stored in the database
type User struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Email       string         `json:"email" gorm:"type:varchar(100);uniqueIndex"`
	Password    string         `json:"-" gorm:"type:varchar(255)"`
	FirstName   string         `json:"first_name,omitempty" gorm:"type:varchar(50)"`
	LastName    string         `json:"last_name,omitempty" gorm:"type:varchar(50)"`
	PhoneNumber string         `json:"phone_number,omitempty" gorm:"type:varchar(20)"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	UserTenants []UserTenant `json:"user_tenants,omitempty" gorm:"foreignKey:UserID"`
}
