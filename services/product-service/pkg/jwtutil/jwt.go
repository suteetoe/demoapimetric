package jwtutil

import (
	"github.com/golang-jwt/jwt/v4"
)

var secret = []byte("secret-key")

// UserClaims represents the JWT claims for user authentication
type UserClaims struct {
	Email      string `json:"email"`
	UserID     uint   `json:"user_id"`
	TenantID   *uint  `json:"tenant_id,omitempty"`   // Adding tenant ID for multi-tenancy
	TenantName string `json:"tenant_name,omitempty"` // Adding tenant name for convenience
	Role       string `json:"role,omitempty"`        // User's role in the current tenant
	jwt.RegisteredClaims
}

// ExtractTenantID extracts tenant ID from JWT token string
func ExtractTenantID(tokenString string) (*uint, error) {
	claims, err := ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	return claims.TenantID, nil
}

// ValidateToken validates and parses the JWT token
func ValidateToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
