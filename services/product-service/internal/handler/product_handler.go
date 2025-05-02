package handler

import (
	"net/http"
	"product-service/internal/model"
	"product-service/pkg/database"
	"product-service/pkg/logger"
	"strconv"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// ProductRequest defines the structure for product creation/update requests
type ProductRequest struct {
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description"`
	SKU         string  `json:"sku" validate:"required"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	Stock       int     `json:"stock"`
	CategoryID  uint    `json:"category_id"`
	TenantID    uint    `json:"tenant_id" validate:"required"`
	IsActive    bool    `json:"is_active"`
}

// ListProducts handles retrieving all products with optional filtering
func ListProducts(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Listing products with filters")

	db := database.GetDB()
	var products []model.Product

	// Extract tenant ID from context (set by auth middleware)
	tenantID, ok := c.Get("tenant_id").(uint)
	if !ok {
		log.Warn("Missing tenant_id in context")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "tenant_id is required",
		})
	}

	// Handle query parameters for filtering
	query := db.Where("tenant_id = ?", tenantID)
	log.Info("Filtering products by tenant", zap.Uint("tenant_id", tenantID))

	// Filter by active status if specified
	isActive := c.QueryParam("is_active")
	if isActive != "" {
		active, err := strconv.ParseBool(isActive)
		if err == nil {
			query = query.Where("is_active = ?", active)
			log.Info("Filtering products by active status", zap.Bool("is_active", active))
		} else {
			log.Warn("Invalid is_active parameter", zap.String("value", isActive), zap.Error(err))
		}
	}

	// Filter by category if specified
	categoryID := c.QueryParam("category_id")
	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
		log.Info("Filtering products by category", zap.String("category_id", categoryID))
	}

	// Execute the query
	result := query.Find(&products)
	if result.Error != nil {
		log.Error("Failed to list products",
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to retrieve products",
		})
	}

	log.Info("Products retrieved successfully",
		zap.Int("count", len(products)),
		zap.Uint("tenant_id", tenantID))
	return c.JSON(http.StatusOK, products)
}

// GetProduct handles retrieving a single product by ID
func GetProduct(c echo.Context) error {
	log := logger.FromContext(c)
	id := c.Param("id")

	// Extract tenant ID from context (set by auth middleware)
	tenantID, ok := c.Get("tenant_id").(uint)
	if !ok {
		log.Warn("Missing tenant_id in context")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "tenant_id is required",
		})
	}

	log.Info("Getting product by ID",
		zap.String("product_id", id),
		zap.Uint("tenant_id", tenantID))

	var product model.Product
	result := database.GetDB().Where("id = ? AND tenant_id = ?", id, tenantID).First(&product)
	if result.Error != nil {
		log.Error("Product not found or does not belong to tenant",
			zap.String("product_id", id),
			zap.Uint("tenant_id", tenantID),
			zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Product not found",
		})
	}

	log.Info("Product retrieved successfully",
		zap.String("product_id", id),
		zap.String("product_name", product.Name),
		zap.String("product_sku", product.SKU),
		zap.Uint("tenant_id", product.TenantID))
	return c.JSON(http.StatusOK, product)
}

// CreateProduct handles creating a new product
func CreateProduct(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Creating new product")

	var req ProductRequest
	if err := c.Bind(&req); err != nil {
		log.Error("Invalid request data", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request data",
		})
	}

	// Extract tenant ID from context (set by auth middleware)
	tenantID, ok := c.Get("tenant_id").(uint)
	if !ok {
		log.Warn("Missing tenant_id in context")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "tenant_id is required",
		})
	}

	// Override the tenant ID in the request with the one from the JWT token
	// This ensures users can't create products for other tenants
	req.TenantID = tenantID

	log.Info("Product creation request",
		zap.String("name", req.Name),
		zap.String("sku", req.SKU),
		zap.Float64("price", req.Price),
		zap.Uint("category_id", req.CategoryID),
		zap.Uint("tenant_id", req.TenantID))

	// Check if product with SKU already exists for this tenant
	var count int64
	database.GetDB().Model(&model.Product{}).
		Where("sku = ? AND tenant_id = ?", req.SKU, req.TenantID).
		Count(&count)
	if count > 0 {
		log.Warn("Product with this SKU already exists for this tenant",
			zap.String("sku", req.SKU),
			zap.Uint("tenant_id", req.TenantID))
		return c.JSON(http.StatusConflict, echo.Map{
			"error": "Product with this SKU already exists for this tenant",
		})
	}

	// Create the product
	product := model.Product{
		Name:        req.Name,
		Description: req.Description,
		SKU:         req.SKU,
		Price:       req.Price,
		Stock:       req.Stock,
		CategoryID:  req.CategoryID,
		TenantID:    req.TenantID,
		IsActive:    req.IsActive,
	}

	result := database.GetDB().Create(&product)
	if result.Error != nil {
		log.Error("Failed to create product",
			zap.String("name", req.Name),
			zap.String("sku", req.SKU),
			zap.Uint("tenant_id", req.TenantID),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to create product",
		})
	}

	log.Info("Product created successfully",
		zap.String("product_id", strconv.FormatUint(uint64(product.ID), 10)),
		zap.String("name", product.Name),
		zap.String("sku", product.SKU),
		zap.Uint("tenant_id", product.TenantID))
	return c.JSON(http.StatusCreated, product)
}

// UpdateProduct handles updating an existing product
func UpdateProduct(c echo.Context) error {
	log := logger.FromContext(c)
	id := c.Param("id")
	log.Info("Updating product", zap.String("product_id", id))

	var req ProductRequest
	if err := c.Bind(&req); err != nil {
		log.Error("Invalid request data",
			zap.String("product_id", id),
			zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid request data",
		})
	}

	// Extract tenant ID from context (set by auth middleware)
	tenantID, ok := c.Get("tenant_id").(uint)
	if !ok {
		log.Warn("Missing tenant_id in context")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "tenant_id is required",
		})
	}

	// Override the tenant ID in the request with the one from the JWT token
	// This ensures users can't update products for other tenants
	req.TenantID = tenantID

	// Find existing product and validate tenant ownership
	var product model.Product
	result := database.GetDB().Where("id = ?", id).First(&product)
	if result.Error != nil {
		log.Error("Product not found for update",
			zap.String("product_id", id),
			zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Product not found",
		})
	}

	// Ensure product belongs to the tenant in JWT token
	if product.TenantID != tenantID {
		log.Warn("Unauthorized attempt to update product from different tenant",
			zap.String("product_id", id),
			zap.Uint("product_tenant", product.TenantID),
			zap.Uint("request_tenant", tenantID))
		return c.JSON(http.StatusForbidden, echo.Map{
			"error": "You don't have permission to update this product",
		})
	}

	oldSKU := product.SKU
	oldPrice := product.Price

	// Check if SKU is changed and if new SKU already exists within the same tenant
	if req.SKU != product.SKU {
		log.Info("Product SKU change requested",
			zap.String("product_id", id),
			zap.String("old_sku", oldSKU),
			zap.String("new_sku", req.SKU))

		var count int64
		database.GetDB().Model(&model.Product{}).
			Where("sku = ? AND id != ? AND tenant_id = ?", req.SKU, id, tenantID).
			Count(&count)
		if count > 0 {
			log.Warn("Product with this SKU already exists for this tenant",
				zap.String("sku", req.SKU),
				zap.Uint("tenant_id", tenantID))
			return c.JSON(http.StatusConflict, echo.Map{
				"error": "Product with this SKU already exists for this tenant",
			})
		}
	}

	// Update fields
	product.Name = req.Name
	product.Description = req.Description
	product.SKU = req.SKU
	product.Price = req.Price
	product.Stock = req.Stock
	product.CategoryID = req.CategoryID
	product.IsActive = req.IsActive
	// TenantID remains unchanged - can't change tenant ownership

	result = database.GetDB().Save(&product)
	if result.Error != nil {
		log.Error("Failed to update product",
			zap.String("product_id", id),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to update product",
		})
	}

	log.Info("Product updated successfully",
		zap.String("product_id", id),
		zap.String("name", product.Name),
		zap.String("old_sku", oldSKU),
		zap.String("new_sku", product.SKU),
		zap.Float64("old_price", oldPrice),
		zap.Float64("new_price", product.Price),
		zap.Uint("tenant_id", product.TenantID))
	return c.JSON(http.StatusOK, product)
}

// DeleteProduct handles deleting a product (soft delete)
func DeleteProduct(c echo.Context) error {
	log := logger.FromContext(c)
	id := c.Param("id")

	// Extract tenant ID from context (set by auth middleware)
	tenantID, ok := c.Get("tenant_id").(uint)
	if !ok {
		log.Warn("Missing tenant_id in context")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "tenant_id is required",
		})
	}

	log.Info("Deleting product",
		zap.String("product_id", id),
		zap.Uint("tenant_id", tenantID))

	// Get product details before deleting and verify tenant ownership
	var product model.Product
	preResult := database.GetDB().Where("id = ? AND tenant_id = ?", id, tenantID).First(&product)
	if preResult.Error != nil {
		log.Warn("Product not found or does not belong to tenant",
			zap.String("product_id", id),
			zap.Uint("tenant_id", tenantID),
			zap.Error(preResult.Error))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Product not found",
		})
	}

	log.Info("Found product to delete",
		zap.String("product_id", id),
		zap.String("name", product.Name),
		zap.String("sku", product.SKU),
		zap.Uint("tenant_id", product.TenantID))

	// Proceed with deletion
	result := database.GetDB().Delete(&product)
	if result.Error != nil {
		log.Error("Failed to delete product",
			zap.String("product_id", id),
			zap.Uint("tenant_id", product.TenantID),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to delete product",
		})
	}

	log.Info("Product deleted successfully",
		zap.String("product_id", id),
		zap.Uint("tenant_id", product.TenantID),
		zap.Int64("rows_affected", result.RowsAffected))
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Product deleted successfully",
	})
}
