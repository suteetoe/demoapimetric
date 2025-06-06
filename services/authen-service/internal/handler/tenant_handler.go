package handler

import (
	"auth-service/internal/model"
	"auth-service/pkg/database"
	"auth-service/pkg/jwtutil"
	"auth-service/pkg/logger"
	"auth-service/prometheus"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// CreateTenant handles tenant creation
func CreateTenant(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Processing tenant creation request")
	prometheus.RecordTenantOperation("create")

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context",
			zap.String("remote_ip", c.RealIP()))
		prometheus.RecordAuthError("unauthorized_tenant_creation")
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"error":   "Authentication required",
			"message": "You must be authenticated to create a tenant",
		})
	}

	// Parse request
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Settings    string `json:"settings,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse tenant creation request",
			zap.Error(err),
			zap.Uint("user_id", userID),
			zap.String("remote_ip", c.RealIP()))
		prometheus.RecordAuthError("invalid_request")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   "Invalid request format",
			"message": "The request could not be processed due to invalid format",
		})
	}

	if req.Name == "" {
		log.Warn("Tenant creation attempt with missing name",
			zap.Uint("user_id", userID),
			zap.String("remote_ip", c.RealIP()))
		prometheus.RecordAuthError("incomplete_tenant_creation")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   "Missing required field",
			"message": "Tenant name is required",
		})
	}

	log.Info("Starting tenant creation process",
		zap.Uint("user_id", userID),
		zap.String("tenant_name", req.Name))

	// Track DB operations
	defer prometheus.TrackDBOperation("transaction")(time.Now())

	// Begin transaction
	tx := database.GetDB().Begin()
	if tx.Error != nil {
		log.Error("Failed to begin transaction",
			zap.Error(tx.Error),
			zap.Uint("user_id", userID),
			zap.String("tenant_name", req.Name))
		prometheus.RecordAuthError("database_error")
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   "Database error",
			"message": "Could not create tenant due to a database error",
		})
	}

	// Create tenant
	tenant := model.Tenant{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID,
		Settings:    req.Settings,
		Active:      true,
	}

	// Save tenant to database
	if result := tx.Create(&tenant); result.Error != nil {
		tx.Rollback()
		log.Error("Failed to create tenant",
			zap.Error(result.Error),
			zap.Uint("user_id", userID),
			zap.String("tenant_name", req.Name))
		prometheus.RecordAuthError("tenant_creation_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   "Tenant creation failed",
			"message": "The system could not create the tenant at this time",
		})
	}

	// Also create UserTenant association with owner role
	userTenant := model.UserTenant{
		UserID:    userID,
		TenantID:  tenant.ID,
		Role:      "owner",
		IsDefault: true, // Make this the default tenant for the user
		Active:    true,
	}

	if result := tx.Create(&userTenant); result.Error != nil {
		tx.Rollback()
		log.Error("Failed to create user-tenant association",
			zap.Error(result.Error),
			zap.Uint("user_id", userID),
			zap.Uint("tenant_id", tenant.ID))
		prometheus.RecordAuthError("tenant_association_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   "Tenant association failed",
			"message": "The system could not associate the user with the tenant",
		})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Error("Failed to commit transaction",
			zap.Error(err),
			zap.Uint("user_id", userID),
			zap.Uint("tenant_id", tenant.ID))
		prometheus.RecordAuthError("transaction_commit_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   "Transaction failed",
			"message": "The tenant creation process could not be completed",
		})
	}

	// Update active tenants metric
	prometheus.UpdateActiveTenants(1) // Increment by 1 since we just created a new tenant

	log.Info("Tenant created successfully",
		zap.String("name", tenant.Name),
		zap.Uint("tenant_id", tenant.ID),
		zap.Uint("owner_id", tenant.OwnerID))

	return c.JSON(http.StatusCreated, echo.Map{
		"message": "Tenant created successfully",
		"tenant": echo.Map{
			"id":          tenant.ID,
			"name":        tenant.Name,
			"description": tenant.Description,
			"owner_id":    tenant.OwnerID,
			"created_at":  tenant.CreatedAt,
		},
	})
}

// GetTenant retrieves tenant details
func GetTenant(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.RecordTenantOperation("access")

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		prometheus.RecordAuthError("unauthorized_tenant_access")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Get ID from path parameter
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		log.Error("Invalid tenant ID", zap.Error(err))
		prometheus.RecordAuthError("invalid_tenant_id")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid tenant ID"})
	}

	// Track DB operations
	defer prometheus.TrackDBOperation("query")(time.Now())

	// Retrieve tenant from database
	var tenant model.Tenant
	if result := database.GetDB().First(&tenant, id); result.Error != nil {
		log.Error("Tenant not found", zap.Uint64("id", id), zap.Error(result.Error))
		prometheus.RecordAuthError("tenant_not_found")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "tenant not found"})
	}

	// Verify user has access to this tenant
	var userTenant model.UserTenant
	result := database.GetDB().Where("user_id = ? AND tenant_id = ?", userID, id).First(&userTenant)
	if result.Error != nil && tenant.OwnerID != userID {
		log.Warn("Unauthorized tenant access attempt",
			zap.Uint("requesting_user_id", userID),
			zap.Uint("tenant_id", uint(id)))
		prometheus.RecordAuthError("tenant_access_denied")
		return c.JSON(http.StatusForbidden, echo.Map{"error": "access denied"})
	}

	return c.JSON(http.StatusOK, tenant)
}

// ListUserTenants retrieves all tenants associated with the authenticated user
func ListUserTenants(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.RecordTenantOperation("list")

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		prometheus.RecordAuthError("unauthorized_tenant_listing")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Track DB operations
	defer prometheus.TrackDBOperation("query")(time.Now())

	// Get user's tenants through UserTenant associations
	var userTenants []model.UserTenant
	if result := database.GetDB().Preload("Tenant").Where("user_id = ? AND active = ?", userID, true).Find(&userTenants); result.Error != nil {
		log.Error("Failed to retrieve user's tenants", zap.Error(result.Error))
		prometheus.RecordAuthError("tenant_retrieval_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to retrieve tenants"})
	}

	// Format response
	type TenantResponse struct {
		ID          uint      `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Role        string    `json:"role"`
		IsDefault   bool      `json:"is_default"`
		CreatedAt   time.Time `json:"created_at"`
	}

	var response []TenantResponse
	for _, ut := range userTenants {
		response = append(response, TenantResponse{
			ID:          ut.TenantID,
			Name:        ut.Tenant.Name,
			Description: ut.Tenant.Description,
			Role:        ut.Role,
			IsDefault:   ut.IsDefault,
			CreatedAt:   ut.Tenant.CreatedAt,
		})
	}

	return c.JSON(http.StatusOK, response)
}

