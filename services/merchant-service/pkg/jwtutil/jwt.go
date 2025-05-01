package jwtutil

import (
	"github.com/golang-jwt/jwt/v4"
)

var secret = []byte("secret-key")

// UserClaims represents the JWT claims for user authentication
type UserClaims struct {
	Email      string `json:"email"`
	UserID     uint   `json:"user_id"`
	MerchantID *uint  `json:"merchant_id,omitempty"`
	jwt.RegisteredClaims
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
