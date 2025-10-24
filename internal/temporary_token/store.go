package temporary_token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"strconv"
	"sync" // Untuk menangani akses bersamaan ke map (goroutine safety)
	"time"
)

// TokenData menyimpan informasi yang terkait dengan token
type TokenData struct {
	UserID    int       // ID pengguna yang memiliki token
	ExpiresAt time.Time // Waktu kedaluwarsa token
}

// TokenStore menyimpan token aktif di memori
type TokenStore struct {
	mu     sync.RWMutex // Mutex untuk melindungi akses ke map
	tokens map[string]TokenData
}

// NewTokenStore membuat instance TokenStore baru
func NewTokenStore() *TokenStore {
	store := &TokenStore{
		tokens: make(map[string]TokenData),
	}
	// Jalankan pembersihan token kedaluwarsa secara berkala (misal: setiap menit)
	go store.cleanupExpiredTokens(1 * time.Minute)
	return store
}

// generateSecureToken membuat string acak yang aman
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateToken membuat token baru untuk user, menyimpannya, dan mengembalikan token + expiry
func (s *TokenStore) CreateToken(userIDStr string, validityDuration time.Duration) (string, time.Time, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return "", time.Time{}, errors.New("ID pengguna tidak valid")
	}

	token, err := generateSecureToken(16) // Buat token 16 byte (32 karakter hex)
	if err != nil {
		return "", time.Time{}, errors.New("gagal membuat token")
	}

	expiresAt := time.Now().Add(validityDuration)
	data := TokenData{
		UserID:    userID,
		ExpiresAt: expiresAt,
	}

	s.mu.Lock() // Kunci map sebelum menulis
	s.tokens[token] = data
	s.mu.Unlock() // Buka kunci setelah selesai

	log.Printf("Generated QR token %s for user ID %d, expires at %s", token, userID, expiresAt.Format(time.RFC3339))
	return token, expiresAt, nil
}

// ValidateToken memeriksa apakah token ada dan belum kedaluwarsa, mengembalikan UserID
// (Fungsi ini akan dipakai oleh endpoint partner nanti)
func (s *TokenStore) ValidateToken(token string) (int, error) {
	s.mu.RLock() // Kunci map untuk membaca
	data, exists := s.tokens[token]
	s.mu.RUnlock() // Buka kunci setelah selesai

	if !exists {
		return 0, errors.New("token tidak ditemukan")
	}

	if time.Now().After(data.ExpiresAt) {
		// Hapus token yang sudah kedaluwarsa saat divalidasi
		s.mu.Lock()
		delete(s.tokens, token)
		s.mu.Unlock()
		return 0, errors.New("token sudah kedaluwarsa")
	}

	return data.UserID, nil
}

// cleanupExpiredTokens berjalan di background untuk menghapus token yang sudah lewat
func (s *TokenStore) cleanupExpiredTokens(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock() // Kunci map untuk operasi delete
		now := time.Now()
		deletedCount := 0
		for token, data := range s.tokens {
			if now.After(data.ExpiresAt) {
				delete(s.tokens, token)
				deletedCount++
			}
		}
		s.mu.Unlock() // Buka kunci setelah selesai
		if deletedCount > 0 {
			log.Printf("Cleaned up %d expired QR tokens", deletedCount)
		}
	}
}