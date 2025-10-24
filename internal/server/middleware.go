package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"xetor.id/backend/internal/config"
	"xetor.id/backend/internal/auth"
)

// AuthMiddleware memvalidasi token JWT dan menyimpan ID serta Role ke context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header dibutuhkan"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Format Authorization header salah (harus: Bearer <token>)"})
			return
		}

		tokenString := parts[1]
		secretKey := config.GetJWTSecret()

		// Parse token menggunakan custom claims struct
		claims := &auth.JwtCustomClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("metode signing tidak terduga: %v", token.Header["alg"])
			}
			return secretKey, nil
		})

		if err != nil {
			status := http.StatusUnauthorized
			errMsg := "Token tidak valid atau kedaluwarsa"
			// Cek spesifik error kedaluwarsa
			if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
                errMsg = "Token sudah kedaluwarsa atau belum valid"
            }
			c.AbortWithStatusJSON(status, gin.H{"error": errMsg})
			return
		}

		if token.Valid {
            // Cek expiry sekali lagi untuk keamanan ganda
            if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
                 c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token sudah kedaluwarsa"})
                 return
            }

			// Simpan entityID (Subject) dan Role ke context
			c.Set("entityID", claims.Subject) // ID User atau Partner sebagai string
			c.Set("role", claims.Role)      // Role ("user" atau "partner")

			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid"})
		}
	}
}

// Helper Middleware (Opsional tapi berguna): RoleCheckMiddleware
// Middleware ini bisa dipasang SETELAH AuthMiddleware untuk memastikan role tertentu
func RoleCheckMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Role tidak ditemukan"})
			return
		}

		role, ok := roleValue.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Format role tidak valid"})
			return
		}

		isAllowed := false
		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Akses ditolak untuk role ini"})
			return
		}

		c.Next()
	}
}