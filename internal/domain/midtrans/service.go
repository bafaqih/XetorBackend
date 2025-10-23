// internal/domain/midtrans/service.go
package midtrans

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings" // Untuk memisahkan order_id

	"xetor.id/backend/internal/config"
)

// Definisikan interface agar service bergantung pada abstraksi, bukan implementasi
type TransactionRepository interface {

	UpdateWithdrawStatus(orderID string, newStatus string, transactionID string) error
	// UpdateTopupStatus(orderID string, newStatus string, transactionID string) error
}

type MidtransService struct {
	repo TransactionRepository // Gunakan interface
}

func NewMidtransService(repo TransactionRepository) *MidtransService {
	return &MidtransService{repo: repo}
}

// ProcessNotification memvalidasi dan memproses notifikasi webhook
func (s *MidtransService) ProcessNotification(notification MidtransTransactionNotification) error {
	// 1. Validasi Signature Key
	err := s.validateSignature(notification)
	if err != nil {
		log.Printf("Midtrans webhook signature validation failed: %v", err)
		return err // Kembalikan error jika signature tidak valid
	}
	log.Println("Midtrans webhook signature validated successfully.")

	// 2. Proses Status Transaksi (Contoh untuk Withdraw & Topup)
	// Kita perlu cara untuk membedakan jenis transaksi dari order_id
	// Misalnya, order_id diawali "WD-" untuk withdraw, "TP-" untuk topup
	orderIDParts := strings.Split(notification.OrderID, "-")
	if len(orderIDParts) < 2 {
		log.Printf("Invalid Order ID format: %s", notification.OrderID)
		return errors.New("format Order ID tidak valid")
	}
	transactionTypePrefix := orderIDParts[0]
	// originalID, _ := strconv.Atoi(orderIDParts[1]) // ID asli dari tabel history

	log.Printf("Processing notification for Order ID: %s, Status: %s", notification.OrderID, notification.TransactionStatus)

	var updateErr error
	finalStatus := ""

	// Tentukan status akhir berdasarkan status Midtrans
	switch notification.TransactionStatus {
	case "settlement", "capture": // Capture untuk kartu kredit
		finalStatus = "Completed"
	case "deny", "cancel", "expire", "failure":
		finalStatus = "Failed"
	case "pending":
		finalStatus = "Pending" // Tetap pending jika notifikasi masih pending
	default:
		log.Printf("Unhandled Midtrans transaction status: %s", notification.TransactionStatus)
		// Mungkin tidak perlu update jika status tidak dikenali
		return nil // Atau kembalikan error jika perlu
	}

	// Update status di database berdasarkan prefix Order ID
	switch transactionTypePrefix {
	case "WD": // Withdraw
		// TODO: Pastikan struct notification sudah sesuai untuk disbursement
		// Mungkin statusnya ada di field `disbursement_status` bukan `transaction_status`
		// Perlu penyesuaian berdasarkan payload webhook disbursement yang asli
		log.Printf("Attempting to update withdraw status for Order ID: %s to %s", notification.OrderID, finalStatus)
		updateErr = s.repo.UpdateWithdrawStatus(notification.OrderID, finalStatus, notification.TransactionID)
	case "TP": // Top Up
		log.Printf("Attempting to update topup status for Order ID: %s to %s", notification.OrderID, finalStatus)
		// updateErr = s.repo.UpdateTopupStatus(notification.OrderID, finalStatus, notification.TransactionID) // Uncomment saat fungsi repo dibuat
	default:
		log.Printf("Unknown transaction type prefix in Order ID: %s", transactionTypePrefix)
		updateErr = errors.New("prefix Order ID tidak dikenali")
	}

	if updateErr != nil {
		log.Printf("Error updating transaction status in DB for Order ID %s: %v", notification.OrderID, updateErr)
		return updateErr // Kembalikan error jika update DB gagal
	}

	log.Printf("Successfully processed notification for Order ID: %s", notification.OrderID)
	return nil // Sukses
}

// validateSignature memverifikasi signature key dari Midtrans
func (s *MidtransService) validateSignature(notification MidtransTransactionNotification) error {
	serverKey := config.GetMidtransServerKey() // Ambil Server Key dari .env
	// String yang di-hash: order_id + status_code + gross_amount + ServerKey
	input := notification.OrderID + notification.StatusCode + notification.GrossAmount + serverKey
	
	// Hash menggunakan SHA512
	hasher := sha512.New()
	hasher.Write([]byte(input))
	calculatedSignature := hex.EncodeToString(hasher.Sum(nil))

	if calculatedSignature != notification.SignatureKey {
		return fmt.Errorf("invalid signature. Expected: %s, Got: %s", calculatedSignature, notification.SignatureKey)
	}
	return nil
}