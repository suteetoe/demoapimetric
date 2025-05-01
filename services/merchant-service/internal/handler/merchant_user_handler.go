package handler

import (
	"merchant-service/internal/model"
	"merchant-service/pkg/database"
	"merchant-service/pkg/logger"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// AddUserToMerchant associates a user with a merchant
func AddUserToMerchant(c echo.Context) error {
	log := logger.FromContext(c)

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Get merchant ID from URL parameter
	merchantID, err := strconv.ParseUint(c.Param("merchant_id"), 10, 32)
	if err != nil {
		log.Error("Invalid merchant ID", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid merchant ID"})
	}

	// Parse request
	var req struct {
		UserID uint   `json:"user_id"`
		Role   string `json:"role"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	if req.UserID == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id is required"})
	}

	// Default role if not provided
	if req.Role == "" {
		req.Role = "member"
	}

	// Check if user has permission to add users to this merchant
	// Only the owner or admin can add users
	var merchant model.Merchant
	if result := database.GetDB().First(&merchant, merchantID); result.Error != nil {
		log.Error("Merchant not found", zap.Uint64("id", merchantID), zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{"error": "merchant not found"})
	}

	// Check if user is the owner of the merchant
	if merchant.OwnerID != userID {
		// Or check if user is an admin of the merchant
		var adminUser model.MerchantUser
		if result := database.GetDB().Where("merchant_id = ? AND user_id = ? AND role = 'admin'", merchantID, userID).First(&adminUser); result.Error != nil {
			log.Warn("Unauthorized attempt to add user to merchant",
				zap.Uint("requesting_user_id", userID),
				zap.Uint64("merchant_id", merchantID))
			return c.JSON(http.StatusForbidden, echo.Map{"error": "access denied"})
		}
	}

	// Check if user is already associated with merchant
	var existingAssociation model.MerchantUser
	result := database.GetDB().Where("merchant_id = ? AND user_id = ?", merchantID, req.UserID).First(&existingAssociation)
	if result.Error == nil {
		// User is already associated with this merchant, update their role if different
		if existingAssociation.Role != req.Role {
			existingAssociation.Role = req.Role
			if updateResult := database.GetDB().Save(&existingAssociation); updateResult.Error != nil {
				log.Error("Failed to update merchant user role", zap.Error(updateResult.Error))
				return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update user role"})
			}
			return c.JSON(http.StatusOK, echo.Map{
				"message": "User role updated successfully",
				"data":    existingAssociation,
			})
		}
		return c.JSON(http.StatusOK, echo.Map{
			"message": "User already associated with merchant",
			"data":    existingAssociation,
		})
	}

	// Create new association
	merchantUser := model.MerchantUser{
		MerchantID: uint(merchantID),
		UserID:     req.UserID,
		Role:       req.Role,
		Active:     true,
	}

	if result := database.GetDB().Create(&merchantUser); result.Error != nil {
		log.Error("Failed to associate user with merchant", zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to associate user with merchant"})
	}

	log.Info("User added to merchant",
		zap.Uint("merchant_id", uint(merchantID)),
		zap.Uint("user_id", req.UserID),
		zap.String("role", req.Role))

	return c.JSON(http.StatusCreated, echo.Map{
		"message": "User successfully associated with merchant",
		"data":    merchantUser,
	})
}

// RemoveUserFromMerchant removes a user from a merchant
func RemoveUserFromMerchant(c echo.Context) error {
	log := logger.FromContext(c)

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Get merchant ID from URL parameter
	merchantID, err := strconv.ParseUint(c.Param("merchant_id"), 10, 32)
	if err != nil {
		log.Error("Invalid merchant ID", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid merchant ID"})
	}

	// Get user ID to remove from URL parameter
	targetUserID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		log.Error("Invalid user ID", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user ID"})
	}

	// Check if user has permission to remove users from this merchant
	var merchant model.Merchant
	if result := database.GetDB().First(&merchant, merchantID); result.Error != nil {
		log.Error("Merchant not found", zap.Uint64("id", merchantID), zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{"error": "merchant not found"})
	}

	// Check if user is the owner of the merchant
	if merchant.OwnerID != userID {
		// Or check if user is an admin of the merchant
		var adminUser model.MerchantUser
		if result := database.GetDB().Where("merchant_id = ? AND user_id = ? AND role = 'admin'", merchantID, userID).First(&adminUser); result.Error != nil {
			log.Warn("Unauthorized attempt to remove user from merchant",
				zap.Uint("requesting_user_id", userID),
				zap.Uint64("merchant_id", merchantID))
			return c.JSON(http.StatusForbidden, echo.Map{"error": "access denied"})
		}
	}

	// Don't allow removing the owner
	if uint(targetUserID) == merchant.OwnerID {
		return c.JSON(http.StatusForbidden, echo.Map{"error": "cannot remove merchant owner"})
	}

	// Remove the association
	result := database.GetDB().Where("merchant_id = ? AND user_id = ?", merchantID, targetUserID).Delete(&model.MerchantUser{})
	if result.Error != nil {
		log.Error("Failed to remove user from merchant", zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to remove user from merchant"})
	}

	if result.RowsAffected == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found in this merchant"})
	}

	log.Info("User removed from merchant",
		zap.Uint("merchant_id", uint(merchantID)),
		zap.Uint64("user_id", targetUserID))

	return c.JSON(http.StatusOK, echo.Map{
		"message": "User successfully removed from merchant",
	})
}

// ListMerchantUsers retrieves all users associated with a merchant
func ListMerchantUsers(c echo.Context) error {
	log := logger.FromContext(c)

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Get merchant ID from URL parameter
	merchantID, err := strconv.ParseUint(c.Param("merchant_id"), 10, 32)
	if err != nil {
		log.Error("Invalid merchant ID", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid merchant ID"})
	}

	// Check if user has access to this merchant
	var merchant model.Merchant
	if result := database.GetDB().First(&merchant, merchantID); result.Error != nil {
		log.Error("Merchant not found", zap.Uint64("id", merchantID), zap.Error(result.Error))
		return c.JSON(http.StatusNotFound, echo.Map{"error": "merchant not found"})
	}

	// Check if user has access to view merchant users
	// Either they are the owner, an admin, or a member of the merchant
	hasAccess := merchant.OwnerID == userID

	if !hasAccess {
		var merchantUser model.MerchantUser
		if result := database.GetDB().Where("merchant_id = ? AND user_id = ?", merchantID, userID).First(&merchantUser); result.Error != nil {
			log.Warn("Unauthorized attempt to view merchant users",
				zap.Uint("requesting_user_id", userID),
				zap.Uint64("merchant_id", merchantID))
			return c.JSON(http.StatusForbidden, echo.Map{"error": "access denied"})
		}
	}

	// Retrieve all users for this merchant
	var merchantUsers []model.MerchantUser
	if result := database.GetDB().Where("merchant_id = ?", merchantID).Find(&merchantUsers); result.Error != nil {
		log.Error("Failed to retrieve merchant users", zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to retrieve merchant users"})
	}

	return c.JSON(http.StatusOK, merchantUsers)
}

// GetUserMerchants retrieves all merchants a user belongs to
func GetUserMerchants(c echo.Context) error {
	log := logger.FromContext(c)

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Get merchants where user is a member
	var merchantUsers []model.MerchantUser
	if result := database.GetDB().Where("user_id = ?", userID).Find(&merchantUsers); result.Error != nil {
		log.Error("Failed to retrieve user's merchants", zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to retrieve user's merchants"})
	}

	// If no merchants found, return empty array
	if len(merchantUsers) == 0 {
		return c.JSON(http.StatusOK, []model.MerchantUser{})
	}

	// Extract merchant IDs
	var merchantIDs []uint
	for _, mu := range merchantUsers {
		merchantIDs = append(merchantIDs, mu.MerchantID)
	}

	// Get merchant details
	var merchants []model.Merchant
	if result := database.GetDB().Where("id IN ?", merchantIDs).Find(&merchants); result.Error != nil {
		log.Error("Failed to retrieve merchant details", zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to retrieve merchant details"})
	}

	// Also get merchants where user is the owner
	var ownedMerchants []model.Merchant
	if result := database.GetDB().Where("owner_id = ?", userID).Find(&ownedMerchants); result.Error != nil {
		log.Error("Failed to retrieve owned merchants", zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to retrieve owned merchants"})
	}

	// Combine results (avoid duplicates)
	merchantMap := make(map[uint]model.Merchant)
	for _, m := range merchants {
		merchantMap[m.ID] = m
	}
	for _, m := range ownedMerchants {
		merchantMap[m.ID] = m
	}

	var result []model.Merchant
	for _, m := range merchantMap {
		result = append(result, m)
	}

	return c.JSON(http.StatusOK, result)
}
