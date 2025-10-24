// internal/auth/jwt.go
package auth

import (
	"time"
	"strconv"


	"github.com/golang-jwt/jwt/v5"
	"xetor.id/backend/internal/config"
)

type JwtCustomClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func GenerateToken(entityID int, role string) (string, error) {
	jwtSecretKey := config.GetJWTSecret()
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &JwtCustomClaims{ // Gunakan struct custom
		role, // Isi role
		jwt.RegisteredClaims{
			Subject:   strconv.Itoa(entityID), // ID User atau Partner
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}