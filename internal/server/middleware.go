package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"xetor.id/backend/internal/config"
)

// AuthMiddleware membuat middleware Gin untuk otentikasi JWT
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header dibutuhkan"})
			return
		}

		// Format header harus: Bearer <token>
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Format Authorization header salah (harus: Bearer <token>)"})
			return
		}

		tokenString := parts[1]
		secretKey := config.GetJWTSecret() // Ambil secret key dari .env

		// Parse dan validasi token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Pastikan algoritma signingnya adalah HMAC (HS256)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("metode signing tidak terduga: %v", token.Header["alg"])
			}
			return secretKey, nil
		})

		if err != nil {
			status := http.StatusUnauthorized
			errMsg := "Token tidak valid"
			if err == jwt.ErrTokenExpired {
				errMsg = "Token sudah kedaluwarsa"
			}
			c.AbortWithStatusJSON(status, gin.H{"error": errMsg})
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Cek apakah token masih valid (meskipun parsing berhasil, cek ulang expiry)
			if expFloat, ok := claims["exp"].(float64); ok {
				if float64(time.Now().Unix()) > expFloat {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token sudah kedaluwarsa"})
					return
				}
			} else {
                 c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Format klaim expiry tidak valid"})
                 return
            }


			// Ambil ID user dari klaim 'sub' (Subject)
			userIDStr, ok := claims["sub"].(string)
			if !ok {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Klaim user ID tidak ditemukan atau format salah"})
				return
			}

			// Simpan userID di context Gin agar bisa diakses oleh handler selanjutnya
			// Kita simpan sebagai string dulu, konversi ke int di handler jika perlu
			c.Set("userID", userIDStr)

			// Lanjutkan ke handler berikutnya
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token tidak valid atau klaim rusak"})
		}
	}
}