// SwitchTenant generates a new token with a different tenant context
func SwitchTenant(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Processing tenant switch request")
	prometheus.RecordTenantOperation("switch")

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context",
			zap.String("remote_ip", c.RealIP()))
		prometheus.RecordAuthError("unauthorized_tenant_switch")
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"error":   "Authentication required",
			"message": "You must be authenticated to switch tenants",
		})
	}

	// Get email from context
	email, ok := c.Get("email").(string)
	if !ok {
		log.Error("Failed to get email from context",
			zap.Uint("user_id", userID),
			zap.String("remote_ip", c.RealIP()))
		prometheus.RecordAuthError("context_missing_email")
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   "Context error",
			"message": "Email missing from authentication context",
		})
	}

	// Parse request
	var req struct {
		TenantID uint `json:"tenant_id"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse tenant switch request",
			zap.Error(err),
			zap.Uint("user_id", userID),
			zap.String("remote_ip", c.RealIP()))
		prometheus.RecordAuthError("invalid_request")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   "Invalid request format",
			"message": "The tenant switch request could not be processed",
		})
	}

	if req.TenantID == 0 {
		log.Warn("Tenant switch attempt with invalid tenant ID",
			zap.Uint("user_id", userID),
			zap.String("remote_ip", c.RealIP()))
		prometheus.RecordAuthError("invalid_tenant_id")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   "Missing tenant ID",
			"message": "A valid tenant ID is required to switch tenants",
		})
	}

	log.Info("Verifying tenant access permissions",
		zap.Uint("user_id", userID),
		zap.Uint("tenant_id", req.TenantID))

	// Track DB operations
	defer prometheus.TrackDBOperation("query")(time.Now())

	// Verify the user has access to this tenant
	var userTenant model.UserTenant
	result := database.GetDB().Where("user_id = ? AND tenant_id = ? AND active = ?", userID, req.TenantID, true).First(&userTenant)
	if result.Error != nil {
		log.Warn("Unauthorized tenant switch attempt",
			zap.Uint("user_id", userID),
			zap.Uint("tenant_id", req.TenantID),
			zap.String("remote_ip", c.RealIP()),
			zap.Error(result.Error))
		prometheus.RecordAuthError("tenant_access_denied")
		return c.JSON(http.StatusForbidden, echo.Map{
			"error":   "Access denied",
			"message": "You do not have permission to access the requested tenant",
		})
	}

	// Get tenant name
	var tenant model.Tenant
	if result := database.GetDB().Select("name").First(&tenant, req.TenantID); result.Error != nil {
		log.Error("Tenant not found",
			zap.Uint("tenant_id", req.TenantID),
			zap.Uint("user_id", userID),
			zap.Error(result.Error))
		prometheus.RecordAuthError("tenant_not_found")
		return c.JSON(http.StatusNotFound, echo.Map{
			"error":   "Tenant not found",
			"message": "The requested tenant could not be found",
		})
	}

	// Generate new JWT token with tenant context
	tenantID := req.TenantID
	token, err := jwtutil.GenerateTokenWithTenant(email, userID, &tenantID, tenant.Name, userTenant.Role)
	if err != nil {
		log.Error("Failed to generate token",
			zap.Error(err),
			zap.Uint("user_id", userID),
			zap.Uint("tenant_id", req.TenantID))
		prometheus.RecordAuthError("token_generation_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   "Token generation failed",
			"message": "Unable to generate authentication token for this tenant",
		})
	}

	// Increment active tokens gauge
	prometheus.IncreaseActiveTokens()

	log.Info("User switched tenant successfully",
		zap.String("email", email),
		zap.Uint("user_id", userID),
		zap.Uint("tenant_id", req.TenantID),
		zap.String("tenant_name", tenant.Name),
		zap.String("role", userTenant.Role))

	return c.JSON(http.StatusOK, echo.Map{
		"message": "Tenant switched successfully",
		"token":   token,
		"tenant": map[string]interface{}{
			"id":   tenant.ID,
			"name": tenant.Name,
			"role": userTenant.Role,
		},
	})
}

// AddUserToTenant adds a user to a tenant
func AddUserToTenant(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.RecordTenantOperation("add_user")

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		prometheus.RecordAuthError("unauthorized_tenant_user_add")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Parse request
	var req struct {
		TenantID  uint   `json:"tenant_id"`
		UserEmail string `json:"user_email"`
		Role      string `json:"role,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse add user request", zap.Error(err))
		prometheus.RecordAuthError("invalid_request")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	if req.TenantID == 0 || req.UserEmail == "" {
		log.Error("Invalid request data",
			zap.Uint("tenant_id", req.TenantID),
			zap.String("user_email", req.UserEmail))
		prometheus.RecordAuthError("incomplete_tenant_user_add")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "tenant_id and user_email are required"})
	}

	// Default role if not provided
	if req.Role == "" {
		req.Role = "member"
	}

	// Track DB operations
	defer prometheus.TrackDBOperation("query")(time.Now())

	// Verify the requesting user has permission to add users to this tenant
	var userTenant model.UserTenant
	result := database.GetDB().Where("user_id = ? AND tenant_id = ? AND role IN ('owner', 'admin')", userID, req.TenantID).First(&userTenant)
	if result.Error != nil {
		log.Warn("Unauthorized attempt to add user to tenant",
			zap.Uint("requesting_user_id", userID),
			zap.Uint("tenant_id", req.TenantID))
		prometheus.RecordAuthError("tenant_permission_denied")
		return c.JSON(http.StatusForbidden, echo.Map{"error": "insufficient permissions"})
	}

	// Find the user by email
	var user model.User
	if result := database.GetDB().Where("email = ?", req.UserEmail).First(&user); result.Error != nil {
		log.Error("User not found", zap.String("email", req.UserEmail))
		prometheus.RecordAuthError("user_not_found")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}

	// Check if user is already in the tenant
	var existingUserTenant model.UserTenant
	result = database.GetDB().Where("user_id = ? AND tenant_id = ?", user.ID, req.TenantID).First(&existingUserTenant)
	if result.Error == nil {
		// User is already in the tenant, update their role if different
		if existingUserTenant.Role != req.Role {
			existingUserTenant.Role = req.Role
			if err := database.GetDB().Save(&existingUserTenant).Error; err != nil {
				log.Error("Failed to update user role in tenant", zap.Error(err))
				prometheus.RecordAuthError("tenant_user_update_failed")
				return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update user role"})
			}
			log.Info("Updated user role in tenant",
				zap.Uint("tenant_id", req.TenantID),
				zap.String("user_email", req.UserEmail),
				zap.String("role", req.Role))
		}

		return c.JSON(http.StatusOK, echo.Map{
			"message":     "User role updated in tenant",
			"user_tenant": existingUserTenant,
		})
	}

	// Add user to tenant
	newUserTenant := model.UserTenant{
		UserID:    user.ID,
		TenantID:  req.TenantID,
		Role:      req.Role,
		IsDefault: false, // Not default for newly added users
		Active:    true,
	}

	if err := database.GetDB().Create(&newUserTenant).Error; err != nil {
		log.Error("Failed to add user to tenant", zap.Error(err))
		prometheus.RecordAuthError("tenant_user_add_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to add user to tenant"})
	}

	log.Info("Added user to tenant",
		zap.Uint("tenant_id", req.TenantID),
		zap.String("user_email", req.UserEmail),
		zap.String("role", req.Role))

	return c.JSON(http.StatusCreated, echo.Map{
		"message":     "User added to tenant successfully",
		"user_tenant": newUserTenant,
	})
}

