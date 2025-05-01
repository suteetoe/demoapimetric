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
	MerchantID *uint  `json:"merchant_id,omitempty"`
	jwt.RegisteredClaims
}

// GenerateToken creates a JWT token with user information
func GenerateToken(email string, userID uint, merchantID *uint) (string, error) {
	claims := UserClaims{
		Email:      email,
		UserID:     userID,
		MerchantID: merchantID,
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
