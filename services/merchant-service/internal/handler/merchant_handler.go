package handler

import (
	"merchant-service/internal/model"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/suteetoe/gomicro/database"
	"github.com/suteetoe/gomicro/jwtutil"
	"github.com/suteetoe/gomicro/logger"
	"go.uber.org/zap"
)

// CreateMerchant handles merchant creation
func CreateMerchant(c echo.Context) error {
	log := logger.FromEcho(c)

	// Get user ID and tenant ID from context (set by AuthMiddleware)
	claims, ok := c.Get("user").(*jwtutil.UserClaims)
	if !ok {
		log.Error("Failed to get user claims from context")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}
	userID := claims.UserID

	// Get tenant ID from claims
	if claims.TenantID == nil {
		log.Error("Tenant ID is missing from user claims")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "tenant context required"})
	}
	tenantID := *claims.TenantID

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

	// Create merchant with tenant ID
	merchant := model.Merchant{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID,
		TenantID:    tenantID,
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
		zap.Uint("owner_id", merchant.OwnerID),
		zap.Uint("tenant_id", merchant.TenantID))

	return c.JSON(http.StatusCreated, echo.Map{
		"message":  "Merchant created successfully",
		"merchant": merchant,
	})
}

// GetMerchant retrieves merchant details
func GetMerchant(c echo.Context) error {
	log := logger.FromEcho(c)

	// Get user ID and tenant ID from context (set by AuthMiddleware)
	claims, ok := c.Get("user").(*jwtutil.UserClaims)
	if !ok {
		log.Error("Failed to get user claims from context")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}
	userID := claims.UserID

	// Get tenant ID from claims
	if claims.TenantID == nil {
		log.Error("Tenant ID is missing from user claims")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "tenant context required"})
	}
	tenantID := *claims.TenantID

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

	// Check if merchant belongs to the same tenant
	if merchant.TenantID != tenantID {
		log.Warn("Cross-tenant merchant access attempt",
			zap.Uint("requesting_tenant_id", tenantID),
			zap.Uint("merchant_tenant_id", merchant.TenantID))
		return c.JSON(http.StatusForbidden, echo.Map{"error": "access denied"})
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
	log := logger.FromEcho(c)

	// Get user ID and tenant ID from context (set by AuthMiddleware)
	claims, ok := c.Get("user").(*jwtutil.UserClaims)
	if !ok {
		log.Error("Failed to get user claims from context")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}
	userID := claims.UserID

	// Get tenant ID from claims
	if claims.TenantID == nil {
		log.Error("Tenant ID is missing from user claims")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "tenant context required"})
	}
	tenantID := *claims.TenantID

	// Retrieve merchants from database with tenant isolation
	var merchants []model.Merchant
	if result := database.GetDB().Where("owner_id = ? AND tenant_id = ?", userID, tenantID).Find(&merchants); result.Error != nil {
		log.Error("Failed to retrieve merchants",
			zap.Uint("owner_id", userID),
			zap.Uint("tenant_id", tenantID),
			zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to retrieve merchants"})
	}

	return c.JSON(http.StatusOK, merchants)
}
