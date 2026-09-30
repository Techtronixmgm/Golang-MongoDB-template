package auth

import "github.com/golang-jwt/jwt/v5"

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`

	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID string `json:"user_id"`
	Type   string `json:"type"`

	jwt.RegisteredClaims
}

type TwoFactorClaims struct {
	UserID string `json:"userId"`
	Type   string `json:"type"`

	jwt.RegisteredClaims
}
