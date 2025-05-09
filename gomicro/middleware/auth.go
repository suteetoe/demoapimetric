package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/suteetoe/gomicro/jwtutil"
	"github.com/suteetoe/gomicro/logger"
	"go.uber.org/zap"
)

// JWTAuthMiddleware creates a middleware that validates JWT tokens
func JWTAuthMiddleware(jwtUtil *jwtutil.JWTUtil) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			log := logger.FromEcho(c)

			// Extract the token from the Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				log.Warn("Missing authorization header")
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing authorization header"})
			}

			// Check if the header format is valid
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Warn("Invalid authorization header format")
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid authorization header format"})
			}

			tokenString := parts[1]

			// Validate the token
			claims, err := jwtUtil.ValidateToken(tokenString)
			if err != nil {
				log.Warn("Invalid or expired token", zap.Error(err))
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or expired token"})
			}

			// Store the claims in the context for later use
			c.Set("user", claims)
			log.Debug("JWT token validated successfully",
				zap.Uint("user_id", claims.UserID),
				zap.String("email", claims.Email))

			return next(c)
		}
	}
}
