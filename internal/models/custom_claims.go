package models

import "github.com/golang-jwt/jwt/v5"

// CustomClaims represents the custom claims in our JWT tokens
type CustomClaims struct {
	jwt.RegisteredClaims
	UserID    string `json:"user_id"`
	Email     string `json:"email,omitempty"`
	Role      string `json:"role,omitempty"`
	TokenType string `json:"token_type"`
}
