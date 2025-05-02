package jwtutil

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

var secret = []byte("secret-key")

// UserClaims represents the JWT claims for user authentication
type UserClaims struct {
	Email      string `json:"email"`
	UserID     uint   `json:"user_id"`
	MerchantID *uint  `json:"merchant_id,omitempty"` // Keeping for backward compatibility
	TenantID   *uint  `json:"tenant_id,omitempty"`   // Adding tenant ID for multi-tenancy
	TenantName string `json:"tenant_name,omitempty"` // Adding tenant name for convenience
	Role       string `json:"role,omitempty"`        // User's role in the current tenant
	jwt.RegisteredClaims
}

// GenerateToken creates a JWT token with user information
func GenerateToken(email string, userID uint, merchantID *uint) (string, error) {
	return GenerateTokenWithTenant(email, userID, merchantID, nil, "", "")
}

// GenerateTokenWithTenant creates a JWT token with user and tenant information
func GenerateTokenWithTenant(email string, userID uint, merchantID *uint, tenantID *uint, tenantName string, role string) (string, error) {
	claims := UserClaims{
		Email:      email,
		UserID:     userID,
		MerchantID: merchantID,
		TenantID:   tenantID,
		TenantName: tenantName,
		Role:       role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
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
