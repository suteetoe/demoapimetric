package handler

import (
	"auth-service/internal/model"
	"auth-service/pkg/database"
	"auth-service/pkg/jwtutil"
	"auth-service/pkg/logger"
	localprometheus "auth-service/prometheus"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Login authenticates a user and returns a JWT token without tenant information
func Login(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Processing login request")
	localprometheus.LoginCounter.Inc()

	// Parse request
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse login request",
			zap.Error(err),
			zap.String("remote_ip", c.RealIP()))
		localprometheus.RecordAuthError("invalid_request")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   "Invalid request format",
			"message": "The request could not be processed due to invalid format",
		})
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		log.Warn("Login attempt with missing credentials",
			zap.Bool("email_provided", req.Email != ""),
			zap.Bool("password_provided", req.Password != ""),
			zap.String("remote_ip", c.RealIP()))
		localprometheus.RecordAuthError("missing_credentials")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   "Missing credentials",
			"message": "Both email and password are required",
		})
	}

	// Track DB operations
	defer localprometheus.TrackDBOperation("query")(time.Now())

	// Find user by email
	var user model.User
	if result := database.GetDB().Where("email = ?", req.Email).First(&user); result.Error != nil {
		log.Warn("Login attempt with non-existent email",
			zap.String("email", req.Email),
			zap.String("remote_ip", c.RealIP()),
			zap.Error(result.Error))
		localprometheus.RecordAuthError("user_not_found")
		// Don't reveal whether the user exists - use generic message
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"error":   "Invalid credentials",
			"message": "The provided email or password is incorrect",
		})
	}

	// Check password
	if !checkPasswordHash(req.Password, user.Password) {
		log.Warn("Login attempt with incorrect password",
			zap.String("email", req.Email),
			zap.String("remote_ip", c.RealIP()))
		localprometheus.RecordAuthError("invalid_password")
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"error":   "Invalid credentials",
			"message": "The provided email or password is incorrect",
		})
	}

	// Generate JWT token without tenant information
	token, err := jwtutil.GenerateToken(user.Email, user.ID)
	if err != nil {
		log.Error("Failed to generate token",
			zap.Error(err),
			zap.String("email", user.Email),
			zap.Uint("user_id", user.ID))
		localprometheus.RecordAuthError("token_generation_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   "Authentication error",
			"message": "Could not process the login request at this time",
		})
	}

	// Increment active tokens gauge
	localprometheus.IncreaseActiveTokens()

	// Fetch available tenants for the user
	var userTenants []model.UserTenant
	if result := database.GetDB().Preload("Tenant").Where("user_id = ? AND active = ?", user.ID, true).Find(&userTenants); result.Error != nil {
		log.Error("Failed to fetch user tenants",
			zap.Error(result.Error),
			zap.Uint("user_id", user.ID))
		// We still continue, just won't return tenant info
	}

	// Format tenant information
	tenants := []map[string]interface{}{}
	for _, ut := range userTenants {
		tenants = append(tenants, map[string]interface{}{
			"id":   ut.TenantID,
			"name": ut.Tenant.Name,
			"role": ut.Role,
		})
	}

	log.Info("User logged in successfully",
		zap.String("email", req.Email),
		zap.Uint("id", user.ID),
		zap.Int("tenant_count", len(tenants)),
		zap.String("remote_ip", c.RealIP()))

	// Record successful authentication
	localprometheus.AuthSuccessCounter.Inc()
	// Record authentication operation
	localprometheus.RecordAuthOperation("login_success")
	// Track API request specifically for rate monitoring
	localprometheus.APIRequestsCounter.With(prometheus.Labels{
		"service":       "authen-service",
		"endpoint_type": "auth_login",
	}).Inc()

	return c.JSON(http.StatusOK, echo.Map{
		"token":   token,
		"user_id": user.ID,
		"email":   user.Email,
		"tenants": tenants,
	})
}

