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
		Email    string `json:"email"`
		Password string `json:"password"`
		TenantID *uint  `json:"tenant_id,omitempty"`
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
	}

	// Generate JWT token with tenant information if available
	var token string
	if selectedTenantID != nil {
		token, err = jwtutil.GenerateTokenWithTenant(user.Email, user.ID, selectedTenantID, tenantName, userRole)
	} else {
		token, err = jwtutil.GenerateToken(user.Email, user.ID)
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
			zap.String("email", user.Email))
	}

	// Build response with tenant info if available
	response := echo.Map{
		"token": token,
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
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
