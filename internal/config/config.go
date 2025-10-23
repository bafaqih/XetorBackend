// internal/config/config.go
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// LoadConfig memuat variabel dari file .env
func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

// GetJWTSecret mengambil nilai JWT_SECRET_KEY dari environment
func GetJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET_KEY")
	if secret == "" {
		log.Fatal("JWT_SECRET_KEY must be set in .env file")
	}
	return []byte(secret)
}

// GetMidtransServerKey mengambil nilai MIDTRANS_SERVER_KEY dari environment
func GetMidtransServerKey() string {
    key := os.Getenv("MIDTRANS_SERVER_KEY")
    if key == "" {
        log.Fatal("MIDTRANS_SERVER_KEY must be set in .env file")
    }
    return key
}