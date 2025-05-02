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

// CategoryRequest defines the structure for category creation/update requests
type CategoryRequest struct {
	Name     string `json:"name" validate:"required"`
	TenantID uint   `json:"tenant_id" validate:"required"`
}

// ListCategories retrieves all product categories for a specific tenant
func ListCategories(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Listing categories")

	// Extract tenant ID from context (set by auth middleware)
	tenantID, ok := c.Get("tenant_id").(uint)
	if !ok {
		log.Warn("Missing tenant_id in context")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "tenant_id is required",
		})
	}

	log.Info("Filtering categories by tenant", zap.Uint("tenant_id", tenantID))

	var categories []model.ProductCategory
	result := database.GetDB().Where("tenant_id = ?", tenantID).Find(&categories)
	if result.Error != nil {
		log.Error("Failed to retrieve categories",
			zap.Error(result.Error),
			zap.Uint("tenant_id", tenantID))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to retrieve categories",
		})
	}

	log.Info("Categories retrieved successfully",
		zap.Int("count", len(categories)),
		zap.Uint("tenant_id", tenantID))
	return c.JSON(http.StatusOK, categories)
}

// GetCategory retrieves a specific category by ID
func GetCategory(c echo.Context) error {
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

	log.Info("Getting category by ID",
		zap.String("category_id", id),
		zap.Uint("tenant_id", tenantID))

	var category model.ProductCategory
	result := database.GetDB().Where("id = ? AND tenant_id = ?", id, tenantID).First(&category)
	if result.Error != nil {
		log.Error("Category not found or does not belong to tenant",
			zap.String("category_id", id),
			zap.Uint("tenant_id", tenantID),
			zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Category not found",
		})
	}

	log.Info("Category retrieved successfully",
		zap.String("category_id", id),
		zap.String("category_name", category.Name),
		zap.Uint("tenant_id", category.TenantID))
	return c.JSON(http.StatusOK, category)
}

// CreateCategory adds a new product category
func CreateCategory(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Creating new category")

	var req CategoryRequest
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
	// This ensures users can't create categories for other tenants
	req.TenantID = tenantID

	log.Info("Category creation request",
		zap.String("name", req.Name),
		zap.Uint("tenant_id", req.TenantID))

	// Check if category with same name exists in the same tenant
	var count int64
	database.GetDB().Model(&model.ProductCategory{}).
		Where("name = ? AND tenant_id = ?", req.Name, req.TenantID).
		Count(&count)
	if count > 0 {
		log.Warn("Category with this name already exists for this tenant",
			zap.String("name", req.Name),
			zap.Uint("tenant_id", req.TenantID))
		return c.JSON(http.StatusConflict, echo.Map{
			"error": "Category with this name already exists for this tenant",
		})
	}

	category := model.ProductCategory{
		Name:     req.Name,
		TenantID: req.TenantID,
	}

	result := database.GetDB().Create(&category)
	if result.Error != nil {
		log.Error("Failed to create category",
			zap.String("name", req.Name),
			zap.Uint("tenant_id", req.TenantID),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to create category",
		})
	}

	log.Info("Category created successfully",
		zap.String("category_id", strconv.FormatUint(uint64(category.ID), 10)),
		zap.String("name", category.Name),
		zap.Uint("tenant_id", category.TenantID))
	return c.JSON(http.StatusCreated, category)
}

// UpdateCategory updates an existing product category
func UpdateCategory(c echo.Context) error {
	log := logger.FromContext(c)
	id := c.Param("id")
	log.Info("Updating category", zap.String("category_id", id))

	var req CategoryRequest
	if err := c.Bind(&req); err != nil {
		log.Error("Invalid request data",
			zap.String("category_id", id),
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
	// This ensures users can't update categories for other tenants
	req.TenantID = tenantID

	// Find existing category and validate tenant ownership
	var category model.ProductCategory
	result := database.GetDB().Where("id = ?", id).First(&category)
	if result.Error != nil {
		log.Error("Category not found",
			zap.String("category_id", id),
			zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Category not found",
		})
	}

	// Ensure category belongs to the tenant in JWT token
	if category.TenantID != tenantID {
		log.Warn("Unauthorized attempt to update category from different tenant",
			zap.String("category_id", id),
			zap.Uint("category_tenant", category.TenantID),
			zap.Uint("request_tenant", tenantID))
		return c.JSON(http.StatusForbidden, echo.Map{
			"error": "You don't have permission to update this category",
		})
	}

	oldName := category.Name
	// Check if name is changed and if new name already exists within the same tenant
	if req.Name != category.Name {
		log.Info("Category name change requested",
			zap.String("category_id", id),
			zap.String("old_name", oldName),
			zap.String("new_name", req.Name))

		var count int64
		database.GetDB().Model(&model.ProductCategory{}).
			Where("name = ? AND id != ? AND tenant_id = ?", req.Name, id, tenantID).
			Count(&count)
		if count > 0 {
			log.Warn("Category with this name already exists for this tenant",
				zap.String("name", req.Name),
				zap.Uint("tenant_id", tenantID))
			return c.JSON(http.StatusConflict, echo.Map{
				"error": "Category with this name already exists for this tenant",
			})
		}
	}

	// Update fields
	category.Name = req.Name
	// TenantID remains unchanged - can't change tenant ownership

	result = database.GetDB().Save(&category)
	if result.Error != nil {
		log.Error("Failed to update category",
			zap.String("category_id", id),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to update category",
		})
	}

	log.Info("Category updated successfully",
		zap.String("category_id", id),
		zap.String("old_name", oldName),
		zap.String("new_name", category.Name),
		zap.Uint("tenant_id", category.TenantID))
	return c.JSON(http.StatusOK, category)
}

// DeleteCategory handles deleting a product category (soft delete)
func DeleteCategory(c echo.Context) error {
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

	log.Info("Deleting category",
		zap.String("category_id", id),
		zap.Uint("tenant_id", tenantID))

	// First verify tenant ownership of the category
	var category model.ProductCategory
	preResult := database.GetDB().Where("id = ? AND tenant_id = ?", id, tenantID).First(&category)
	if preResult.Error != nil {
		log.Warn("Category not found or does not belong to tenant",
			zap.String("category_id", id),
			zap.Uint("tenant_id", tenantID))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Category not found",
		})
	}

	// Check if any products from this tenant are using this category
	var count int64
	database.GetDB().Model(&model.Product{}).
		Where("category_id = ? AND tenant_id = ?", id, tenantID).
		Count(&count)
	if count > 0 {
		log.Warn("Cannot delete category that is being used by products",
			zap.String("category_id", id),
			zap.Uint("tenant_id", category.TenantID),
			zap.Int64("product_count", count))
		return c.JSON(http.StatusConflict, echo.Map{
			"error": "Cannot delete category that is being used by products",
		})
	}

	// Proceed with deletion
	result := database.GetDB().Delete(&category)
	if result.Error != nil {
		log.Error("Failed to delete category",
			zap.String("category_id", id),
			zap.Uint("tenant_id", category.TenantID),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to delete category",
		})
	}

	log.Info("Category deleted successfully",
		zap.String("category_id", id),
		zap.Uint("tenant_id", category.TenantID),
		zap.Int64("rows_affected", result.RowsAffected))
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Category deleted successfully",
	})
}
