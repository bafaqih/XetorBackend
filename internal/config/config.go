// internal/config/config.go
package config

import (
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

// GetDefaultPhotoURL mengembalikan URL foto profil default.
// Contoh di VPS: DEFAULT_PHOTO_URL=https://cdn.xetor.bafagih.my.id/profile/default.jpg
// Jika tidak di-set, akan menggunakan fallback CDN URL.
func GetDefaultPhotoURL() string {
	defaultPhotoURL := os.Getenv("DEFAULT_PHOTO_URL")
	if defaultPhotoURL == "" {
		// Fallback jika tidak ada di .env (sebaiknya selalu ada) - gunakan CDN VPS
		defaultPhotoURL = "https://cdn.xetor.bafagih.my.id/profile/default.jpg"
		log.Println("WARNING: DEFAULT_PHOTO_URL not set in .env, using fallback CDN URL.")
	}
	return defaultPhotoURL
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
