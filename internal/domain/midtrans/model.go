package midtrans

import (
	"time"

	"github.com/midtrans/midtrans-go"
)

// MidtransTransactionNotification adalah struct untuk payload webhook umum
// Referensi: https://docs.midtrans.com/en/after-payment/http-notification
type MidtransTransactionNotification struct {
	TransactionTime      string `json:"transaction_time"`       // Waktu transaksi (format: "yyyy-MM-dd HH:mm:ss")
	TransactionStatus    string `json:"transaction_status"`     // Status utama: "capture", "settlement", "pending", "deny", "cancel", "expire", "failure"
	TransactionID        string `json:"transaction_id"`         // ID unik dari Midtrans
	StatusMessage        string `json:"status_message"`         // Pesan status
	StatusCode           string `json:"status_code"`            // Kode status HTTP
	SignatureKey         string `json:"signature_key"`          // Hash untuk validasi
	PaymentType          string `json:"payment_type"`           // Metode pembayaran (e.g., "gopay", "bank_transfer")
	OrderID              string `json:"order_id"`               // ID Order/Transaksi dari sisi KITA (Xetor)
	MerchantID           string `json:"merchant_id"`            // ID Merchant Midtrans
	MaskedCard           string `json:"masked_card,omitempty"`  // Nomor kartu (masked) jika pakai kartu
	GrossAmount          string `json:"gross_amount"`           // Jumlah total (sebagai string)
	FraudStatus          string `json:"fraud_status,omitempty"` // Status fraud ("accept", "challenge", "deny")
	Currency             string `json:"currency"`               // Mata uang (e.g., "IDR")
	ChannelResponseCode  string `json:"channel_response_code,omitempty"`
	ChannelResponseMessage string `json:"channel_response_message,omitempty"`
	ApprovalCode         string `json:"approval_code,omitempty"` // Kode approval bank (jika ada)

	// -- Fields Spesifik untuk Disbursement (Withdraw) --
	// Midtrans mungkin punya format notifikasi berbeda untuk disbursement,
	// perlu dicek di dokumentasi Disbursement API mereka.
	// Jika berbeda, kita perlu buat struct terpisah atau tambahkan field opsional.
	// Contoh field disbursement (PERLU DIVERIFIKASI DGN DOKS MIDTRANS):
	ReferenceID      string    `json:"reference_id,omitempty"`     // ID referensi disbursement dari sisi kita
	DisbursementID   string    `json:"disbursement_id,omitempty"`  // ID disbursement dari Midtrans
	DisbursementStatus string    `json:"disbursement_status,omitempty"`// Status disbursement: "completed", "failed"
	FailureReason    string    `json:"failure_reason,omitempty"`   // Alasan jika gagal
	Timestamp        time.Time `json:"timestamp,omitempty"`        // Waktu proses disbursement
}

// Catatan Penting:
// Struktur payload webhook Midtrans bisa bervariasi tergantung jenis transaksi
// (pembayaran vs disbursement). Pastikan untuk memeriksa dokumentasi API Midtrans
// yang relevan (terutama untuk Disbursement/Payout) dan sesuaikan struct ini jika perlu.

// SnapTransactionRequest adalah struct untuk request create Snap transaction
type SnapTransactionRequest struct {
	OrderID     string  `json:"order_id"`     // Order ID dari sistem kita (misal: "TP-123")
	Amount      float64 `json:"amount"`       // Jumlah pembayaran
	CustomerName string `json:"customer_name"` // Nama customer
	CustomerEmail string `json:"customer_email"` // Email customer
	ItemDetails []midtrans.ItemDetails `json:"item_details"` // Detail item (opsional)
}

// SnapTransactionResponse adalah struct untuk response create Snap transaction
type SnapTransactionResponse struct {
	Token       string `json:"token"`        // Snap token untuk frontend
	RedirectURL string `json:"redirect_url"` // URL redirect (alternatif)
}