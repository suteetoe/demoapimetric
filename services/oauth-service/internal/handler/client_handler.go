package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"oauth-service/internal/model"
	"oauth-service/pkg/database"
	"oauth-service/pkg/logger"
	"oauth-service/prometheus"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// RegisterClient creates a new OAuth client
func RegisterClient(c echo.Context) error {
	log := logger.FromContext(c)

	// Track client registration attempt
	prometheus.ClientRegistrationCounter.Inc()

	// Parse request
	var req struct {
		Name         string   `json:"name" validate:"required"`
		RedirectURIs []string `json:"redirect_uris" validate:"required"`
		Grants       []string `json:"grants" validate:"required"`
		Scopes       []string `json:"scopes"`
		UserID       *uint    `json:"user_id"`
		TenantID     *uint    `json:"tenant_id"`
	}

	if err := c.Bind(&req); err != nil {
		log.Error("Failed to parse client registration request", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "invalid_request",
			"error_description": "Could not parse request body",
		})
	}

	// Basic validation
	if req.Name == "" || len(req.RedirectURIs) == 0 || len(req.Grants) == 0 {
		log.Warn("Validation failed for client registration")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "invalid_request",
			"error_description": "Name, redirect URIs, and grant types are required",
		})
	}

	// Generate client ID and secret
	clientSecret := generateRandomClientSecret()

	// Hash client secret for storage
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		log.Error("Failed to hash client secret", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":             "server_error",
			"error_description": "Failed to process client registration",
		})
	}

	// Create client record
	client := model.Client{
		Name:         req.Name,
		Secret:       string(hashedSecret),
		RedirectURIs: joinStrings(req.RedirectURIs, ","),
		Grants:       joinStrings(req.Grants, ","),
		Scopes:       joinStrings(req.Scopes, ","),
		UserID:       req.UserID,
		TenantID:     req.TenantID,
		IsActive:     true,
	}

	// Track database operation
	defer prometheus.TrackDBOperation("insert")(time.Now())

	// Save to database
	if err := database.GetDB().Create(&client).Error; err != nil {
		log.Error("Failed to create client", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":             "server_error",
			"error_description": "Failed to register client",
		})
	}

	// Update metrics
	prometheus.ActiveClientsGauge.Inc()

	// Return client details with plaintext secret (only time it's shown)
	return c.JSON(http.StatusCreated, echo.Map{
		"client_id":     client.ID,
		"client_secret": clientSecret,
		"name":          client.Name,
		"redirect_uris": req.RedirectURIs,
		"grants":        req.Grants,
		"scopes":        req.Scopes,
	})
}

// GetClient retrieves client details (requires authentication)
func GetClient(c echo.Context) error {
	log := logger.FromContext(c)

	// Get client ID from path parameter
	clientID := c.Param("id")
	if clientID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "invalid_request",
			"error_description": "Client ID is required",
		})
	}

	// Track database operation
	defer prometheus.TrackDBOperation("query")(time.Now())

	// Retrieve client from database
	var client model.Client
	if err := database.GetDB().First(&client, "id = ?", clientID).Error; err != nil {
		log.Error("Client not found", zap.String("client_id", clientID), zap.Error(err))
		return c.JSON(http.StatusNotFound, echo.Map{
			"error":             "not_found",
			"error_description": "Client not found",
		})
	}

	// Return client details (without secret)
	return c.JSON(http.StatusOK, client)
}

// Helper function to generate a secure client secret
func generateRandomClientSecret() string {
	b := make([]byte, 32) // 256 bits
	if _, err := rand.Read(b); err != nil {
		// Handle error. In production code, we should handle this better.
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// Helper to join string slices
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
