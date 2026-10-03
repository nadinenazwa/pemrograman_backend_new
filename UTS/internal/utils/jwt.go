package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"siakad/config"
)

type JWTClaim struct {
	UserID int    `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(userID int, email string, role string) (string, int, error) {
	secret := config.GetEnv("JWT_SECRET", "supersecret")
	// For simplicity in UTS, expires in 24h
	expiresIn := 24 * 3600

	claims := &JWTClaim{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresIn) * time.Second)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, err := token.SignedString([]byte(secret))
	return t, expiresIn, err
}

func ValidateToken(signedToken string) (*JWTClaim, error) {
	secret := config.GetEnv("JWT_SECRET", "supersecret")
	token, err := jwt.ParseWithClaims(signedToken, &JWTClaim{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaim)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
