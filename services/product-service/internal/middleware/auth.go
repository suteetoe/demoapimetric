package middleware

import (
	"net/http"
	"product-service/pkg/jwtutil"
	"product-service/pkg/logger"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// AuthMiddleware validates the JWT token and extracts tenant information
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := logger.FromContext(c)

		// Get the Authorization header
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			log.Warn("Missing Authorization header")
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "missing authorization token"})
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Warn("Invalid Authorization header format")
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid authorization format, expected Bearer token"})
		}

		// Extract the token
		tokenString := parts[1]

		// Validate the token
		claims, err := jwtutil.ValidateToken(tokenString)
		if err != nil {
			log.Error("Invalid JWT token", zap.Error(err))
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid or expired token"})
		}

		// Store user info in context for later use
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		// Store tenant information if available
		if claims.TenantID != nil {
			c.Set("tenant_id", *claims.TenantID)
			c.Set("tenant_name", claims.TenantName)
			c.Set("user_role", claims.Role)
			log.Info("Request authenticated with tenant context",
				zap.Uint("tenant_id", *claims.TenantID),
				zap.String("tenant_name", claims.TenantName),
				zap.String("role", claims.Role))
		} else {
			log.Warn("JWT token does not contain tenant_id")
			return c.JSON(http.StatusBadRequest, echo.Map{"error": "tenant_id is required in the token"})
		}

		// Token is valid, proceed with the request
		return next(c)
	}
}

// GetTenantIDFromContext retrieves the tenant ID from the context
// Returns 0, false if tenant ID is not found
func GetTenantIDFromContext(c echo.Context) (uint, bool) {
	tenantID, ok := c.Get("tenant_id").(uint)
	return tenantID, ok
}
