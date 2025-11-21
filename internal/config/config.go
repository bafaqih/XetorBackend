// internal/config/config.go
package config

import (
	"fmt"
	"log"
	"os"
	"strings"

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

// GetMediaBasePath mengembalikan direktori dasar untuk menyimpan file media (gambar, dll).
// Di VPS sebaiknya di-set, misal: MEDIA_BASE_PATH=/var/www/xetor/images
// Untuk development lokal, default ke "./media" jika tidak di-set.
func GetMediaBasePath() string {
	basePath := os.Getenv("MEDIA_BASE_PATH")
	if basePath == "" {
		basePath = "./media"
	}
	return basePath
}

// GetCDNBaseURL mengembalikan base URL untuk mengakses file media melalui CDN / domain statis.
// Contoh di VPS: CDN_BASE_URL=https://cdn.xetor.bafagih.my.id
func GetCDNBaseURL() string {
	baseURL := os.Getenv("CDN_BASE_URL")
	if baseURL == "" {
		log.Fatal("CDN_BASE_URL must be set in .env file")
	}
	// Pastikan tidak ada trailing slash agar mudah di-join
	return strings.TrimRight(baseURL, "/")
}

// GetGoogleClientID mengambil nilai GOOGLE_CLIENT_ID dari environment
func GetGoogleClientID() string {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		// Log.Fatal akan menghentikan aplikasi jika ID tidak ada
		log.Fatal("GOOGLE_CLIENT_ID must be set in .env file")
	}
	return clientID
}