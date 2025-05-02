package handler

import (
	"auth-service/internal/model"
	"auth-service/pkg/database"
	"auth-service/pkg/jwtutil"
	"auth-service/pkg/logger"
	"auth-service/prometheus"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func Login(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.LoginCounter.Inc()

	// Parse request
	var req struct {
		Email      string `json:"email"`
		Password   string `json:"password"`
		MerchantID *uint  `json:"merchant_id,omitempty"`
		TenantID   *uint  `json:"tenant_id,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse login request", zap.Error(err))
		prometheus.RecordAuthError("invalid_request")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	// Find user in database - track DB operation duration
	defer prometheus.TrackDBOperation("query")(time.Now())
	var user model.User
	result := database.GetDB().Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		log.Error("User not found", zap.String("email", req.Email))
		prometheus.RecordAuthError("user_not_found")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid credentials"})
	}

	// Verify password
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		log.Error("Invalid password", zap.String("email", req.Email))
		prometheus.RecordAuthError("invalid_password")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid credentials"})
	}

	// Handle tenant selection logic
	var selectedTenantID *uint
	var tenantName string
	var userRole string

	if req.TenantID != nil {
		// If tenant ID is provided, verify the user has access to this tenant
		var userTenant model.UserTenant
		result := database.GetDB().Where("user_id = ? AND tenant_id = ? AND active = ?", user.ID, *req.TenantID, true).First(&userTenant)
		if result.Error != nil {
			log.Error("User does not have access to the specified tenant",
				zap.String("email", req.Email),
				zap.Uint("tenant_id", *req.TenantID))
			prometheus.RecordAuthError("tenant_access_denied")
			return c.JSON(http.StatusForbidden, echo.Map{"error": "access denied to the specified tenant"})
		}

		// Get tenant name
		var tenant model.Tenant
		if result := database.GetDB().Select("name").First(&tenant, *req.TenantID); result.Error == nil {
			tenantName = tenant.Name
		}

		selectedTenantID = req.TenantID
		userRole = userTenant.Role
	} else if user.TenantID != nil {
		// Use the user's default tenant if available
		selectedTenantID = user.TenantID

		// Get tenant name and user role
		var tenant model.Tenant
		if result := database.GetDB().Select("name").First(&tenant, *user.TenantID); result.Error == nil {
			tenantName = tenant.Name
		}

		var userTenant model.UserTenant
		if result := database.GetDB().Select("role").Where("user_id = ? AND tenant_id = ?", user.ID, *user.TenantID).First(&userTenant); result.Error == nil {
			userRole = userTenant.Role
		}
	} else if req.MerchantID != nil {
		// For backward compatibility
		user.MerchantID = req.MerchantID
		if err := database.GetDB().Model(&user).Update("merchant_id", req.MerchantID).Error; err != nil {
			log.Error("Failed to update user's merchant ID", zap.Error(err))
			prometheus.RecordAuthError("merchant_update_failed")
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update merchant association"})
		}
	}

	// Generate JWT token with tenant information if available
	var token string
	if selectedTenantID != nil {
		token, err = jwtutil.GenerateTokenWithTenant(user.Email, user.ID, user.MerchantID, selectedTenantID, tenantName, userRole)
	} else {
		token, err = jwtutil.GenerateToken(user.Email, user.ID, user.MerchantID)
	}

	if err != nil {
		log.Error("Failed to generate token", zap.Error(err))
		prometheus.RecordAuthError("token_generation_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "token error"})
	}

	// Increment active tokens gauge
	prometheus.IncreaseActiveTokens()

	// Log with tenant information if available
	if selectedTenantID != nil {
		log.Info("User logged in with tenant context",
			zap.String("email", user.Email),
			zap.Uint("tenant_id", *selectedTenantID),
			zap.String("tenant_name", tenantName),
			zap.String("role", userRole))
	} else {
		log.Info("User logged in",
			zap.String("email", user.Email),
			zap.Uint("merchant_id", nilSafeUint(user.MerchantID)))
	}

	// Build response with tenant info if available
	response := echo.Map{
		"token": token,
		"user": map[string]interface{}{
			"id":          user.ID,
			"email":       user.Email,
			"merchant_id": user.MerchantID,
		},
	}

	if selectedTenantID != nil {
		response["tenant"] = map[string]interface{}{
			"id":   *selectedTenantID,
			"name": tenantName,
			"role": userRole,
		}
	}

	return c.JSON(http.StatusOK, response)
}

func Register(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.RegisterCounter.Inc()

	// Parse request
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse registration request", zap.Error(err))
		prometheus.RecordAuthError("invalid_request")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	if req.Email == "" || req.Password == "" {
		log.Error("Invalid registration data",
			zap.String("email", req.Email),
			zap.Bool("password_provided", req.Password != ""))
		prometheus.RecordAuthError("incomplete_registration")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "email and password are required"})
	}

	// Check if user already exists - track DB query
	defer prometheus.TrackDBOperation("query")(time.Now())
	var existingUser model.User
	result := database.GetDB().Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		log.Error("User already exists", zap.String("email", req.Email))
		prometheus.RecordAuthError("email_already_exists")
		return c.JSON(http.StatusConflict, echo.Map{"error": "email already registered"})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("Failed to hash password", zap.Error(err))
		prometheus.RecordAuthError("password_hash_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "registration failed"})
	}

	// Create new user
	user := model.User{
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	// Save to database - track DB insert operation
	defer prometheus.TrackDBOperation("insert")(time.Now())
	if result := database.GetDB().Create(&user); result.Error != nil {
		log.Error("Failed to create user", zap.Error(result.Error))
		prometheus.RecordAuthError("user_creation_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "registration failed"})
	}

	log.Info("User registered", zap.String("email", user.Email))
	return c.JSON(http.StatusCreated, echo.Map{
		"message": "User registered successfully",
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

// AssociateMerchant associates a user with a merchant
func AssociateMerchant(c echo.Context) error {
	log := logger.FromContext(c)
	prometheus.MerchantAssociationCounter.Inc()

	// Parse request
	var req struct {
		UserID     uint   `json:"user_id"`
		MerchantID uint   `json:"merchant_id"`
		Role       string `json:"role,omitempty"` // Optional role
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse merchant association request", zap.Error(err))
		prometheus.RecordAuthError("invalid_request")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	if req.UserID == 0 || req.MerchantID == 0 {
		log.Error("Invalid merchant association data",
			zap.Uint("user_id", req.UserID),
			zap.Uint("merchant_id", req.MerchantID))
		prometheus.RecordAuthError("incomplete_merchant_association")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id and merchant_id are required"})
	}

	// Set default role if not provided
	if req.Role == "" {
		req.Role = "member"
	}

	// Find user in database - track DB query
	defer prometheus.TrackDBOperation("query")(time.Now())
	var user model.User
	result := database.GetDB().First(&user, req.UserID)
	if result.Error != nil {
		log.Error("User not found", zap.Uint("user_id", req.UserID))
		prometheus.RecordAuthError("user_not_found")
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}

	// Create merchant_user association in database - track DB insert operation
	defer prometheus.TrackDBOperation("insert")(time.Now())

	// Define a struct that represents the merchant_users table
	type MerchantUser struct {
		MerchantID uint   `json:"merchant_id"`
		UserID     uint   `json:"user_id"`
		Role       string `json:"role"`
		Active     bool   `json:"active"`
		CreatedAt  time.Time
		UpdatedAt  time.Time
	}

	// Check if association already exists
	var existingAssociation MerchantUser
	checkResult := database.GetDB().Table("merchant_users").
		Where("merchant_id = ? AND user_id = ?", req.MerchantID, req.UserID).
		First(&existingAssociation)

	if checkResult.Error == nil {
		// Association exists, update it
		updateData := map[string]interface{}{
			"role":       req.Role,
			"updated_at": time.Now(),
		}

		if updateResult := database.GetDB().Table("merchant_users").
			Where("merchant_id = ? AND user_id = ?", req.MerchantID, req.UserID).
			Updates(updateData).Error; updateResult != nil {
			log.Error("Failed to update merchant user association", zap.Error(updateResult))
			prometheus.RecordAuthError("merchant_association_update_failed")
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update merchant association"})
		}

		log.Info("User association with merchant updated",
			zap.Uint("user_id", user.ID),
			zap.Uint("merchant_id", req.MerchantID),
			zap.String("role", req.Role))

		return c.JSON(http.StatusOK, echo.Map{
			"message": "User association with merchant updated successfully",
			"user": map[string]interface{}{
				"id":    user.ID,
				"email": user.Email,
			},
			"merchant_id": req.MerchantID,
			"role":        req.Role,
		})
	}

	// Create new association
	merchantUser := MerchantUser{
		MerchantID: req.MerchantID,
		UserID:     user.ID,
		Role:       req.Role,
		Active:     true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if result := database.GetDB().Table("merchant_users").Create(&merchantUser); result.Error != nil {
		log.Error("Failed to create merchant user association", zap.Error(result.Error))
		prometheus.RecordAuthError("merchant_association_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to create merchant association"})
	}

	// Also update user's primary merchant ID for backward compatibility and convenience
	if err := database.GetDB().Model(&user).Update("merchant_id", req.MerchantID).Error; err != nil {
		log.Error("Failed to update user's merchant ID", zap.Error(err))
		// We won't fail here as the main association was created successfully
	}

	log.Info("User associated with merchant",
		zap.Uint("user_id", user.ID),
		zap.Uint("merchant_id", req.MerchantID),
		zap.String("role", req.Role))

	return c.JSON(http.StatusOK, echo.Map{
		"message": "User associated with merchant successfully",
		"user": map[string]interface{}{
			"id":          user.ID,
			"email":       user.Email,
			"merchant_id": user.MerchantID, // Still include the primary merchant ID
		},
		"merchant_user": map[string]interface{}{
			"merchant_id": req.MerchantID,
			"user_id":     user.ID,
			"role":        req.Role,
		},
	})
}

func MetricsHandler(c echo.Context) error {
	handler := prometheus.GetPrometheusHandler()
	handler.ServeHTTP(c.Response(), c.Request())
	return nil
}

// Helper function to safely handle nil uint pointers for logging
func nilSafeUint(val *uint) uint {
	if val == nil {
		return 0
	}
	return *val
}
