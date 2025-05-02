package jwtutil

import (
	"errors"
	"fmt"
	"product-service/pkg/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtConfig *config.JWTConfig

// TenantClaims extends jwt.RegisteredClaims to include tenant information
// Renamed from UserClaims to TenantClaims for consistency with supplier-service
type TenantClaims struct {
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

// GenerateToken creates a new JWT token for a user
func GenerateToken(email string, userID uint) (string, error) {
	return generateTokenWithClaims(email, userID, nil, "", "")
}

// GenerateTokenWithTenant creates a new JWT token with tenant context
func GenerateTokenWithTenant(email string, userID uint, tenantID *uint, tenantName, role string) (string, error) {
	return generateTokenWithClaims(email, userID, tenantID, tenantName, role)
}

// generateTokenWithClaims is a helper function that creates a token with the given claims
func generateTokenWithClaims(email string, userID uint, tenantID *uint, tenantName, role string) (string, error) {
	if jwtConfig == nil {
		return "", errors.New("JWT configuration not initialized")
	}

	// Get signing key from configuration
	signingKey := jwtConfig.SigningKey

	// Token expiration time from configuration
	expirationHours := jwtConfig.ExpirationHours

	// Create the claims
	claims := &TenantClaims{
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

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token
	return token.SignedString([]byte(signingKey))
}

// ExtractTenantID extracts tenant ID from JWT token string
// This function is maintained from the product-service as it adds value
func ExtractTenantID(tokenString string) (*uint, error) {
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return claims.TenantID, nil
}

// ValidateToken validates and parses the JWT token
func ValidateToken(tokenString string) (*TenantClaims, error) {
	if jwtConfig == nil {
		return nil, errors.New("JWT configuration not initialized")
	}

	// Get signing key from configuration
	signingKey := jwtConfig.SigningKey

	// Parse the token
	token, err := jwt.ParseWithClaims(
		tokenString,
		&TenantClaims{},
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

	// Validate the token and extract claims
	if claims, ok := token.Claims.(*TenantClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