func Register(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Processing registration request")
	localprometheus.RegisterCounter.Inc()

	// Parse request
	var req struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		FirstName string `json:"first_name,omitempty"`
		LastName  string `json:"last_name,omitempty"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse registration request",
			zap.Error(err),
			zap.String("remote_ip", c.RealIP()))
		localprometheus.RecordAuthError("invalid_request")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   "Invalid request format",
			"message": "The request could not be processed due to invalid format",
		})
	}

	// Validate required fields
	if req.Email == "" || req.Password == "" {
		log.Warn("Registration attempt with missing required fields",
			zap.Bool("email_provided", req.Email != ""),
			zap.Bool("password_provided", req.Password != ""),
			zap.String("remote_ip", c.RealIP()))
		localprometheus.RecordAuthError("incomplete_registration")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   "Missing required fields",
			"message": "Email and password are required for registration",
		})
	}

	// Check if user already exists
	defer localprometheus.TrackDBOperation("query")(time.Now())
	var existingUser model.User
	result := database.GetDB().Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		log.Warn("Registration attempt with existing email",
			zap.String("email", req.Email),
			zap.String("remote_ip", c.RealIP()))
		localprometheus.RecordAuthError("email_already_exists")
		return c.JSON(http.StatusConflict, echo.Map{
			"error":   "Email already registered",
			"message": "An account with this email already exists",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("Failed to hash password",
			zap.Error(err),
			zap.String("remote_ip", c.RealIP()))
		localprometheus.RecordAuthError("password_hash_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   "Registration failed",
			"message": "Could not process the registration request at this time",
		})
	}

	// Create new user
	user := model.User{
		Email:     req.Email,
		Password:  string(hashedPassword),
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}

	// Save to database - track DB insert operation
	defer localprometheus.TrackDBOperation("insert")(time.Now())
	if result := database.GetDB().Create(&user); result.Error != nil {
		log.Error("Failed to create user record",
			zap.Error(result.Error),
			zap.String("email", req.Email),
			zap.String("remote_ip", c.RealIP()))
		localprometheus.RecordAuthError("user_creation_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   "Registration failed",
			"message": "Could not complete the registration process",
		})
	}

	log.Info("User registered successfully",
		zap.String("email", user.Email),
		zap.Uint("user_id", user.ID),
		zap.String("remote_ip", c.RealIP()))

	// Track API request specifically for rate monitoring
	localprometheus.APIRequestsCounter.With(prometheus.Labels{
		"service":       "authen-service",
		"endpoint_type": "auth_register",
	}).Inc()

	return c.JSON(http.StatusCreated, echo.Map{
		"message": "User registered successfully",
		"user": map[string]interface{}{
			"id":    user.ID,
			"email": user.Email,
		},
	})
}

// GetProfile retrieves the authenticated user's profile information
func GetProfile(c echo.Context) error {
	log := logger.FromContext(c)
	localprometheus.RecordAuthOperation("profile_access")

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		localprometheus.RecordAuthError("unauthorized_profile_access")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Track DB operations
	defer localprometheus.TrackDBOperation("query")(time.Now())

	// Find user by ID
	var user model.User
	if result := database.GetDB().First(&user, userID); result.Error != nil {
		log.Error("Failed to retrieve user profile", zap.Error(result.Error))
		localprometheus.RecordAuthError("profile_retrieval_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to retrieve profile"})
	}

	// Return user profile (password is excluded via JSON tag in model)
	log.Info("Profile accessed", zap.Uint("user_id", userID))
	return c.JSON(http.StatusOK, echo.Map{
		"id":           user.ID,
		"email":        user.Email,
		"first_name":   user.FirstName,
		"last_name":    user.LastName,
		"phone_number": user.PhoneNumber,
		"created_at":   user.CreatedAt,
		"updated_at":   user.UpdatedAt,
	})
}

// UpdateProfile updates the authenticated user's profile information
func UpdateProfile(c echo.Context) error {
	log := logger.FromContext(c)
	localprometheus.RecordAuthOperation("profile_update")

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		localprometheus.RecordAuthError("unauthorized_profile_update")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Parse request
	var req struct {
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		PhoneNumber string `json:"phone_number"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse profile update request", zap.Error(err))
		localprometheus.RecordAuthError("invalid_request")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	// Track DB operations
	defer localprometheus.TrackDBOperation("update")(time.Now())

	// Find user by ID
	var user model.User
	if result := database.GetDB().First(&user, userID); result.Error != nil {
		log.Error("Failed to retrieve user for update", zap.Error(result.Error))
		localprometheus.RecordAuthError("profile_update_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update profile"})
	}

	// Update user profile fields
	changes := false

	if req.FirstName != "" && req.FirstName != user.FirstName {
		user.FirstName = req.FirstName
		changes = true
	}

	if req.LastName != "" && req.LastName != user.LastName {
		user.LastName = req.LastName
		changes = true
	}

	if req.PhoneNumber != "" && req.PhoneNumber != user.PhoneNumber {
		user.PhoneNumber = req.PhoneNumber
		changes = true
	}

	if !changes {
		log.Info("No changes to update in profile")
		return c.JSON(http.StatusOK, echo.Map{
			"message": "No changes to update",
			"user": echo.Map{
				"id":           user.ID,
				"email":        user.Email,
				"first_name":   user.FirstName,
				"last_name":    user.LastName,
				"phone_number": user.PhoneNumber,
			},
		})
	}

	// Save updated user profile
	if result := database.GetDB().Save(&user); result.Error != nil {
		log.Error("Failed to update profile", zap.Error(result.Error))
		localprometheus.RecordAuthError("profile_save_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to save profile updates"})
	}

	log.Info("Profile updated successfully", zap.Uint("user_id", userID))
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Profile updated successfully",
		"user": echo.Map{
			"id":           user.ID,
			"email":        user.Email,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"phone_number": user.PhoneNumber,
		},
	})
}

