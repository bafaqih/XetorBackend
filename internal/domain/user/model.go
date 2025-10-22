package user

import (
	"database/sql"
	"time"
)

// User adalah representasi data user di dalam database
type User struct {
	ID       int            `json:"id"`
	Fullname string         `json:"fullname"`
	Email    string         `json:"email"`
	Phone    sql.NullString `json:"phone,omitempty"` // omitempty agar tidak muncul jika null
	Password string         `json:"-"`
	Photo    sql.NullString `json:"photo,omitempty"` // omitempty agar tidak muncul jika null
}

// SignUpRequest adalah data yang kita harapkan dari request API
type SignUpRequest struct {
	Fullname string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

// ChangePasswordRequest adalah data yang kita harapkan dari request API untuk mengganti password
type ChangePasswordRequest struct {
	OldPassword        string `json:"old_password" binding:"required"`
	NewPassword        string `json:"new_password" binding:"required,min=6"` // Tambahkan validasi minimal panjang
	ConfirmNewPassword string `json:"confirm_new_password" binding:"required"`
}

// UserAddress merepresentasikan data dari tabel user_addresses
type UserAddress struct {
	ID          int            `json:"id"`
	UserID      int            `json:"user_id"` // Foreign key ke users
	Fullname    string         `json:"fullname"`
	Phone       string         `json:"phone"`
	Address     string         `json:"address"`
	CityRegency string         `json:"city_regency"`
	Province    string         `json:"province"`
	PostalCode  sql.NullString `json:"postal_code,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// CreateUserAddressRequest data untuk membuat alamat baru
type CreateUserAddressRequest struct {
	Fullname    string `json:"fullname" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	Address     string `json:"address" binding:"required"`
	CityRegency string `json:"city_regency" binding:"required"`
	Province    string `json:"province" binding:"required"`
	PostalCode  string `json:"postal_code"` // Opsional
}

// UpdateUserAddressRequest data untuk mengupdate alamat
type UpdateUserAddressRequest struct {
	Fullname    string `json:"fullname"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	CityRegency string `json:"city_regency"`
	Province    string `json:"province"`
	PostalCode  string `json:"postal_code"`
}

// TransactionHistoryItem adalah format standar untuk riwayat transaksi gabungan
type TransactionHistoryItem struct {
	ID          string         `json:"id"`           // ID unik (misal: "deposit-1", "withdraw-5")
	Type        string         `json:"type"`         // 'deposit', 'withdraw', 'topup', 'transfer'
	Amount      sql.NullString `json:"amount,omitempty"` // Jumlah (Rp) untuk withdraw, topup, transfer
	Points      sql.NullInt32  `json:"points,omitempty"` // Jumlah poin untuk deposit
	Status      string         `json:"status"`
	Timestamp   time.Time      `json:"timestamp"`
	Description string         `json:"description"` // Deskripsi singkat (misal: "Withdraw ke BCA", "Topup via GoPay", "Transfer ke email@...", "Deposit Sampah")
}