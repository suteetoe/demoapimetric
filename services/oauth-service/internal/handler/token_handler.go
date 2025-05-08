package handler

import (
	"net/http"
	"oauth-service/internal/model"
	"oauth-service/pkg/config"
	"oauth-service/pkg/database"
	"oauth-service/pkg/logger"
	"oauth-service/prometheus"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// TokenConfig holds configuration for token generation
type TokenConfig struct {
	AccessTokenLifetime  time.Duration
	RefreshTokenLifetime time.Duration
}

var tokenConfig TokenConfig

// InitTokenHandler initializes token handler with configuration
func InitTokenHandler(cfg *config.Config) {
	tokenConfig = TokenConfig{
		AccessTokenLifetime:  cfg.OAuth.AccessTokenExpiration,
		RefreshTokenLifetime: cfg.OAuth.RefreshTokenExpiration,
	}
}

// IssueToken handles OAuth2 token requests
func IssueToken(c echo.Context) error {
	log := logger.FromContext(c)

	// Get client from context (set by ClientAuthMiddleware)
	client, ok := c.Get("client").(model.Client)
	if !ok {
		log.Error("Client not found in context")
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"error":             "invalid_client",
			"error_description": "Client authentication failed",
		})
	}

	// Parse token request form
	if err := c.Request().ParseForm(); err != nil {
		log.Error("Failed to parse form data", zap.Error(err))
		prometheus.InvalidTokenRequestCounter.With(map[string]string{"error_type": "invalid_form"}).Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "invalid_request",
			"error_description": "Could not parse form data",
		})
	}

	// Get grant type
	grantType := c.FormValue("grant_type")
	prometheus.TokenRequestCounter.With(map[string]string{"grant_type": grantType}).Inc()

	// Validate grant type is allowed for this client
	if !isGrantAllowed(client.Grants, grantType) {
		log.Warn("Grant type not allowed for client",
			zap.String("grant_type", grantType),
			zap.String("client_id", client.ID))
		prometheus.InvalidTokenRequestCounter.With(map[string]string{"error_type": "unauthorized_grant_type"}).Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "unauthorized_client",
			"error_description": "The client is not authorized to use this grant type",
		})
	}

	// Handle different grant types
	switch grantType {
	case "client_credentials":
		return handleClientCredentialsGrant(c, client)
	case "refresh_token":
		return handleRefreshTokenGrant(c, client)
	case "password":
		return handlePasswordGrant(c, client)
	default:
		log.Warn("Unsupported grant type", zap.String("grant_type", grantType))
		prometheus.InvalidTokenRequestCounter.With(map[string]string{"error_type": "unsupported_grant_type"}).Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "unsupported_grant_type",
			"error_description": "The authorization grant type is not supported",
		})
	}
}

// ValidateToken validates an access token and returns its details
func ValidateToken(c echo.Context) error {
	log := logger.FromContext(c)

	// Get token from form or query
	token := c.FormValue("token")
	if token == "" {
		token = c.QueryParam("token")
	}

	if token == "" {
		log.Warn("Missing token in validation request")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "invalid_request",
			"error_description": "Token parameter is required",
		})
	}

	// Track database operation
	defer prometheus.TrackDBOperation("query")(time.Now())

	// Find token in database
	var accessToken model.AccessToken
	if err := database.GetDB().Where("token = ?", token).First(&accessToken).Error; err != nil {
		log.Warn("Token not found", zap.Error(err))
		return c.JSON(http.StatusOK, echo.Map{
			"active": false,
		})
	}

	// Check if token is valid
	isValid := !accessToken.IsExpired() && !accessToken.Revoked

	if !isValid {
		return c.JSON(http.StatusOK, echo.Map{
			"active": false,
		})
	}

	// If token is valid, return token details
	response := map[string]interface{}{
		"active":    true,
		"client_id": accessToken.ClientID,
		"exp":       accessToken.ExpiresAt.Unix(),
		"iat":       accessToken.CreatedAt.Unix(),
		"scope":     accessToken.Scopes,
	}

	// Add optional fields if present
	if accessToken.UserID != nil {
		response["user_id"] = *accessToken.UserID
	}

	if accessToken.TenantID != nil {
		response["tenant_id"] = *accessToken.TenantID
	}

	return c.JSON(http.StatusOK, response)
}

