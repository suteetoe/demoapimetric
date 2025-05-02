package model

import (
	"time"

	"gorm.io/gorm"
)

// UserTenant represents the association between users and tenants
// This enables multi-tenancy by allowing users to belong to multiple tenants
type UserTenant struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"index;not null"`
	TenantID  uint           `json:"tenant_id" gorm:"index;not null"`
	Role      string         `json:"role" gorm:"type:varchar(50);not null;default:'member'"` // Role within tenant: 'owner', 'admin', 'member', etc.
	IsDefault bool           `json:"is_default" gorm:"default:false"`                        // Whether this is the user's default tenant
	Active    bool           `json:"active" gorm:"default:true"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	User   User   `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Tenant Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`
}
