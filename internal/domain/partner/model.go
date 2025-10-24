package partner

import (
	"database/sql"
	"time"
)

// Partner merepresentasikan data partner dari tabel partners
type Partner struct {
	ID           int            `json:"id"`
	BusinessName string         `json:"business_name"`
	Email        string         `json:"email"`
	Phone        sql.NullString `json:"phone,omitempty"`
	Password     string         `json:"-"` // Jangan kirim password hash
	Photo        sql.NullString `json:"photo,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// PartnerSignUpRequest data untuk request registrasi partner
type PartnerSignUpRequest struct {
	BusinessName string `json:"business_name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Phone        string `json:"phone" binding:"required"` // Tambahkan validasi format phone jika perlu
	Password     string `json:"password" binding:"required,min=6"`
}

// PartnerLoginRequest data untuk request login partner (sama seperti user)
type PartnerLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// PartnerLoginResponse data respons setelah login berhasil
type PartnerLoginResponse struct {
	Token   string   `json:"token"`
	Status  string   `json:"status"`
}

// UpdatePartnerProfileRequest data untuk update profil partner
type UpdatePartnerProfileRequest struct {
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
}