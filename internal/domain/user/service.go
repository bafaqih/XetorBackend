// internal/domain/user/service.go
package user

import (
	"errors"
	"log"

	"golang.org/x/crypto/bcrypt"
)

type Repository interface {
	Save(user *User) error
	FindByEmail(email string) (*User, error)
}

type Service struct {
	repo Repository
}

// NewService membuat instance baru dari Service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// RegisterUser memproses data dari handler dan menyimpannya
func (s *Service) RegisterUser(req SignUpRequest) error {
	user := &User{
		Fullname: req.Fullname,
		Email:    req.Email,
		Phone:    req.Phone,
		Password: req.Password,
	}

	return s.repo.Save(user)
}

// Login memvalidasi kredensial pengguna
func (s *Service) ValidateLogin(email, password string) (*User, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, err // Error teknis dari database
	}
	if user == nil {
		return nil, errors.New("kredensial tidak valid") // User tidak ditemukan
	}

	// Bandingkan password yang diinput dengan hash di database
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	log.Println("Bcrypt comparison error:", err) // Log untuk debugging
	if err != nil {
		return nil, errors.New("kredensial tidak valid") // Password tidak cocok
	}

	return user, nil // Kembalikan data user jika berhasil
}