// ChangePassword updates the user's password
func ChangePassword(c echo.Context) error {
	log := logger.FromContext(c)
	localprometheus.RecordAuthOperation("password_change")

	// Get user ID from context (set by AuthMiddleware)
	userID, ok := c.Get("user_id").(uint)
	if !ok {
		log.Error("Failed to get user ID from context")
		localprometheus.RecordAuthError("unauthorized_password_change")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
	}

	// Parse request
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse password change request", zap.Error(err))
		localprometheus.RecordAuthError("invalid_request")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		log.Error("Missing password data")
		localprometheus.RecordAuthError("incomplete_password_change")
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "current and new password are required"})
	}

	// Track DB operations
	defer localprometheus.TrackDBOperation("update")(time.Now())

	// Find user by ID
	var user model.User
	if result := database.GetDB().First(&user, userID); result.Error != nil {
		log.Error("Failed to retrieve user for password change", zap.Error(result.Error))
		localprometheus.RecordAuthError("user_not_found")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update password"})
	}

	// Verify current password
	if !checkPasswordHash(req.CurrentPassword, user.Password) {
		log.Warn("Invalid current password", zap.Uint("user_id", userID))
		localprometheus.RecordAuthError("invalid_current_password")
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "current password is incorrect"})
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Error("Failed to hash new password", zap.Error(err))
		localprometheus.RecordAuthError("password_hash_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to process new password"})
	}

	// Update password
	user.Password = string(hashedPassword)
	if result := database.GetDB().Save(&user); result.Error != nil {
		log.Error("Failed to save new password", zap.Error(result.Error))
		localprometheus.RecordAuthError("password_update_failed")
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update password"})
	}

	log.Info("Password changed successfully", zap.Uint("user_id", userID))
	return c.JSON(http.StatusOK, echo.Map{
		"message": "Password updated successfully",
	})
}

func MetricsHandler(c echo.Context) error {
	handler := localprometheus.GetPrometheusHandler()
	handler.ServeHTTP(c.Response(), c.Request())
	return nil
}

// HealthCheck handles the health check endpoint
func HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"status":  "healthy",
		"service": "authen-service",
	})
}

// Helper function to safely handle nil uint pointers for logging
func nilSafeUint(val *uint) uint {
	if val == nil {
		return 0
	}
	return *val
}

// Helper function to check password hash
func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