// RevokeToken revokes an access token or refresh token
func RevokeToken(c echo.Context) error {
	log := logger.FromContext(c)

	// Get client from context (set by ClientAuthMiddleware)
	client, ok := c.Get("client").(model.Client)
	if !ok {
		log.Error("Client not found in context")
		return c.JSON(http.StatusUnauthorized, echo.Map{
			"error":             "invalid_client",
			"error_description": "Client authentication failed",
		})
	}

	// Parse form
	if err := c.Request().ParseForm(); err != nil {
		log.Error("Failed to parse form data", zap.Error(err))
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "invalid_request",
			"error_description": "Could not parse form data",
		})
	}

	// Get token and token type hint
	token := c.FormValue("token")
	tokenTypeHint := c.FormValue("token_type_hint") // Optional

	if token == "" {
		log.Warn("Missing token in revocation request")
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "invalid_request",
			"error_description": "Token parameter is required",
		})
	}

	// Track database operation
	defer prometheus.TrackDBOperation("update")(time.Now())

	// Try to revoke based on token type hint
	var success bool

	if tokenTypeHint == "access_token" || tokenTypeHint == "" {
		// Try to revoke access token
		if result := database.GetDB().Model(&model.AccessToken{}).
			Where("token = ? AND client_id = ?", token, client.ID).
			Update("revoked", true); result.RowsAffected > 0 {

			prometheus.RecordTokenRevoked("access_token", "client_request")
			success = true
		}
	}

	if (tokenTypeHint == "refresh_token" || tokenTypeHint == "") && !success {
		// Try to revoke refresh token
		if result := database.GetDB().Model(&model.RefreshToken{}).
			Where("token = ? AND client_id = ?", token, client.ID).
			Update("revoked", true); result.RowsAffected > 0 {

			prometheus.RecordTokenRevoked("refresh_token", "client_request")
			success = true
		}
	}

	// RFC 7009 requires 200 OK even if token was invalid
	return c.NoContent(http.StatusOK)
}

// Helper function to check if a grant type is allowed for a client
func isGrantAllowed(allowedGrants string, requestedGrant string) bool {
	grants := strings.Split(allowedGrants, ",")
	for _, grant := range grants {
		if strings.TrimSpace(grant) == requestedGrant {
			return true
		}
	}
	return false
}

// Handle client_credentials grant type
func handleClientCredentialsGrant(c echo.Context, client model.Client) error {
	log := logger.FromContext(c)

	// Parse requested scopes
	requestedScopes := c.FormValue("scope")

	// Validate scopes against allowed client scopes
	finalScopes := validateScopes(client.Scopes, requestedScopes)

	// Create access token
	accessToken, refreshToken, err := createTokens(client.ID, nil, nil, finalScopes)
	if err != nil {
		log.Error("Failed to create tokens", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":             "server_error",
			"error_description": "Failed to generate access token",
		})
	}

	// Update metrics
	prometheus.RecordTokenIssued("client_credentials", "access_token")
	prometheus.RecordTokenIssued("client_credentials", "refresh_token")
	prometheus.ActiveTokensGauge.Inc()

	// Return tokens
	return c.JSON(http.StatusOK, echo.Map{
		"access_token":  accessToken.Token,
		"token_type":    "Bearer",
		"expires_in":    int(tokenConfig.AccessTokenLifetime.Seconds()),
		"refresh_token": refreshToken.Token,
		"scope":         finalScopes,
	})
}

