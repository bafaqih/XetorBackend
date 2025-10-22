// internal/auth/jwt.go
package auth

import (
	"time"
	"strconv"


	"github.com/golang-jwt/jwt/v5"
	"xetor.id/backend/internal/config"
)

func GenerateToken(userID int) (string, error) {
	// Ambil kunci rahasia dari config, bukan hardcode lagi
	jwtSecretKey := config.GetJWTSecret()

	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &jwt.RegisteredClaims{
    Subject:   strconv.Itoa(userID),
    ExpiresAt: jwt.NewNumericDate(expirationTime),
}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}