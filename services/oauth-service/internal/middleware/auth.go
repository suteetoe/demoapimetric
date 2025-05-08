package middleware

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"oauth-service/internal/model"
	"oauth-service/pkg/database"
	"oauth-service/pkg/logger"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ClientAuthMiddleware validates client credentials for OAuth2 endpoints
func ClientAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := logger.FromContext(c)

		// Get Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			log.Warn("Missing client authentication")
			return c.JSON(http.StatusUnauthorized, echo.Map{
				"error":             "invalid_client",
				"error_description": "Client authentication required",
			})
		}

		// Parse Basic Authentication header
		if !strings.HasPrefix(authHeader, "Basic ") {
			log.Warn("Invalid authentication scheme", zap.String("scheme", strings.Split(authHeader, " ")[0]))
			return c.JSON(http.StatusUnauthorized, echo.Map{
				"error":             "invalid_client",
				"error_description": "Client authentication must use Basic scheme",
			})
		}

		// Extract client credentials
		clientID, clientSecret, err := parseBasicAuth(authHeader[6:])
		if err != nil {
			log.Warn("Invalid Basic auth header", zap.Error(err))
			return c.JSON(http.StatusUnauthorized, echo.Map{
				"error":             "invalid_client",
				"error_description": "Invalid client credentials format",
			})
		}

		// Validate client credentials against the database
		var client model.Client
		if err := database.GetDB().Where("id = ? AND is_active = ?", clientID, true).First(&client).Error; err != nil {
			log.Warn("Client not found or inactive", zap.String("client_id", clientID))
			return c.JSON(http.StatusUnauthorized, echo.Map{
				"error":             "invalid_client",
				"error_description": "Unknown client or client is inactive",
			})
		}

		// Verify client secret
		if !validateClientSecret(client.Secret, clientSecret) {
			log.Warn("Invalid client secret", zap.String("client_id", clientID))
			return c.JSON(http.StatusUnauthorized, echo.Map{
				"error":             "invalid_client",
				"error_description": "Invalid client credentials",
			})
		}

		// Add client to context
		c.Set("client", client)
		c.Set("client_id", client.ID)

		// Update logger with client information
		log = log.With(zap.String("client_id", client.ID))
		c.Set("logger", log)

		return next(c)
	}
}

// BearerTokenMiddleware validates access tokens for protected resource endpoints
func BearerTokenMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := logger.FromContext(c)

		// Get Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			log.Warn("Missing access token")
			return c.JSON(http.StatusUnauthorized, echo.Map{
				"error":             "invalid_token",
				"error_description": "Access token required",
			})
		}

		// Parse Bearer token
		if !strings.HasPrefix(authHeader, "Bearer ") {
			log.Warn("Invalid token scheme", zap.String("scheme", strings.Split(authHeader, " ")[0]))
			return c.JSON(http.StatusUnauthorized, echo.Map{
				"error":             "invalid_token",
				"error_description": "Token must use Bearer scheme",
			})
		}

		// Extract token
		tokenString := authHeader[7:]

		// Validate token against the database
		var accessToken model.AccessToken
		if err := database.GetDB().Where("token = ? AND revoked = ?", tokenString, false).First(&accessToken).Error; err != nil {
			log.Warn("Token not found or revoked", zap.Error(err))
			return c.JSON(http.StatusUnauthorized, echo.Map{
				"error":             "invalid_token",
				"error_description": "The access token is invalid",
			})
		}

		// Check if token is expired
		if accessToken.IsExpired() {
			log.Warn("Expired token", zap.String("token_id", accessToken.ID))
			return c.JSON(http.StatusUnauthorized, echo.Map{
				"error":             "invalid_token",
				"error_description": "The access token has expired",
			})
		}

		// Add token and related info to context
		c.Set("access_token", accessToken)
		c.Set("client_id", accessToken.ClientID)

		if accessToken.UserID != nil {
			c.Set("user_id", *accessToken.UserID)
		}

		if accessToken.TenantID != nil {
			c.Set("tenant_id", *accessToken.TenantID)
		}

		// Update logger with token information
		log = log.With(
			zap.String("token_id", accessToken.ID),
			zap.String("client_id", accessToken.ClientID),
		)

		if accessToken.UserID != nil {
			log = log.With(zap.Uint("user_id", *accessToken.UserID))
		}

		if accessToken.TenantID != nil {
			log = log.With(zap.Uint("tenant_id", *accessToken.TenantID))
		}

		c.Set("logger", log)

		return next(c)
	}
}

// Helper function to parse Basic authentication
func parseBasicAuth(auth string) (string, string, error) {
	// Decode the Base64 encoded string
	decodedBytes, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return "", "", fmt.Errorf("invalid Base64 encoding in Authorization header: %w", err)
	}

	// Convert decoded bytes to string
	decodedString := string(decodedBytes)

	// Split at the first colon
	colonIndex := strings.IndexByte(decodedString, ':')
	if colonIndex < 0 {
		return "", "", fmt.Errorf("invalid Basic auth format: missing colon separator")
	}

	// Extract clientID and clientSecret
	clientID := decodedString[:colonIndex]
	clientSecret := decodedString[colonIndex+1:]

	// Validate that both values are present
	if clientID == "" {
		return "", "", fmt.Errorf("missing client ID in Basic auth")
	}

	// Return the extracted values
	return clientID, clientSecret, nil
}

// Helper function to validate client secret
func validateClientSecret(storedSecret, providedSecret string) bool {
	// Use bcrypt's CompareHashAndPassword for secure constant-time comparison
	// storedSecret should already be hashed during client registration
	err := bcrypt.CompareHashAndPassword([]byte(storedSecret), []byte(providedSecret))
	return err == nil
}