// Handle refresh_token grant type
func handleRefreshTokenGrant(c echo.Context, client model.Client) error {
	log := logger.FromContext(c)

	// Get refresh token
	refreshTokenValue := c.FormValue("refresh_token")
	if refreshTokenValue == "" {
		log.Warn("Missing refresh token")
		prometheus.InvalidTokenRequestCounter.With(map[string]string{"error_type": "invalid_request"}).Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "invalid_request",
			"error_description": "Refresh token is required",
		})
	}

	// Track database operation
	defer prometheus.TrackDBOperation("query")(time.Now())

	// Find refresh token in database
	var refreshToken model.RefreshToken
	if err := database.GetDB().Where("token = ? AND client_id = ? AND revoked = ?",
		refreshTokenValue, client.ID, false).First(&refreshToken).Error; err != nil {

		log.Warn("Invalid refresh token", zap.Error(err))
		prometheus.InvalidTokenRequestCounter.With(map[string]string{"error_type": "invalid_grant"}).Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "invalid_grant",
			"error_description": "The refresh token is invalid",
		})
	}

	// Check if token is expired
	if refreshToken.IsExpired() {
		log.Warn("Expired refresh token", zap.String("token_id", refreshToken.ID))
		prometheus.InvalidTokenRequestCounter.With(map[string]string{"error_type": "invalid_grant"}).Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "invalid_grant",
			"error_description": "The refresh token has expired",
		})
	}

	// Get the original access token to get scope and user info
	var originalAccessToken model.AccessToken
	if err := database.GetDB().First(&originalAccessToken, "id = ?", refreshToken.AccessTokenID).Error; err != nil {
		log.Error("Original access token not found", zap.Error(err))
		prometheus.InvalidTokenRequestCounter.With(map[string]string{"error_type": "server_error"}).Inc()
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":             "server_error",
			"error_description": "Failed to process refresh token",
		})
	}

	// Create new tokens
	accessToken, newRefreshToken, err := createTokens(
		client.ID,
		originalAccessToken.UserID,
		originalAccessToken.TenantID,
		originalAccessToken.Scopes,
	)

	if err != nil {
		log.Error("Failed to create new tokens", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":             "server_error",
			"error_description": "Failed to generate access token",
		})
	}

	// Mark old refresh token as revoked
	defer prometheus.TrackDBOperation("update")(time.Now())
	database.GetDB().Model(&refreshToken).Update("revoked", true)

	// Update metrics
	prometheus.TokensRefreshedCounter.Inc()
	prometheus.RecordTokenIssued("refresh_token", "access_token")
	prometheus.RecordTokenIssued("refresh_token", "refresh_token")

	// Return new tokens
	return c.JSON(http.StatusOK, echo.Map{
		"access_token":  accessToken.Token,
		"token_type":    "Bearer",
		"expires_in":    int(tokenConfig.AccessTokenLifetime.Seconds()),
		"refresh_token": newRefreshToken.Token,
		"scope":         accessToken.Scopes,
	})
}

