package model

import (
	"time"

	"gorm.io/gorm"
)

// Product represents the product master data
type Product struct {
	ID          uint           `json:"id" gorm:"primarykey"`
	Name        string         `json:"name" gorm:"type:varchar(255);not null"`
	Description string         `json:"description" gorm:"type:text"`
	SKU         string         `json:"sku" gorm:"type:varchar(100);unique;not null"`
	Price       float64        `json:"price" gorm:"not null"`
	Stock       int            `json:"stock" gorm:"default:0"`
	CategoryID  uint           `json:"category_id"`
	TenantID    uint           `json:"tenant_id" gorm:"index;not null;comment:'Tenant this product belongs to'"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

// ProductCategory represents product categories
type ProductCategory struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	Name      string         `json:"name" gorm:"type:varchar(100);not null;unique"`
	TenantID  uint           `json:"tenant_id" gorm:"index;not null;comment:'Tenant this category belongs to'"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}
