package middleware

import (
	"auth-service/pkg/jwtutil"
	"auth-service/pkg/logger"
	"auth-service/prometheus"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// AuthMiddleware validates the JWT token from the Authorization header
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := logger.FromContext(c)

		// Get the Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			log.Error("Missing Authorization header")
			prometheus.RecordAuthError("missing_token")
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "missing authorization token"})
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Error("Invalid Authorization header format")
			prometheus.RecordAuthError("invalid_auth_format")
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid authorization format, expected Bearer token"})
		}

		// Extract the token
		tokenString := parts[1]

		// Validate the token
		claims, err := jwtutil.ValidateToken(tokenString)
		if err != nil {
			log.Error("Invalid JWT token", zap.Error(err))
			prometheus.RecordAuthError("invalid_token")
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid or expired token"})
		}

		// Store user info in context for later use
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		// Store merchant ID for backward compatibility
		if claims.MerchantID != nil {
			c.Set("merchant_id", *claims.MerchantID)
		}

		// Store tenant information if available
		if claims.TenantID != nil {
			c.Set("tenant_id", *claims.TenantID)
			c.Set("tenant_name", claims.TenantName)
			c.Set("user_role", claims.Role)

			// Add tenant ID to request header for downstream services
			c.Request().Header.Set("X-Tenant-ID", fmt.Sprintf("%d", *claims.TenantID))
			if claims.TenantName != "" {
				c.Request().Header.Set("X-Tenant-Name", claims.TenantName)
			}
			if claims.Role != "" {
				c.Request().Header.Set("X-User-Role", claims.Role)
			}

			log.Debug("Request authenticated with tenant context",
				zap.Uint("tenant_id", *claims.TenantID),
				zap.String("tenant_name", claims.TenantName),
				zap.String("role", claims.Role))
		}

		// Token is valid, proceed with the request
		return next(c)
	}
}
