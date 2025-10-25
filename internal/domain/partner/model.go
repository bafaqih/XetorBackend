package partner

import (
	"database/sql"
	"time"
	"strings"
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
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// UpdatePartnerAddressRequest data untuk membuat/mengupdate alamat partner
type UpdatePartnerAddressRequest struct {
	Address     string `json:"address" binding:"required"`
	CityRegency string `json:"city_regency" binding:"required"`
	Province    string `json:"province" binding:"required"`
	PostalCode  string `json:"postal_code"` // Opsional
}

// PartnerSchedule merepresentasikan satu baris data dari tabel partner_schedules (struktur baru)
type PartnerSchedule struct {
	ID              int       `json:"id"`
	PartnerID       int       `json:"partner_id"`
	DaysOpen        []string  `json:"days_open"` // Kirim sebagai array string ["Senin", "Selasa"]
	OpenTime        string    `json:"open_time"`  // Format "HH:MM"
	CloseTime       string    `json:"close_time"` // Format "HH:MM"
	OperatingStatus string    `json:"operating_status"` // "Open" atau "Closed"
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// UpdateScheduleRequest data untuk mengupdate seluruh jadwal operasional (struktur baru)
type UpdateScheduleRequest struct {
	DaysOpen        []string `json:"days_open"` // Array hari buka ["Senin", "Selasa", ...]
	OpenTime        string   `json:"open_time"` // Format "HH:MM"
	CloseTime       string   `json:"close_time"` // Format "HH:MM"
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
	ID                   int            `json:"id"`
	PartnerWastePriceID  int            `json:"partner_waste_price_id"` // FK ke header
	Image                sql.NullString `json:"image,omitempty"`        // URL Gambar Cloudinary
	Name                 string         `json:"name"`                   // Nama jenis sampah (misal: Botol PET)
	Price                string         `json:"price"`                  // Harga (Rp), kirim sebagai string
	Unit                 string         `json:"unit"`                   // Satuan (kg, pcs)
	Xpoin                int            `json:"xpoin"`                  // Poin yg didapat user
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

// WastePriceRequest digunakan untuk Create (POST) dan Update (PUT)
// Karena melibatkan file upload, kita akan baca field ini dari form-data, bukan JSON
type WastePriceRequest struct {
	Name  string  `form:"name" binding:"required"`
	Price float64 `form:"price" binding:"required,gt=0"`
	Unit  string  `form:"unit" binding:"required"`
	// Image *multipart.FileHeader `form:"image"` // Ditangani terpisah di handler
}

// UpdateWastePriceRequest data UNTUK UPDATE harga sampah (fields opsional)
type UpdateWastePriceRequest struct {
	Name  string  `form:"name"`                      // Opsional
	Price float64 `form:"price" binding:"omitempty,gt=0"` // Opsional, tapi jika ada > 0
	Unit  string  `form:"unit"`                      // Opsional
	// Image *multipart.FileHeader `form:"image"` // Ditangani terpisah
}