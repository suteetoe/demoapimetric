package handler

import (
	"merchant-service/internal/model"
	"merchant-service/pkg/database"
	"merchant-service/pkg/logger"
	"merchant-service/prometheus"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// CreateMerchant handles merchant creation
func CreateMerchant(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.CreateMerchantCounter.Inc()

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Parse request
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse merchant creation request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	if req.Name == "" {
		log.Error("Invalid merchant data", zap.String("name", req.Name))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "name is required"})
	}

	// Create merchant
	merchant := model.Merchant{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID,
		Active:      true,
	}

	// Save to database
	if result := database.GetDB().Create(&merchant); result.Error != nil {
		log.Error("Failed to create merchant", zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "merchant creation failed"})
	}

	log.Info("Merchant created",
		zap.String("name", merchant.Name),
		zap.Uint("id", merchant.ID),
		zap.Uint("owner_id", merchant.OwnerID))

	return c.JSON(http.StatusCreated, echo.Map{
		"message":  "Merchant created successfully",
		"merchant": merchant,
	})
}

// GetMerchant retrieves merchant details
func GetMerchant(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.GetMerchantCounter.Inc()

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Get ID from path parameter
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		log.Error("Invalid merchant ID", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid merchant ID"})
	}

	// Retrieve merchant from database
	var merchant model.Merchant
	if result := database.GetDB().First(&merchant, id); result.Error != nil {
		log.Error("Merchant not found", zap.Uint64("id", id), zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{"error": "merchant not found"})
	}

	// Check if user owns this merchant
	if merchant.OwnerID != userID {
		log.Warn("Unauthorized merchant access attempt",
			zap.Uint("requesting_user_id", userID),
			zap.Uint("merchant_owner_id", merchant.OwnerID))
		return c.JSON(http.StatusForbidden, echo.Map{"error": "access denied"})
	}

	return c.JSON(http.StatusOK, merchant)
}

// ListMerchantsByOwner retrieves all merchants associated with an owner
func ListMerchantsByOwner(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.ListMerchantsCounter.Inc()

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Retrieve merchants from database
	var merchants []model.Merchant
	if result := database.GetDB().Where("owner_id = ?", userID).Find(&merchants); result.Error != nil {
		log.Error("Failed to retrieve merchants", zap.Uint("owner_id", userID), zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to retrieve merchants"})
	}

	return c.JSON(http.StatusOK, merchants)
}

// MetricsHandler exposes Prometheus metrics
func MetricsHandler(c echo.Context) error {
	handler := prometheus.GetPrometheusHandler()
	handler.ServeHTTP(c.Response(), c.Request())
	return nil
}
