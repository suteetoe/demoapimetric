package oauth

import (
	"errors"
	"fmt"
	"net/http"
	"product-service/prometheus"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// Middleware creates an Echo middleware for OAuth2 token validation
func Middleware(client *Client, requiredScopes []string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			start := time.Now()

			// Extract token from Authorization header
			authHeader := ctx.Request().Header.Get("Authorization")
			if authHeader == "" {
				prometheus.AuthErrorsCounter.Inc()
				return ctx.JSON(http.StatusUnauthorized, map[string]string{
					"error":             "missing_token",
					"error_description": "Authorization header is required",
				})
			}

			// Check if it's a Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				prometheus.AuthErrorsCounter.Inc()
				return ctx.JSON(http.StatusUnauthorized, map[string]string{
					"error":             "invalid_token",
					"error_description": "Invalid authorization format, expected Bearer token",
				})
			}

			// Extract the token
			token := parts[1]

			// Get logger from context
			logger, ok := ctx.Get("logger").(*zap.Logger)
			if !ok {
				logger = client.Logger
			}

			// Validate token with OAuth service
			validation, err := client.ValidateToken(token)
			if err != nil {
				logger.Warn("Token validation failed", zap.Error(err))
				prometheus.AuthErrorsCounter.Inc()
				return ctx.JSON(http.StatusUnauthorized, map[string]string{
					"error":             "invalid_token",
					"error_description": "The access token is invalid",
				})
			}

			// Check if token is active
			if !validation.Active {
				logger.Warn("Token is inactive",
					zap.String("client_id", validation.ClientID))
				prometheus.AuthErrorsCounter.Inc()
				return ctx.JSON(http.StatusUnauthorized, map[string]string{
					"error":             "invalid_token",
					"error_description": "The token is inactive or expired",
				})
			}

			// Validate scopes if required
			if len(requiredScopes) > 0 {
				if err := validateScopes(validation.Scope, requiredScopes); err != nil {
					logger.Warn("Insufficient scope",
						zap.String("required", strings.Join(requiredScopes, " ")),
						zap.String("provided", validation.Scope))
					prometheus.AuthErrorsCounter.Inc()
					return ctx.JSON(http.StatusForbidden, map[string]string{
						"error":             "insufficient_scope",
						"error_description": fmt.Sprintf("The token does not have the required scope: %s", strings.Join(requiredScopes, " ")),
					})
				}
			}

			// Record successful authentication
			prometheus.AuthSuccessCounter.Inc()

			// Add claims to the context
			ctx.Set("client_id", validation.ClientID)

			if validation.UserID != 0 {
				ctx.Set("user_id", validation.UserID)
			}

			if validation.TenantID != 0 {
				ctx.Set("tenant_id", validation.TenantID)
			}

			ctx.Set("token_scopes", validation.Scope)

			// Update the logger in context with claims
			fields := []zap.Field{
				zap.String("client_id", validation.ClientID),
			}

			if validation.UserID != 0 {
				fields = append(fields, zap.Uint("user_id", validation.UserID))
			}

			if validation.TenantID != 0 {
				fields = append(fields, zap.Uint("tenant_id", validation.TenantID))
			}

			ctx.Set("logger", logger.With(fields...))

			// Record the authentication duration
			duration := time.Since(start).Seconds()
			prometheus.AuthDurationHistogram.Observe(duration)

			return next(ctx)
		}
	}
}

// validateScopes checks if the token's scope string contains all required scopes
func validateScopes(tokenScopes string, requiredScopes []string) error {
	if tokenScopes == "" {
		return errors.New("token has no scopes")
	}

	// Convert token scopes to a map for easy lookup
	scopeMap := make(map[string]bool)
	for _, scope := range strings.Split(tokenScopes, " ") {
		scopeMap[scope] = true
	}

	// Check if all required scopes are in the token
	for _, requiredScope := range requiredScopes {
		if !scopeMap[requiredScope] {
			return fmt.Errorf("missing required scope: %s", requiredScope)
		}
	}

	return nil
}
