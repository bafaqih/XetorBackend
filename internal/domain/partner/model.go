package partner

import (
	"database/sql"
	"strings"
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
	Token  string `json:"token"`
	Status string `json:"status"`
}

// UpdatePartnerProfileRequest data untuk update profil partner
type UpdatePartnerProfileRequest struct {
	BusinessName string `json:"business_name"`
	Email        string `json:"email"`
	Phone        string `json:"phone"`
}

// ChangePasswordRequest data untuk request ganti password partner
type ChangePasswordRequest struct {
	OldPassword        string `json:"old_password" binding:"required"`
	NewPassword        string `json:"new_password" binding:"required,min=6"` // Validasi min 6 karakter
	ConfirmNewPassword string `json:"confirm_new_password" binding:"required"`
}

// PartnerAddress merepresentasikan data dari tabel partner_addresses
// Tidak menyertakan business_name dan phone karena diambil dari tabel partners
type PartnerAddress struct {
	ID          int            `json:"id"`
	PartnerID   int            `json:"partner_id"`
	Address     string         `json:"address"`
	CityRegency string         `json:"city_regency"`
	Province    string         `json:"province"`
	PostalCode  sql.NullString `json:"postal_code,omitempty"`
	Latitude    sql.NullFloat64 `json:"latitude,omitempty"`    // Koordinat latitude
	Longitude   sql.NullFloat64 `json:"longitude,omitempty"`   // Koordinat longitude
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// UpdatePartnerAddressRequest data untuk membuat/mengupdate alamat partner
type UpdatePartnerAddressRequest struct {
	Address     string   `json:"address" binding:"required"`
	CityRegency string   `json:"city_regency" binding:"required"`
	Province    string   `json:"province" binding:"required"`
	PostalCode  string   `json:"postal_code"`        // Opsional
	Latitude    *float64 `json:"latitude"`          // Opsional, pointer agar bisa null
	Longitude   *float64 `json:"longitude"`         // Opsional, pointer agar bisa null
}

// PartnerSchedule merepresentasikan satu baris data dari tabel partner_schedules (struktur baru)
type PartnerSchedule struct {
	ID              int       `json:"id"`
	PartnerID       int       `json:"partner_id"`
	DaysOpen        []string  `json:"days_open"`        // Kirim sebagai array string ["Senin", "Selasa"]
	OpenTime        string    `json:"open_time"`        // Format "HH:MM"
	CloseTime       string    `json:"close_time"`       // Format "HH:MM"
	OperatingStatus string    `json:"operating_status"` // "Open" atau "Closed"
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UpdateScheduleRequest data untuk mengupdate seluruh jadwal operasional (struktur baru)
type UpdateScheduleRequest struct {
	DaysOpen        []string `json:"days_open"`        // Array hari buka ["Senin", "Selasa", ...]
	OpenTime        string   `json:"open_time"`        // Format "HH:MM"
	CloseTime       string   `json:"close_time"`       // Format "HH:MM"
	OperatingStatus string   `json:"operating_status"` // "Open" atau "Closed"
}

// Helper function untuk mengubah string database ke slice
func DaysOpenFromString(dbString sql.NullString) []string {
	if !dbString.Valid || dbString.String == "" {
		return []string{} // Return slice kosong jika null atau string kosong
	}
	return strings.Split(dbString.String, ",") // Pisahkan string berdasarkan koma
}

// Helper function untuk mengubah slice ke string database
func DaysOpenToString(days []string) sql.NullString {
	if len(days) == 0 {
		return sql.NullString{Valid: false} // Simpan sebagai NULL jika array kosong
	}
	return sql.NullString{String: strings.Join(days, ","), Valid: true} // Gabungkan dengan koma
}

// PartnerWastePriceDetail merepresentasikan satu baris dari partner_waste_price_details
type PartnerWastePriceDetail struct {
	ID                  int            `json:"id"`
	PartnerWastePriceID int            `json:"partner_waste_price_id"` // FK ke header
	WasteDetailID       sql.NullInt32  `json:"waste_detail_id,omitempty"`
	Image               sql.NullString `json:"image,omitempty"` // URL Gambar Cloudinary
	Name                string         `json:"name"`            // Nama jenis sampah (misal: Botol PET)
	Price               string         `json:"price"`           // Harga (Rp), kirim sebagai string
	Unit                string         `json:"unit"`            // Satuan (kg, pcs)
	Xpoin               int            `json:"xpoin"`           // Poin yg didapat user
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// WastePriceRequest digunakan untuk Create (POST) dan Update (PUT)
// Karena melibatkan file upload, kita akan baca field ini dari form-data, bukan JSON
type WastePriceRequest struct {
	Name          string  `form:"name" binding:"required"`
	Price         float64 `form:"price" binding:"required,gt=0"`
	Unit          string  `form:"unit" binding:"required"`
	WasteDetailID int     `form:"waste_detail_id" binding:"required"`
	// Image *multipart.FileHeader `form:"image"` // Ditangani terpisah di handler
}

// UpdateWastePriceRequest data UNTUK UPDATE harga sampah (fields opsional)
type UpdateWastePriceRequest struct {
	Name          string  `form:"name"`                           // Opsional
	Price         float64 `form:"price" binding:"omitempty,gt=0"` // Opsional, tapi jika ada > 0
	Unit          string  `form:"unit"`                           // Opsional
	WasteDetailID *int    `form:"waste_detail_id"`
	// Image *multipart.FileHeader `form:"image"` // Ditangani terpisah
}

// PartnerTransactionHistoryItem adalah format standar untuk riwayat transaksi finansial gabungan partner
type PartnerTransactionHistoryItem struct {
	ID          string         `json:"id"`               // ID unik (misal: "withdraw-5", "topup-2")
	Type        string         `json:"type"`             // 'withdraw', 'topup', 'convert', 'transfer'
	Amount      sql.NullString `json:"amount,omitempty"` // Jumlah (Rp)
	Points      sql.NullInt32  `json:"points,omitempty"` // Jumlah Xpoin (untuk convert)
	Status      string         `json:"status"`
	Timestamp   time.Time      `json:"timestamp"`
	Description string         `json:"description"` // Deskripsi singkat
}

// PartnerConversionHistory merepresentasikan data dari tabel partner_conversion_histories
// (Perlu didefinisikan jika belum)
type PartnerConversionHistory struct {
	ID             int       `json:"id"`
	PartnerID      int       `json:"partner_id"`
	Type           string    `json:"type"`
	AmountXp       int       `json:"amount_xp"`
	AmountRp       string    `json:"amount_rp"`
	Rate           float64   `json:"rate"`
	ConversionTime time.Time `json:"conversion_time"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// PartnerWallet merepresentasikan data dari tabel partner_wallets
type PartnerWallet struct {
	ID        int       `json:"id"`
	PartnerID int       `json:"partner_id"`
	Balance   string    `json:"balance"` // Kirim sebagai string
	Xpoin     int       `json:"xpoin"`   // Harus int
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PartnerStatistic merepresentasikan data dari tabel partner_statistics
type PartnerStatistic struct {
	ID          int       `json:"id"`
	PartnerID   int       `json:"partner_id"`
	Waste       string    `json:"waste"`       // Total sampah (kg), kirim sbg string
	Revenue     string    `json:"revenue"`     // Total pendapatan (Rp), kirim sbg string
	Customer    int       `json:"customer"`    // Jumlah unique customer
	Transaction int       `json:"transaction"` // Jumlah transaksi
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// PartnerWithdrawRequest data untuk request penarikan saldo partner
type PartnerWithdrawRequest struct {
	PaymentMethodID   int     `json:"payment_method_id" binding:"required"`
	AccountNumber     string  `json:"account_number" binding:"required"`
	Amount            float64 `json:"amount" binding:"required,gt=0"`
	AccountHolderName string  `json:"account_holder_name"` // Opsional, tergantung bank
}

// PartnerTopupRequest data untuk request top up saldo partner
type PartnerTopupRequest struct {
	PaymentMethodID int     `json:"payment_method_id" binding:"required"`
	Amount          float64 `json:"amount" binding:"required,gt=0"` // Jumlah > 0
}

// PartnerTransferRequest data untuk request transfer Xpoin dari partner
type PartnerTransferRequest struct {
	RecipientEmail string `json:"recipient_email" binding:"required,email"`
	Amount         int    `json:"amount" binding:"required,gt=0"` // Xpoin > 0
}

// PartnerConversionRequest data umum untuk request konversi partner
type PartnerConversionRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"` // Jumlah Xp atau Rp
}

// VerifyQrTokenRequest data yang dikirim partner saat scan QR
type VerifyQrTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// VerifyQrTokenResponse data user yang dikirim kembali jika token valid
type VerifyQrTokenResponse struct {
	UserID   int    `json:"user_id"`
	Fullname string `json:"fullname"`
	Email    string `json:"email"`
}

// CheckUserRequest data yang dikirim partner untuk cek user
type CheckUserRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// CheckUserResponse data user yang dikirim kembali jika ditemukan
// (Sama seperti VerifyQrTokenResponse)
type CheckUserResponse struct {
	UserID   int    `json:"user_id"`
	Fullname string `json:"fullname"`
	Email    string `json:"email"`
}

// --- Structs untuk GET History Deposit ---

// DepositHistoryDetailItem merepresentasikan satu item sampah dalam satu transaksi deposit
type DepositHistoryDetailItem struct {
	ID                      int            `json:"id"`
	PartnerDepositHistoryID int            `json:"partner_deposit_history_id"`
	WasteDetailID           sql.NullInt32  `json:"waste_detail_id,omitempty"`
	WasteName               sql.NullString `json:"waste_name,omitempty"`   // Nama dari waste_types
	WasteWeight             sql.NullString `json:"waste_weight,omitempty"` // Berat (kg), sbg string
	Xpoin                   int            `json:"xpoin"`
	Photo                   sql.NullString `json:"photo,omitempty"` // URL Foto bukti
	Notes                   sql.NullString `json:"notes,omitempty"`
	Status                  string         `json:"status"`
}

// DepositHistoryHeader merepresentasikan satu transaksi deposit (header)
type DepositHistoryHeader struct {
	ID              int                        `json:"id"`
	PartnerID       int                        `json:"partner_id"`
	UserID          int                        `json:"user_id"`
	UserName        sql.NullString             `json:"user_name,omitempty"`    // Nama user penyetor
	UserEmail       sql.NullString             `json:"user_email,omitempty"`   // Email user penyetor
	TotalWeight     sql.NullString             `json:"total_weight,omitempty"` // Berat total (kg), sbg string
	TotalXpoin      int                        `json:"total_xpoin"`
	TransactionTime time.Time                  `json:"transaction_time"`
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
	Details         []DepositHistoryDetailItem `json:"details"` // Slice untuk menampung detail item
}

// --- Structs untuk Create Deposit ---

// DepositWasteItem representasi satu item sampah dalam JSON string di request
type DepositWasteItem struct {
	PartnerWastePriceDetailID int     `json:"partner_waste_price_detail_id"` // ID dari partner_waste_price_details
	Weight                    float64 `json:"weight"`                        // Berat dalam KG
	// Field internal untuk kalkulasi service, BUKAN dari JSON request
	CalculatedXpoin int           `json:"-"` // Akan diisi oleh service
	WasteDetailID   sql.NullInt32 `json:"-"` // Akan diisi oleh service
}

// CreateDepositRequest data yang diterima dari partner via multipart/form-data
type CreateDepositRequest struct {
	UserID          int    `form:"user_id" binding:"required"`           // ID User Penyetor
	DepositMethodID int    `form:"deposit_method_id" binding:"required"` // ID Metode Deposit (DropOff/PickUp)
	ItemsJSON       string `form:"items_json" binding:"required"`        // JSON string dari []DepositWasteItem
	Notes           string `form:"notes"`                                // Catatan opsional
	// Photo *multipart.FileHeader `form:"photo"` // Akan diambil manual di handler
}

// WastePriceInfo struct helper untuk mengambil data harga/poin
type WastePriceInfo struct {
	PricePerUnit  float64
	XpoinPerUnit  int
	Unit          string
	WasteDetailID sql.NullInt32 // Foreign Key ke waste_details
}

// ArgsDepositCreation struct untuk parameter fungsi transaksi deposit
type ArgsDepositCreation struct {
	PartnerID        int
	UserID           int
	DepositMethodID  int
	Items            []DepositWasteItem // Slice dari item yang disetor (sudah dihitung Xpoin & WasteDetailID)
	TotalWeight      float64
	TotalXpoin       int
	Notes            sql.NullString
	PhotoURL         sql.NullString
	TransactionTime  time.Time // Waktu transaksi aktual
}