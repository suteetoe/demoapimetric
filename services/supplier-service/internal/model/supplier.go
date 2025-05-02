package model

import (
	"time"

	"gorm.io/gorm"
)

// Supplier represents the supplier model stored in the database
type Supplier struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	TenantID      uint           `json:"tenant_id" gorm:"index;not null;comment:'Tenant this supplier belongs to'"` // For multi-tenancy
	Name          string         `json:"name" gorm:"type:varchar(100);index;not null"`
	Code          string         `json:"code" gorm:"type:varchar(50);index;uniqueIndex:idx_tenant_code"` // Unique per tenant
	ContactPerson string         `json:"contact_person" gorm:"type:varchar(100)"`
	Email         string         `json:"email" gorm:"type:varchar(100)"`
	Phone         string         `json:"phone" gorm:"type:varchar(20)"`
	Address       string         `json:"address" gorm:"type:text"`
	City          string         `json:"city" gorm:"type:varchar(50)"`
	State         string         `json:"state" gorm:"type:varchar(50)"`
	Country       string         `json:"country" gorm:"type:varchar(50)"`
	PostalCode    string         `json:"postal_code" gorm:"type:varchar(20)"`
	TaxID         string         `json:"tax_id" gorm:"type:varchar(50)"`
	PaymentTerms  string         `json:"payment_terms" gorm:"type:varchar(100)"`
	Notes         string         `json:"notes" gorm:"type:text"`
	IsActive      bool           `json:"is_active" gorm:"default:true"`
	Rating        int            `json:"rating" gorm:"type:int;default:0"`
	CreatedBy     uint           `json:"created_by" gorm:"index"`
	UpdatedBy     uint           `json:"updated_by"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}
