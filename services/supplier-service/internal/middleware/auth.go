package middleware

import (
	"net/http"
	"strings"
	"supplier-service/pkg/jwtutil"
	"supplier-service/pkg/logger"
	"supplier-service/prometheus"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// AuthMiddleware verifies the JWT token and extracts claims
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := logger.FromContext(c)

		// Track authentication attempts
		prometheus.AuthAttemptsCounter.Inc()

		// Extract the token from the Authorization header
		tokenString := c.Request().Header.Get("Authorization")
		if tokenString == "" {
			log.Warn("Missing authorization token")
			prometheus.AuthErrorsCounter.Inc()
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "authentication required"})
		}

		// Remove "Bearer " prefix if present
		if len(tokenString) > 7 && strings.ToUpper(tokenString[0:7]) == "BEARER " {
			tokenString = tokenString[7:]
		}

		// Validate the token
		claims, err := jwtutil.ValidateToken(tokenString)
		if err != nil {
			log.Warn("Invalid token", zap.Error(err))
			prometheus.AuthErrorsCounter.Inc()
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "invalid token"})
		}

		// Increment successful auth counter
		prometheus.AuthSuccessCounter.Inc()

		// Store user information in the context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)

		// If token has tenant context, store it in the context
		if claims.TenantID != nil {
			c.Set("tenant_id", *claims.TenantID)
			c.Set("tenant_name", claims.TenantName)
			c.Set("role", claims.Role)

			// Add tenant info to logger
			log = log.With(
				zap.Uint("tenant_id", *claims.TenantID),
				zap.String("tenant_name", claims.TenantName),
				zap.String("role", claims.Role),
			)
		}

		// Update logger with user information
		log = log.With(
			zap.Uint("user_id", claims.UserID),
			zap.String("email", claims.Email),
		)
		c.Set("logger", log)

		// Call the next handler
		return next(c)
	}
}

// RequireTenantContext ensures the request has tenant context in the JWT
func RequireTenantContext(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		log := logger.FromContext(c)

		// Check if tenant_id exists in context
		tenantID, ok := c.Get("tenant_id").(uint)
		if !ok || tenantID == 0 {
			log.Warn("Missing tenant context")
			prometheus.TenantContextMissingCounter.Inc()
			return c.JSON(http.StatusForbidden, echo.Map{
				"error":   "tenant context required",
				"message": "Please select a tenant before accessing this resource",
			})
		}

		// Tenant context exists, proceed
		return next(c)
	}
}
