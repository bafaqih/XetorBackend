// internal/repository/user_repo.go
package repository

import (
	"database/sql"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"xetor.id/backend/internal/domain/user"
)

type UserRepository struct {
	db *sql.DB
}

// NewUserRepository membuat instance baru dari UserRepository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Save menyimpan user baru ke database
func (r *UserRepository) Save(u *user.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Ambil URL default dari environment variable (lebih baik dari hardcode)
	defaultPhotoURL := os.Getenv("DEFAULT_PHOTO_URL")
	if defaultPhotoURL == "" {
		// Fallback jika tidak ada di .env (sebaiknya selalu ada)
		defaultPhotoURL = "https://res.cloudinary.com/db6vyj3n9/image/upload/v1761057514/onzkl78dvit7fyiqiphx.jpg"
		log.Println("WARNING: DEFAULT_PHOTO_URL not set in .env, using fallback.")
	}

	// --- PERUBAHAN DI SINI ---
	// Tambahkan kolom 'photo' dan $5 ke query
	query := "INSERT INTO users (fullname, email, phone, password, photo) VALUES ($1, $2, $3, $4, $5)"
	// Masukkan defaultPhotoURL sebagai argumen kelima
	_, err = r.db.Exec(query, u.Fullname, u.Email, u.Phone, string(hashedPassword), defaultPhotoURL)
	// -------------------------

	if err != nil {
		log.Printf("Error saving user to DB: %v", err)
		return err
	}

	log.Printf("User with email %s successfully saved to database.", u.Email)
	return nil
}

func (r *UserRepository) FindByEmail(email string) (*user.User, error) {
	// --- PERUBAHAN DI SINI ---
	// Tambahkan 'photo' ke query SELECT
	query := "SELECT id, fullname, email, phone, password, photo FROM users WHERE email = $1"

	var u user.User
	// Tambahkan &u.Photo ke Scan
	err := r.db.QueryRow(query, email).Scan(&u.ID, &u.Fullname, &u.Email, &u.Phone, &u.Password, &u.Photo)
	// -------------------------

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}
