package admin

import (
		"time"
		"database/sql"
)

// WasteType merepresentasikan data dari tabel waste_types
type WasteType struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateWasteTypeRequest adalah data yang dibutuhkan saat membuat WasteType baru
type CreateWasteTypeRequest struct {
	Name   string `json:"name" binding:"required"`
	Status string `json:"status"` // Status bisa opsional saat create, default di DB
}

// UpdateWasteTypeRequest adalah data yang dibutuhkan saat mengupdate WasteType
type UpdateWasteTypeRequest struct {
	Name   string `json:"name"`   // Opsional saat update
	Status string `json:"status"` // Opsional saat update
}

// WasteDetail merepresentasikan data dari tabel waste_details
type WasteDetail struct {
	ID                   int            `json:"id"`
	Name                 string         `json:"name"`
	WasteTypeID          sql.NullInt32  `json:"waste_type_id"` // Nullable Foreign Key
	ProperDisposalMethod sql.NullString `json:"proper_disposal_method"`
	PositiveImpact       sql.NullString `json:"positive_impact"`
	DecompositionTime    sql.NullString `json:"decomposition_time"`
	Xpoin                int            `json:"xpoin"`
	Status               string         `json:"status"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	// Tambahan: Untuk menampilkan nama Waste Type saat Get (optional)
	WasteTypeName        sql.NullString `json:"waste_type_name,omitempty"`
}

// CreateWasteDetailRequest data untuk membuat WasteDetail baru
type CreateWasteDetailRequest struct {
	Name                 string `json:"name" binding:"required"`
	WasteTypeID          *int   `json:"waste_type_id"` // Pointer agar bisa null
	ProperDisposalMethod string `json:"proper_disposal_method"`
	PositiveImpact       string `json:"positive_impact"`
	DecompositionTime    string `json:"decomposition_time"`
	Xpoin                int    `json:"xpoin"`
	Status               string `json:"status"`
}

// UpdateWasteDetailRequest data untuk mengupdate WasteDetail
type UpdateWasteDetailRequest struct {
	Name                 string `json:"name"`
	WasteTypeID          *int   `json:"waste_type_id"`
	ProperDisposalMethod string `json:"proper_disposal_method"`
	PositiveImpact       string `json:"positive_impact"`
	DecompositionTime    string `json:"decomposition_time"`
	Xpoin                *int   `json:"xpoin"` // Pointer agar bisa bedakan 0 dan tidak diisi
	Status               string `json:"status"`
}

// PaymentMethod merepresentasikan data dari tabel payment_methods
type PaymentMethod struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`        // e.g., "Gopay", "BCA Virtual Account"
	Type      string    `json:"type"`        // e.g., "e-wallet", "bank_transfer"
	Logo      string    `json:"logo"`        // URL logo
	Code      string    `json:"code"`        // Kode unik (opsional, bisa null)
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreatePaymentMethodRequest data untuk membuat PaymentMethod baru
type CreatePaymentMethodRequest struct {
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type" binding:"required"`
	Logo   string `json:"logo"`
	Code   string `json:"code"`
	Status string `json:"status"`
}

// UpdatePaymentMethodRequest data untuk mengupdate PaymentMethod
type UpdatePaymentMethodRequest struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Logo   string `json:"logo"`
	Code   string `json:"code"`
	Status string `json:"status"`
}

// DepositMethod merepresentasikan data dari tabel deposit_methods
type DepositMethod struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"` // e.g., "Drop Off", "Pickup"
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateDepositMethodRequest data untuk membuat DepositMethod baru
type CreateDepositMethodRequest struct {
	Name   string `json:"name" binding:"required"`
	Status string `json:"status"`
}

// UpdateDepositMethodRequest data untuk mengupdate DepositMethod
type UpdateDepositMethodRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// PromotionBanner merepresentasikan data dari tabel promotion_banners
type PromotionBanner struct {
	ID        int            `json:"id"`
	Name      string         `json:"name"`
	Image     string         `json:"image"` // URL gambar banner
	Link      sql.NullString `json:"link"`  // Link tujuan jika banner diklik (opsional)
	Status    string         `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// CreatePromotionBannerRequest data untuk membuat PromotionBanner baru
type CreatePromotionBannerRequest struct {
	Name   string `json:"name" binding:"required"`
	Image  string `json:"image" binding:"required"` // URL gambar wajib
	Link   string `json:"link"`                     // Link opsional
	Status string `json:"status"`
}

// UpdatePromotionBannerRequest data untuk mengupdate PromotionBanner
type UpdatePromotionBannerRequest struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	Link   string `json:"link"`
	Status string `json:"status"`
}

// AboutXetor merepresentasikan data dari tabel about_xetor
type AboutXetor struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`   // e.g., "version", "terms_conditions", "privacy_policy"
	Content   string    `json:"content"` // Isi teks yang panjang
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateAboutXetorRequest data untuk membuat AboutXetor baru
type CreateAboutXetorRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}

// UpdateAboutXetorRequest data untuk mengupdate AboutXetor
// Biasanya hanya konten yang diupdate, title jarang berubah
type UpdateAboutXetorRequest struct {
	Title   string `json:"title"`   // Opsional
	Content string `json:"content"` // Opsional
}

// XetorPartner merepresentasikan data dari tabel xetor_partners
type XetorPartner struct {
	ID        int       `json:"id"`
	PartnerID int       `json:"partner_id"` // Foreign key ke tabel partners
	Status    string    `json:"status"`   // e.g., "Pending", "Approved", "Rejected"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Tambahan: Untuk menampilkan nama Partner saat Get (optional)
	PartnerBusinessName sql.NullString `json:"partner_business_name,omitempty"`
}

// CreateXetorPartnerRequest data untuk membuat XetorPartner baru manual oleh Admin
type CreateXetorPartnerRequest struct {
	PartnerID int    `json:"partner_id" binding:"required"`
	Status    string `json:"status"` // Default di DB 'Pending'
}

// UpdateXetorPartnerRequest - Admin hanya bisa update status
type UpdateXetorPartnerRequest struct {
	Status string `json:"status" binding:"required"` // Status wajib diisi saat update
}