// RemoveUserFromTenant removes a user from a tenant
func RemoveUserFromTenant(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.RecordTenantOperation("remove_user")

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		prometheus.RecordAuthError("unauthorized_tenant_user_remove")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Parse parameters from URL
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 32)
	if err != nil {
		log.Error("Invalid tenant ID", zap.Error(err))
		prometheus.RecordAuthError("invalid_tenant_id")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid tenant ID"})
	}

	targetUserID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		log.Error("Invalid user ID", zap.Error(err))
		prometheus.RecordAuthError("invalid_user_id")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid user ID"})
	}

	// Track DB operations
	defer prometheus.TrackDBOperation("query")(time.Now())

	// Verify the requesting user has permission to remove users from this tenant
	var userTenant model.UserTenant
	result := database.GetDB().Where("user_id = ? AND tenant_id = ? AND role IN ('owner', 'admin')", userID, tenantID).First(&userTenant)
	if result.Error != nil {
		log.Warn("Unauthorized attempt to remove user from tenant",
			zap.Uint("requesting_user_id", userID),
			zap.Uint64("tenant_id", tenantID))
		prometheus.RecordAuthError("tenant_permission_denied")
		return c.JSON(http.StatusForbidden, echo.Map{"error": "insufficient permissions"})
	}

	// Check if target user is the tenant owner (can't remove the owner)
	var tenant model.Tenant
	if result := database.GetDB().First(&tenant, tenantID); result.Error != nil {
		log.Error("Tenant not found", zap.Uint64("id", tenantID))
		prometheus.RecordAuthError("tenant_not_found")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "tenant not found"})
	}

	if tenant.OwnerID == uint(targetUserID) {
		log.Warn("Attempted to remove tenant owner",
			zap.Uint64("tenant_id", tenantID),
			zap.Uint64("owner_id", targetUserID))
		prometheus.RecordAuthError("tenant_owner_removal_blocked")
		return c.JSON(http.StatusForbidden, echo.Map{"error": "cannot remove tenant owner"})
	}

	// Remove the user from the tenant
	result = database.GetDB().Where("user_id = ? AND tenant_id = ?", targetUserID, tenantID).Delete(&model.UserTenant{})
	if result.Error != nil {
		log.Error("Failed to remove user from tenant", zap.Error(result.Error))
		prometheus.RecordAuthError("tenant_user_remove_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to remove user from tenant"})
	}

	if result.RowsAffected == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found in this tenant"})
	}

	// Update default tenant status if needed
	database.GetDB().Model(&model.UserTenant{}).
		Where("user_id = ?", targetUserID).
		Order("created_at asc").
		Limit(1).
		Update("is_default", true)

	log.Info("Removed user from tenant",
		zap.Uint64("tenant_id", tenantID),
		zap.Uint64("user_id", targetUserID))

	return c.JSON(http.StatusOK, echo.Map{
		"message": "User removed from tenant successfully",
	})
}

