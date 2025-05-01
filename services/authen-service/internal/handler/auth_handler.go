package handler

import (
	"auth-service/internal/model"
	"auth-service/pkg/database"
	"auth-service/pkg/jwtutil"
	"auth-service/pkg/logger"
	"auth-service/prometheus"
	"net/http"

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
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse login request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	// Find user in database
	var user model.User
	result := database.GetDB().Where("email = ?", req.Email).First(&user)
	if result.Error != nil {
		log.Error("User not found", zap.String("email", req.Email))
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid credentials"})
	}

	// Verify password
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		log.Error("Invalid password", zap.String("email", req.Email))
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid credentials"})
	}

	// If merchantID is provided, validate and update the user's merchant association
	if req.MerchantID != nil {
		// Check if merchant exists by making a HTTP request to merchant service
		// For now, just update the user's merchant ID
		// In a real-world scenario, you'd make an HTTP call to merchant service to validate
		user.MerchantID = req.MerchantID
		if err := database.GetDB().Model(&user).Update("merchant_id", req.MerchantID).Error; err != nil {
			log.Error("Failed to update user's merchant ID", zap.Error(err))
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update merchant association"})
		}
	}

	// Generate JWT token with merchant ID if available
	token, err := jwtutil.GenerateToken(user.Email, user.ID, user.MerchantID)
	if err != nil {
		log.Error("Failed to generate token", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "token error"})
	}

	log.Info("User logged in",
		zap.String("email", user.Email),
		zap.Uint("merchant_id", nilSafeUint(user.MerchantID)))

	return c.JSON(http.StatusOK, echo.Map{
		"token": token,
		"user": map[string]interface{}{
			"id":          user.ID,
			"email":       user.Email,
			"merchant_id": user.MerchantID,
		},
	})
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
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	if req.Email == "" || req.Password == "" {
		log.Error("Invalid registration data",
			zap.String("email", req.Email),
			zap.Bool("password_provided", req.Password != ""))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "email and password are required"})
	}

	// Check if user already exists
	var existingUser model.User
	result := database.GetDB().Where("email = ?", req.Email).First(&existingUser)
	if result.Error == nil {
		log.Error("User already exists", zap.String("email", req.Email))
		return c.JSON(http.StatusConflict, echo.Map{"error": "email already registered"})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("Failed to hash password", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "registration failed"})
	}

	// Create new user
	user := model.User{
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	// Save to database
	if result := database.GetDB().Create(&user); result.Error != nil {
		log.Error("Failed to create user", zap.Error(result.Error))
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

	// Parse request
	var req struct {
		UserID     uint `json:"user_id"`
		MerchantID uint `json:"merchant_id"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse merchant association request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "invalid request"})
	}

	if req.UserID == 0 || req.MerchantID == 0 {
		log.Error("Invalid merchant association data",
			zap.Uint("user_id", req.UserID),
			zap.Uint("merchant_id", req.MerchantID))
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "user_id and merchant_id are required"})
	}

	// Find user in database
	var user model.User
	result := database.GetDB().First(&user, req.UserID)
	if result.Error != nil {
		log.Error("User not found", zap.Uint("user_id", req.UserID))
		return c.JSON(http.StatusNotFound, echo.Map{"error": "user not found"})
	}

	// Update user's merchant ID
	user.MerchantID = &req.MerchantID
	if result := database.GetDB().Save(&user); result.Error != nil {
		log.Error("Failed to update user's merchant ID", zap.Error(result.Error))
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to update merchant association"})
	}

	log.Info("User associated with merchant",
		zap.Uint("user_id", user.ID),
		zap.Uint("merchant_id", *user.MerchantID))

	return c.JSON(http.StatusOK, echo.Map{
		"message": "User associated with merchant successfully",
		"user": map[string]interface{}{
			"id":          user.ID,
			"email":       user.Email,
			"merchant_id": user.MerchantID,
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
