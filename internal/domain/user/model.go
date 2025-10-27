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

// UserWallet merepresentasikan data dari tabel user_wallets
type UserWallet struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Balance   string    `json:"balance"` // Kirim sebagai string agar presisi terjaga
	Xpoin     int       `json:"xpoin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserStatistic merepresentasikan data dari tabel user_statistics
type UserStatistic struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Waste     string    `json:"waste"`     // Total sampah (kg), kirim sbg string
	Energy    string    `json:"energy"`    // Energi dihemat (kWh), kirim sbg string
	CO2       string    `json:"co2"`       // CO2 terselamatkan (kg), kirim sbg string
	Water     string    `json:"water"`     // Air dihemat (L), kirim sbg string
	Tree      int       `json:"tree"`      // Pohon terselamatkan
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WithdrawRequest data untuk request penarikan saldo
type WithdrawRequest struct {
	PaymentMethodID int     `json:"payment_method_id" binding:"required"`
	AccountNumber   string  `json:"account_number" binding:"required"`
	Amount          float64 `json:"amount" binding:"required,gt=0"`
	AccountHolderName string `json:"account_holder_name"`
}

// TopupRequest data untuk request top up saldo
type TopupRequest struct {
	PaymentMethodID int     `json:"payment_method_id" binding:"required"`
	Amount          float64 `json:"amount" binding:"required,gt=0"` // Jumlah harus lebih besar dari 0
}

// TransferRequest data untuk request transfer Xpoin
type TransferRequest struct {
	RecipientEmail string `json:"recipient_email" binding:"required,email"` // Validasi email
	Amount         int    `json:"amount" binding:"required,gt=0"`         // Jumlah Xpoin harus > 0
}

// ConversionRequest data umum untuk request konversi
type ConversionRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"` // Jumlah Xp atau Rp
}

// ConversionHistory merepresentasikan data dari tabel user_conversion_histories
type ConversionHistory struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`
	Type           string    `json:"type"`
	AmountXp       int       `json:"amount_xp"`
	AmountRp       string    `json:"amount_rp"` // Kirim sebagai string
	Rate           float64   `json:"rate"`
	ConversionTime time.Time `json:"conversion_time"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// GenerateQrTokenResponse data respons untuk pembuatan token QR
type GenerateQrTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"` // Waktu kedaluwarsa dalam format timestamp
}

// UpdateUserProfileRequest data untuk update profil user
type UpdateUserProfileRequest struct {
	Fullname string `json:"fullname"`
	Email    string `json:"email"` // Tanpa validasi email unik di binding, cek di service/repo
	Phone    string `json:"phone"` // Bisa string kosong jika ingin menghapus
}

// ImpactFactors struct helper untuk menyimpan faktor dampak lingkungan per jenis sampah
type ImpactFactors struct {
	Energy float64 // Contoh: kWh per kg
	CO2    float64 // Contoh: kg CO2 per kg
	Water  float64 // Contoh: Liter per kg
	Tree   float64 // Contoh: Pohon per kg (nanti dibulatkan)
}