// SetDefaultTenant sets a tenant as the user's default
func SetDefaultTenant(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.RecordTenantOperation("set_default")

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		prometheus.RecordAuthError("unauthorized_default_tenant_set")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Parse request
	var req struct {
		TenantID uint `json:"tenant_id"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse set default tenant request", zap.Error(err))
		prometheus.RecordAuthError("invalid_request")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	if req.TenantID == 0 {
		log.Error("Invalid tenant ID", zap.Uint("tenant_id", req.TenantID))
		prometheus.RecordAuthError("invalid_tenant_id")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "tenant_id is required"})
	}

	// Track DB operations
	defer prometheus.TrackDBOperation("update")(time.Now())

	// Begin transaction
	tx := database.GetDB().Begin()
	if tx.Error != nil {
		log.Error("Failed to begin transaction", zap.Error(tx.Error))
		prometheus.RecordAuthError("database_error")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "database error"})
	}

	// Verify the user has access to this tenant
	var userTenant model.UserTenant
	result := tx.Where("user_id = ? AND tenant_id = ? AND active = ?", userID, req.TenantID, true).First(&userTenant)
	if result.Error != nil {
		tx.Rollback()
		log.Warn("Unauthorized default tenant set attempt",
			zap.Uint("user_id", userID),
			zap.Uint("tenant_id", req.TenantID))
		prometheus.RecordAuthError("tenant_access_denied")
		return c.JSON(http.StatusForbidden, echo.Map{"error": "access denied to requested tenant"})
	}

	// Update all user's tenant associations to not be default
	if err := tx.Model(&model.UserTenant{}).Where("user_id = ?", userID).Update("is_default", false).Error; err != nil {
		tx.Rollback()
		log.Error("Failed to update user-tenant associations", zap.Error(err))
		prometheus.RecordAuthError("tenant_update_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update tenant associations"})
	}

	// Set the requested tenant as default
	userTenant.IsDefault = true
	if err := tx.Save(&userTenant).Error; err != nil {
		tx.Rollback()
		log.Error("Failed to set default tenant", zap.Error(err))
		prometheus.RecordAuthError("tenant_update_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to set default tenant"})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Error("Failed to commit transaction", zap.Error(err))
		prometheus.RecordAuthError("transaction_commit_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "transaction commit failed"})
	}

	log.Info("Set default tenant for user",
		zap.Uint("user_id", userID),
		zap.Uint("tenant_id", req.TenantID))

	return c.JSON(http.StatusOK, echo.Map{
		"message":   "Default tenant set successfully",
		"tenant_id": req.TenantID,
	})
}

// SelectTenant allows a user to select a tenant and receive a new JWT token with tenant information
func SelectTenant(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.TenantSelectionCounter.Inc()

	// Parse request
	var req struct {
		TenantID uint `json:"tenant_id"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse tenant selection request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	// Get token from Authorization header
	tokenString := c.Request().Header.Get("Authorization")
	if tokenString == "" {
		log.Error("Missing token")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Remove "Bearer " prefix if present
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// Validate the token
	claims, err := jwtutil.ValidateToken(tokenString)
	if err != nil {
		log.Error("Invalid token", zap.Error(err))
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid token"})
	}

	// Track DB operations
	defer prometheus.TrackDBOperation("query")(time.Now())

	// Verify user has access to the specified tenant
	var userTenant model.UserTenant
	if result := database.GetDB().Preload("Tenant").Where("user_id = ? AND tenant_id = ? AND active = ?", claims.UserID, req.TenantID, true).First(&userTenant); result.Error != nil {
		log.Warn("Tenant selection attempt for unauthorized tenant",
			zap.Uint("user_id", claims.UserID),
			zap.Uint("tenant_id", req.TenantID))
		return c.JSON(http.StatusForbidden, echo.Map{"error": "access denied to the specified tenant"})
	}

	// Generate new JWT token with tenant information
	token, err := jwtutil.GenerateTokenWithTenant(claims.Email, claims.UserID, &req.TenantID, userTenant.Tenant.Name, userTenant.Role)
	if err != nil {
		log.Error("Failed to generate token with tenant", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "token generation failed"})
	}

	log.Info("User selected tenant",
		zap.Uint("user_id", claims.UserID),
		zap.Uint("tenant_id", req.TenantID))

	return c.JSON(http.StatusOK, echo.Map{
		"token": token,
		"tenant": map[string]interface{}{
			"id":   req.TenantID,
			"name": userTenant.Tenant.Name,
			"role": userTenant.Role,
		},
	})
}
