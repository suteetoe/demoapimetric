package jwtutil

import (
	"auth-service/pkg/config"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var jwtConfig *config.JWTConfig

// UserClaims represents the JWT claims for user authentication
type UserClaims struct {
	Email      string `json:"email"`
	UserID     uint   `json:"user_id"`
	TenantID   *uint  `json:"tenant_id,omitempty"`   // Adding tenant ID for multi-tenancy
	TenantName string `json:"tenant_name,omitempty"` // Adding tenant name for convenience
	Role       string `json:"role,omitempty"`        // User's role in the current tenant
	jwt.RegisteredClaims
}

// Initialize sets up the JWT utility with configuration
func Initialize(config *config.JWTConfig) {
	jwtConfig = config
}

// GenerateToken creates a JWT token with user information
func GenerateToken(email string, userID uint) (string, error) {
	return GenerateTokenWithTenant(email, userID, nil, "", "")
}

// GenerateTokenWithTenant creates a JWT token with user and tenant information
func GenerateTokenWithTenant(email string, userID uint, tenantID *uint, tenantName string, role string) (string, error) {
	if jwtConfig == nil {
		return "", errors.New("JWT configuration not initialized")
	}

	// Get signing key and expiration from configuration
	signingKey := jwtConfig.SigningKey
	expirationHours := jwtConfig.ExpirationHours

	claims := UserClaims{
		Email:      email,
		UserID:     userID,
		TenantID:   tenantID,
		TenantName: tenantName,
		Role:       role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expirationHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(signingKey))
}

// ValidateToken validates and parses the JWT token
func ValidateToken(tokenString string) (*UserClaims, error) {
	if jwtConfig == nil {
		return nil, errors.New("JWT configuration not initialized")
	}

	// Get signing key from configuration
	signingKey := jwtConfig.SigningKey

	token, err := jwt.ParseWithClaims(
		tokenString,
		&UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(signingKey), nil
		},
	)

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