// Handle password grant type
func handlePasswordGrant(c echo.Context, client model.Client) error {
	log := logger.FromContext(c)

	// Get username and password
	username := c.FormValue("username")
	password := c.FormValue("password")

	if username == "" || password == "" {
		log.Warn("Missing username or password")
		prometheus.InvalidTokenRequestCounter.With(map[string]string{"error_type": "invalid_request"}).Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "invalid_request",
			"error_description": "Username and password are required",
		})
	}

	// Get tenant ID if provided
	tenantIDStr := c.FormValue("tenant_id")
	var tenantID *uint

	// In a real implementation, we would validate user credentials against auth service
	// For now, we'll use a mock implementation
	user, err := authenticateUser(username, password, tenantIDStr)
	if err != nil {
		log.Warn("Authentication failed", zap.Error(err), zap.String("username", username))
		prometheus.InvalidTokenRequestCounter.With(map[string]string{"error_type": "invalid_grant"}).Inc()
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":             "invalid_grant",
			"error_description": "The user credentials are invalid",
		})
	}

	// If tenant ID is provided, parse it
	if tenantIDStr != "" {
		tenantIDValue, err := strconv.ParseUint(tenantIDStr, 10, 32)
		if err != nil {
			log.Warn("Invalid tenant ID", zap.Error(err), zap.String("tenant_id", tenantIDStr))
			prometheus.InvalidTokenRequestCounter.With(map[string]string{"error_type": "invalid_request"}).Inc()
			return c.JSON(http.StatusBadRequest, echo.Map{
				"error":             "invalid_request",
				"error_description": "Invalid tenant ID format",
			})
		}
		tenantID = new(uint)
		*tenantID = uint(tenantIDValue)
	}

	// Parse requested scopes
	requestedScopes := c.FormValue("scope")

	// Validate scopes against allowed client scopes
	finalScopes := validateScopes(client.Scopes, requestedScopes)

	// Create access token with user info
	accessToken, refreshToken, err := createTokens(client.ID, &user.ID, tenantID, finalScopes)
	if err != nil {
		log.Error("Failed to create tokens", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":             "server_error",
			"error_description": "Failed to generate access token",
		})
	}

	// Update metrics
	prometheus.RecordTokenIssued("password", "access_token")
	prometheus.RecordTokenIssued("password", "refresh_token")
	prometheus.ActiveTokensGauge.Inc()

	// Return tokens
	return c.JSON(http.StatusOK, echo.Map{
		"access_token":  accessToken.Token,
		"token_type":    "Bearer",
		"expires_in":    int(tokenConfig.AccessTokenLifetime.Seconds()),
		"refresh_token": refreshToken.Token,
		"scope":         finalScopes,
	})
}

// Helper function to validate and filter requested scopes against allowed scopes
func validateScopes(allowedScopes, requestedScopes string) string {
	if requestedScopes == "" {
		return allowedScopes // Use all allowed scopes if none requested
	}

	allowedScopeMap := make(map[string]bool)
	for _, scope := range strings.Split(allowedScopes, " ") {
		allowedScopeMap[scope] = true
	}

	validScopes := []string{}
	for _, scope := range strings.Split(requestedScopes, " ") {
		if allowedScopeMap[scope] {
			validScopes = append(validScopes, scope)
		}
	}

	return strings.Join(validScopes, " ")
}

// Helper function to create access and refresh tokens
func createTokens(clientID string, userID, tenantID *uint, scopes string) (*model.AccessToken, *model.RefreshToken, error) {
	// Create access token
	accessToken := &model.AccessToken{
		ClientID:  clientID,
		UserID:    userID,
		TenantID:  tenantID,
		Scopes:    scopes,
		ExpiresAt: time.Now().Add(tokenConfig.AccessTokenLifetime),
		Revoked:   false,
	}

	// Track database operation
	defer prometheus.TrackDBOperation("insert")(time.Now())

	// Save access token to database
	if err := database.GetDB().Create(accessToken).Error; err != nil {
		return nil, nil, err
	}

	// Create refresh token
	refreshToken := &model.RefreshToken{
		AccessTokenID: accessToken.ID,
		ClientID:      clientID,
		UserID:        userID,
		TenantID:      tenantID,
		ExpiresAt:     time.Now().Add(tokenConfig.RefreshTokenLifetime),
		Revoked:       false,
	}

	// Save refresh token to database
	if err := database.GetDB().Create(refreshToken).Error; err != nil {
		return nil, nil, err
	}

	return accessToken, refreshToken, nil
}

// Mock user authentication function
// In a real implementation, this would call the auth service
type User struct {
	ID       uint
	Username string
}

func authenticateUser(username, password, tenantIDStr string) (*User, error) {
	// This is a mock implementation for demonstration purposes
	// In reality, you would verify credentials against your auth service

	// Always authenticate test user for demo
	if username == "test@example.com" && password == "password" {
		return &User{
			ID:       1,
			Username: username,
		}, nil
	}

	return nil, echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
}
