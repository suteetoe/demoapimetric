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
	IsActive    bool    `json:"is_active"`
}

// ListProducts handles retrieving all products with optional filtering
func ListProducts(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Listing products with filters")

	db := database.GetDB()
	var products []model.Product

	// Handle query parameters for filtering
	query := db

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

	log.Info("Products retrieved successfully", zap.Int("count", len(products)))
	return c.JSON(http.StatusOK, products)
}

// GetProduct handles retrieving a single product by ID
func GetProduct(c echo.Context) error {
	log := logger.FromContext(c)
	id := c.Param("id")
	log.Info("Getting product by ID", zap.String("product_id", id))

	var product model.Product
	result := database.GetDB().First(&product, id)
	if result.Error != nil {
		log.Error("Product not found",
			zap.String("product_id", id),
			zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Product not found",
		})
	}

	log.Info("Product retrieved successfully",
		zap.String("product_id", id),
		zap.String("product_name", product.Name),
		zap.String("product_sku", product.SKU))
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

	log.Info("Product creation request",
		zap.String("name", req.Name),
		zap.String("sku", req.SKU),
		zap.Float64("price", req.Price),
		zap.Uint("category_id", req.CategoryID))

	// Check if product with SKU already exists
	var count int64
	database.GetDB().Model(&model.Product{}).Where("sku = ?", req.SKU).Count(&count)
	if count > 0 {
		log.Warn("Product with this SKU already exists", zap.String("sku", req.SKU))
		return c.JSON(http.StatusConflict, echo.Map{
			"error": "Product with this SKU already exists",
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
		IsActive:    req.IsActive,
	}

	result := database.GetDB().Create(&product)
	if result.Error != nil {
		log.Error("Failed to create product",
			zap.String("name", req.Name),
			zap.String("sku", req.SKU),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to create product",
		})
	}

	log.Info("Product created successfully",
		zap.String("product_id", strconv.FormatUint(uint64(product.ID), 10)),
		zap.String("name", product.Name),
		zap.String("sku", product.SKU))
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

	// Find existing product
	var product model.Product
	result := database.GetDB().First(&product, id)
	if result.Error != nil {
		log.Error("Product not found for update",
			zap.String("product_id", id),
			zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Product not found",
		})
	}

	oldSKU := product.SKU
	oldPrice := product.Price

	// Check if SKU is changed and if new SKU already exists
	if req.SKU != product.SKU {
		log.Info("Product SKU change requested",
			zap.String("product_id", id),
			zap.String("old_sku", oldSKU),
			zap.String("new_sku", req.SKU))

		var count int64
		database.GetDB().Model(&model.Product{}).Where("sku = ? AND id != ?", req.SKU, id).Count(&count)
		if count > 0 {
			log.Warn("Product with this SKU already exists",
				zap.String("sku", req.SKU))
			return c.JSON(http.StatusConflict, echo.Map{
				"error": "Product with this SKU already exists",
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
		zap.Float64("new_price", product.Price))
	return c.JSON(http.StatusOK, product)
}

// DeleteProduct handles deleting a product (soft delete)
func DeleteProduct(c echo.Context) error {
	log := logger.FromContext(c)
	id := c.Param("id")
	log.Info("Deleting product", zap.String("product_id", id))

	// Get product details before deleting
	var product model.Product
	preResult := database.GetDB().First(&product, id)
	if preResult.Error == nil {
		log.Info("Found product to delete",
			zap.String("product_id", id),
			zap.String("name", product.Name),
			zap.String("sku", product.SKU))
	}

	result := database.GetDB().Delete(&model.Product{}, id)
	if result.Error != nil {
		log.Error("Failed to delete product",
			zap.String("product_id", id),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to delete product",
		})
	}

	if result.RowsAffected == 0 {
		log.Warn("Product not found for deletion",
			zap.String("product_id", id))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Product not found",
		})
	}

	log.Info("Product deleted successfully",
		zap.String("product_id", id),
		zap.Int64("rows_affected", result.RowsAffected))
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Product deleted successfully",
	})
}
