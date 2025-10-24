// internal/config/config.go
package config

import (
	"log"
	"os"
	"fmt"

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

// GetCloudinaryURL mengambil URL Cloudinary dari environment
func GetCloudinaryURL() string {
	// Format: cloudinary://<api_key>:<api_secret>@<cloud_name>
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")

	if apiKey == "" || apiSecret == "" || cloudName == "" {
		log.Fatal("Cloudinary credentials (API_KEY, API_SECRET, CLOUD_NAME) must be set in .env file")
	}
	return fmt.Sprintf("cloudinary://%s:%s@%s", apiKey, apiSecret, cloudName)
}