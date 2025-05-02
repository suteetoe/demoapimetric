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
	Name string `json:"name" validate:"required"`
}

// ListCategories retrieves all product categories
func ListCategories(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Listing all categories")

	var categories []model.ProductCategory
	result := database.GetDB().Find(&categories)
	if result.Error != nil {
		log.Error("Failed to retrieve categories",
			zap.Error(result.Error),
			zap.Int("count", len(categories)))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to retrieve categories",
		})
	}

	log.Info("Categories retrieved successfully", zap.Int("count", len(categories)))
	return c.JSON(http.StatusOK, categories)
}

// GetCategory retrieves a specific category by ID
func GetCategory(c echo.Context) error {
	log := logger.FromContext(c)
	id := c.Param("id")
	log.Info("Getting category by ID", zap.String("category_id", id))

	var category model.ProductCategory
	result := database.GetDB().First(&category, id)
	if result.Error != nil {
		log.Error("Category not found",
			zap.String("category_id", id),
			zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Category not found",
		})
	}

	log.Info("Category retrieved successfully",
		zap.String("category_id", id),
		zap.String("category_name", category.Name))
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

	log.Info("Category creation request", zap.String("name", req.Name))

	// Check if category with same name exists
	var count int64
	database.GetDB().Model(&model.ProductCategory{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		log.Warn("Category with this name already exists", zap.String("name", req.Name))
		return c.JSON(http.StatusConflict, echo.Map{
			"error": "Category with this name already exists",
		})
	}

	category := model.ProductCategory{
		Name: req.Name,
	}

	result := database.GetDB().Create(&category)
	if result.Error != nil {
		log.Error("Failed to create category",
			zap.String("name", req.Name),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to create category",
		})
	}

	log.Info("Category created successfully",
		zap.String("category_id", strconv.FormatUint(uint64(category.ID), 10)),
		zap.String("name", category.Name))
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

	// Find existing category
	var category model.ProductCategory
	result := database.GetDB().First(&category, id)
	if result.Error != nil {
		log.Error("Category not found",
			zap.String("category_id", id),
			zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Category not found",
		})
	}

	oldName := category.Name
	// Check if name is changed and if new name already exists
	if req.Name != category.Name {
		log.Info("Category name change requested",
			zap.String("category_id", id),
			zap.String("old_name", oldName),
			zap.String("new_name", req.Name))

		var count int64
		database.GetDB().Model(&model.ProductCategory{}).Where("name = ? AND id != ?", req.Name, id).Count(&count)
		if count > 0 {
			log.Warn("Category with this name already exists",
				zap.String("name", req.Name))
			return c.JSON(http.StatusConflict, echo.Map{
				"error": "Category with this name already exists",
			})
		}
	}

	// Update fields
	category.Name = req.Name

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
		zap.String("new_name", category.Name))
	return c.JSON(http.StatusOK, category)
}

// DeleteCategory handles deleting a product category (soft delete)
func DeleteCategory(c echo.Context) error {
	log := logger.FromContext(c)
	id := c.Param("id")
	log.Info("Deleting category", zap.String("category_id", id))

	// Check if any products are using this category
	var count int64
	database.GetDB().Model(&model.Product{}).Where("category_id = ?", id).Count(&count)
	if count > 0 {
		log.Warn("Cannot delete category that is being used by products",
			zap.String("category_id", id),
			zap.Int64("product_count", count))
		return c.JSON(http.StatusConflict, echo.Map{
			"error": "Cannot delete category that is being used by products",
		})
	}

	result := database.GetDB().Delete(&model.ProductCategory{}, id)
	if result.Error != nil {
		log.Error("Failed to delete category",
			zap.String("category_id", id),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to delete category",
		})
	}

	if result.RowsAffected == 0 {
		log.Warn("Category not found for deletion",
			zap.String("category_id", id))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error": "Category not found",
		})
	}

	log.Info("Category deleted successfully",
		zap.String("category_id", id),
		zap.Int64("rows_affected", result.RowsAffected))
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Category deleted successfully",
	})
}
