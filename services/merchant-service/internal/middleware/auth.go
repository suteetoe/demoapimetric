package middleware

import (
	"merchant-service/pkg/jwtutil"
	"merchant-service/pkg/logger"
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
			return c.JSON(http.StatusUnauthorized, echo.Map{"error": "missing authorization token"})
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Error("Invalid Authorization header format")
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
		if claims.MerchantID != nil {
			c.Set("merchant_id", *claims.MerchantID)
		}

		// Token is valid, proceed with the request
		return next(c)
	}
}
