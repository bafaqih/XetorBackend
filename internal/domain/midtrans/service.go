// internal/domain/midtrans/service.go
package midtrans

import (
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings" // Untuk memisahkan order_id

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
	"xetor.id/backend/internal/config"
)

// Definisikan interface agar service bergantung pada abstraksi, bukan implementasi
type TransactionRepository interface {
	UpdateWithdrawStatus(orderID string, newStatus string, transactionID string) error
	UpdateTopupStatus(orderID string, newStatus string, transactionID string) error
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
		updateErr = s.repo.UpdateTopupStatus(notification.OrderID, finalStatus, notification.TransactionID)
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

// CreateSnapTransaction adalah method adapter untuk interface (menerima interface{}, return interface{})
func (s *MidtransService) CreateSnapTransaction(req interface{}) (interface{}, error) {
	// Convert interface{} ke map atau SnapTransactionRequest
	reqMap, ok := req.(map[string]interface{})
	if ok {
		// Gunakan method adapter
		return s.CreateSnapTransactionFromMap(reqMap)
	}
	
	// Jika bukan map, coba convert ke SnapTransactionRequest
	snapReq, ok := req.(SnapTransactionRequest)
	if !ok {
		return nil, errors.New("invalid request type for CreateSnapTransaction")
	}
	
	return s.createSnapTransactionInternal(snapReq)
}

// createSnapTransactionInternal adalah implementasi internal untuk create Snap transaction
func (s *MidtransService) createSnapTransactionInternal(req SnapTransactionRequest) (*SnapTransactionResponse, error) {
	// Setup Midtrans client
	serverKey := config.GetMidtransServerKey()
	
	// Tentukan environment (sandbox atau production)
	// Untuk sekarang kita pakai sandbox, bisa diubah via env variable nanti
	env := midtrans.Sandbox
	if os.Getenv("MIDTRANS_ENV") == "production" {
		env = midtrans.Production
	}
	
	snapClient := snap.Client{}
	snapClient.New(serverKey, env)

	// Konversi amount ke int64 (Midtrans menggunakan int64 untuk amount)
	amountInt64 := int64(req.Amount)

	// Buat Snap request
	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  req.OrderID,
			GrossAmt: amountInt64,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: req.CustomerName,
			Email: req.CustomerEmail,
		},
	}

	// Jika ada item details, tambahkan
	if len(req.ItemDetails) > 0 {
		snapReq.Items = &req.ItemDetails
	} else {
		// Default item: Top Up Saldo
		snapReq.Items = &[]midtrans.ItemDetails{
			{
				ID:    "TOPUP",
				Price: amountInt64,
				Qty:   1,
				Name:  "Top Up Saldo",
			},
		}
	}

	// Panggil API Midtrans untuk create transaction
	snapResp, err := snapClient.CreateTransaction(snapReq)
	if err != nil {
		log.Printf("Error creating Snap transaction for Order ID %s: %v", req.OrderID, err)
		return nil, fmt.Errorf("gagal membuat transaksi Midtrans: %w", err)
	}

	// Return response
	response := &SnapTransactionResponse{
		Token:       snapResp.Token,
		RedirectURL: snapResp.RedirectURL,
	}

	log.Printf("Snap transaction created successfully for Order ID: %s, Token: %s", req.OrderID, snapResp.Token)
	return response, nil
}

// CreateSnapTransactionFromMap adalah adapter method yang menerima map dan convert ke SnapTransactionRequest
// Method ini digunakan untuk menghindari circular dependency dengan user package
func (s *MidtransService) CreateSnapTransactionFromMap(reqMap map[string]interface{}) (map[string]interface{}, error) {
	// Convert map ke SnapTransactionRequest
	orderID, _ := reqMap["order_id"].(string)
	amount, _ := reqMap["amount"].(float64)
	customerName, _ := reqMap["customer_name"].(string)
	customerEmail, _ := reqMap["customer_email"].(string)

	req := SnapTransactionRequest{
		OrderID:       orderID,
		Amount:        amount,
		CustomerName:  customerName,
		CustomerEmail: customerEmail,
		ItemDetails:   nil,
	}

	resp, err := s.createSnapTransactionInternal(req)
	if err != nil {
		return nil, err
	}

	// Convert response ke map
	return map[string]interface{}{
		"token":        resp.Token,
		"redirect_url": resp.RedirectURL,
	}, nil
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