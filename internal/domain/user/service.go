// internal/domain/user/service.go
package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/idtoken"
	"xetor.id/backend/internal/auth"
	"xetor.id/backend/internal/config"
	"xetor.id/backend/internal/notification"
	"xetor.id/backend/internal/temporary_token"
)

const (
	minWithdrawalAmount = 10000.0 // Minimal penarikan Rp 10.000
	withdrawalFee       = 2500.0  // Biaya admin Rp 2.500
	minTopupAmount      = 10000.0 // Minimal topup Rp 10.000
)

const conversionRateXpToRp = 5.0 // 1 Xp = 5 Rp

// MidtransServiceInterface adalah interface untuk Midtrans service (menghindari circular dependency)
type MidtransServiceInterface interface {
	CreateSnapTransaction(req interface{}) (interface{}, error)
	CreateSnapTransactionFromMap(reqMap map[string]interface{}) (map[string]interface{}, error)
}

type Repository interface {
	// User-related methods
	CreateUserFromGoogle(u *User) error
	Save(user *User) error
	FindByEmail(email string) (*User, error)
	FindByID(id int) (*User, error)
	GetCurrentPasswordHashByID(id int) (string, error)
	UpdatePassword(id int, newHashedPassword string) error
	DeleteUserByID(id int) error
	FindOrCreateWalletByUserID(userID int) (*UserWallet, error)
	FindOrCreateStatisticsByUserID(userID int) (*UserStatistic, error)
	UpdateUserProfile(id int, req *UpdateUserProfileRequest) error
	UpdateUserPhotoURL(id int, photoURL string) error

	// Address-related methods
	CreateAddress(addr *UserAddress) error
	GetAddressesByUserID(userID int) ([]UserAddress, error)
	GetAddressByID(id int, userID int) (*UserAddress, error)
	UpdateAddress(id int, userID int, req *UpdateUserAddressRequest) error
	DeleteAddress(id int, userID int) error

	// Transaction history methods
	GetDepositHistoryForUser(userID int) ([]TransactionHistoryItem, error)
	GetWithdrawHistoryForUser(userID int) ([]TransactionHistoryItem, error)
	GetTopupHistoryForUser(userID int) ([]TransactionHistoryItem, error)
	GetTransferHistoryForUser(userID int) ([]TransactionHistoryItem, error)

	// Withdraw methods
	GetCurrentBalanceByUserID(userID int) (float64, error)
	ExecuteWithdrawTransaction(userID int, amountToDeduct float64, fee float64, paymentMethodID int, accountNumber string) (string, error)
	GetPaymentMethodByID(id int) (*PaymentMethod, error)

	// Payment methods
	GetAllActivePaymentMethods() ([]PaymentMethod, error)

	// Promotion banners
	GetAllActivePromotionBanners() ([]PromotionBanner, error)

	// Topup methods
	CreateTopupTransaction(userID int, amount float64, paymentMethodID int) (string, error)
	GenerateTopupOrderID() (string, error) // Generate orderID tanpa create record
	UpdateTopupStatus(orderID string, newStatus string, transactionID string, amount float64, paymentMethodID int) error

	// Transfer methods
	FindUserIDByEmail(email string) (int, error)
	ExecuteTransferTransaction(senderUserID, recipientUserID, amount int, recipientEmail string) (string, error)

	// Conversion methods
	ExecuteConversionTransaction(userID int, xpoinChange int, balanceChange float64, conversionType string, amountXpInvolved int, amountRpInvolved float64, rate float64) (*UserWallet, error)
}

type Service struct {
	repo            Repository
	tokenStore      *temporary_token.TokenStore
	notifService    *notification.NotificationService
	midtransService MidtransServiceInterface
}

// NewService membuat instance baru dari Service
func NewService(repo Repository, tokenStore *temporary_token.TokenStore, notifService *notification.NotificationService, midtransService MidtransServiceInterface) *Service {
	return &Service{
		repo:            repo,
		tokenStore:      tokenStore,
		notifService:    notifService,
		midtransService: midtransService,
	}
}

// RegisterUser memproses data dari handler dan menyimpannya
func (s *Service) RegisterUser(req SignUpRequest) error {
	user := &User{
		Fullname: req.Fullname,
		Email:    req.Email,
		Phone:    stringToPtr(req.Phone),
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
		return nil, errors.New("email atau password salah") // User tidak ditemukan
	}

	// Bandingkan password yang diinput dengan hash di database
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	log.Println("Bcrypt comparison error:", err) // Log untuk debugging
	if err != nil {
		return nil, errors.New("email atau password salah") // Password tidak cocok
	}

	return user, nil // Kembalikan data user jika berhasil
}

// GetProfile mengambil data user berdasarkan ID
func (s *Service) GetProfile(userIDStr string) (*User, error) {
	userID, err := strconv.Atoi(userIDStr) // Konversi ID dari string (dari JWT) ke int
	if err != nil {
		log.Printf("Error converting userID string to int: %v", err)
		return nil, errors.New("ID pengguna tidak valid")
	}

	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, err // Error teknis dari repo
	}
	if user == nil {
		return nil, errors.New("pengguna tidak ditemukan")
	}
	return user, nil
}

// ChangePassword memvalidasi dan mengubah password user
func (s *Service) ChangePassword(userIDStr string, req ChangePasswordRequest) error {
	// 1. Validasi input dasar
	if req.NewPassword != req.ConfirmNewPassword {
		return errors.New("konfirmasi password baru tidak cocok")
	}
	// Tambahkan validasi panjang minimal di handler/binding, tapi bisa juga di sini
	if len(req.NewPassword) < 6 {
		return errors.New("password baru minimal 6 karakter")
	}

	// 2. Konversi userID
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Printf("Error converting userID string to int: %v", err)
		return errors.New("ID pengguna tidak valid")
	}

	// 3. Ambil hash password saat ini dari DB
	currentPasswordHash, err := s.repo.GetCurrentPasswordHashByID(userID)
	if err != nil {
		return err // Error teknis repo
	}
	if currentPasswordHash == "" {
		return errors.New("pengguna tidak ditemukan")
	}

	// 4. Bandingkan password lama yang diinput dengan hash di DB
	err = bcrypt.CompareHashAndPassword([]byte(currentPasswordHash), []byte(req.OldPassword))
	if err != nil {
		log.Println("Bcrypt compare old password error:", err) // Log error perbandingan
		return errors.New("password lama salah")
	}

	// 5. Hash password baru
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing new password: %v", err)
		return errors.New("gagal memproses password baru")
	}

	// 6. Update password di DB
	err = s.repo.UpdatePassword(userID, string(newHashedPassword))
	if err != nil {
		if err == sql.ErrNoRows { // Pastikan error ini diteruskan
			return errors.New("pengguna tidak ditemukan saat update")
		}
		return err // Error teknis repo
	}

	return nil // Sukses
}

// --- User Address Service Methods ---

func (s *Service) AddUserAddress(userIDStr string, req CreateUserAddressRequest) (*UserAddress, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, errors.New("ID pengguna tidak valid")
	}

	addr := &UserAddress{
		UserID:      userID,
		Fullname:    req.Fullname,
		Phone:       req.Phone,
		Address:     req.Address,
		CityRegency: req.CityRegency,
		Province:    req.Province,
		PostalCode:  sql.NullString{String: req.PostalCode, Valid: req.PostalCode != ""},
	}

	err = s.repo.CreateAddress(addr)
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func (s *Service) GetUserAddresses(userIDStr string) ([]UserAddress, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, errors.New("ID pengguna tidak valid")
	}
	return s.repo.GetAddressesByUserID(userID)
}

func (s *Service) GetUserAddressByID(id int, userIDStr string) (*UserAddress, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, errors.New("ID pengguna tidak valid")
	}
	return s.repo.GetAddressByID(id, userID)
}

func (s *Service) UpdateUserAddress(id int, userIDStr string, req UpdateUserAddressRequest) error {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return errors.New("ID pengguna tidak valid")
	}
	// Cek apakah ada data yang diupdate
	if req.Fullname == "" && req.Phone == "" && req.Address == "" && req.CityRegency == "" && req.Province == "" && req.PostalCode == "" {
		return errors.New("tidak ada data untuk diupdate")
	}
	return s.repo.UpdateAddress(id, userID, &req)
}

func (s *Service) DeleteUserAddress(id int, userIDStr string) error {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return errors.New("ID pengguna tidak valid")
	}
	return s.repo.DeleteAddress(id, userID)
}

// --- Transaction History Service Method ---

// GetTransactionHistory menggabungkan dan mengurutkan semua riwayat transaksi user
func (s *Service) GetTransactionHistory(userIDStr string) ([]TransactionHistoryItem, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, errors.New("ID pengguna tidak valid")
	}

	allTransactions := make([]TransactionHistoryItem, 0)

	// Ambil data dari masing-masing tabel
	depositHistory, err := s.repo.GetDepositHistoryForUser(userID)
	if err != nil {
		log.Printf("Error getting deposit history: %v", err) /* Lanjutkan saja */
	}
	allTransactions = append(allTransactions, depositHistory...)

	withdrawHistory, err := s.repo.GetWithdrawHistoryForUser(userID)
	if err != nil {
		log.Printf("Error getting withdraw history: %v", err) /* Lanjutkan saja */
	}
	allTransactions = append(allTransactions, withdrawHistory...)

	topupHistory, err := s.repo.GetTopupHistoryForUser(userID)
	if err != nil {
		log.Printf("Error getting topup history: %v", err) /* Lanjutkan saja */
	}
	allTransactions = append(allTransactions, topupHistory...)

	transferHistory, err := s.repo.GetTransferHistoryForUser(userID)
	if err != nil {
		log.Printf("Error getting transfer history: %v", err) /* Lanjutkan saja */
	}
	allTransactions = append(allTransactions, transferHistory...)

	// Urutkan semua transaksi berdasarkan waktu (terbaru dulu)
	sort.SliceStable(allTransactions, func(i, j int) bool {
		return allTransactions[i].Timestamp.After(allTransactions[j].Timestamp)
	})

	return allTransactions, nil
}

// DeleteAccount menghapus akun user
func (s *Service) DeleteAccount(userIDStr string) error {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Printf("Error converting userID string to int: %v", err)
		return errors.New("ID pengguna tidak valid")
	}

	err = s.repo.DeleteUserByID(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("pengguna tidak ditemukan")
		}
		return err // Error teknis repo
	}
	return nil // Sukses
}

// --- Payment Methods Service Method ---

// GetAllActivePaymentMethods mengambil semua payment methods yang aktif
func (s *Service) GetAllActivePaymentMethods() ([]PaymentMethod, error) {
	methods, err := s.repo.GetAllActivePaymentMethods()
	if err != nil {
		return nil, errors.New("gagal mengambil metode pembayaran")
	}
	return methods, nil
}

// GetAllActivePromotionBanners mengambil semua promotion banners yang aktif
func (s *Service) GetAllActivePromotionBanners() ([]PromotionBanner, error) {
	banners, err := s.repo.GetAllActivePromotionBanners()
	if err != nil {
		return nil, errors.New("gagal mengambil banner promosi")
	}
	return banners, nil
}

// --- User Wallet Service Method ---

// GetUserWallet mengambil data wallet user (membuat jika belum ada)
func (s *Service) GetUserWallet(userIDStr string) (*UserWallet, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Printf("Error converting userID string to int: %v", err)
		return nil, errors.New("ID pengguna tidak valid")
	}

	wallet, err := s.repo.FindOrCreateWalletByUserID(userID)
	if err != nil {
		// Repository sudah log error spesifik
		return nil, errors.New("gagal mengambil atau membuat wallet pengguna")
	}
	// Repository sudah memastikan wallet ada
	return wallet, nil
}

// --- User Statistics Service Method ---

// GetUserStatistics mengambil data statistik user (membuat jika belum ada)
func (s *Service) GetUserStatistics(userIDStr string) (*UserStatistic, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		log.Printf("Error converting userID string to int: %v", err)
		return nil, errors.New("ID pengguna tidak valid")
	}

	stats, err := s.repo.FindOrCreateStatisticsByUserID(userID)
	if err != nil {
		return nil, errors.New("gagal mengambil atau membuat statistik pengguna")
	}
	return stats, nil
}

// --- User Withdraw Service Method ---

// RequestWithdrawal memproses permintaan penarikan saldo
func (s *Service) RequestWithdrawal(userIDStr string, req WithdrawRequest) (string, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return "", errors.New("ID pengguna tidak valid")
	}

	// 1. Validasi Input Dasar
	if req.Amount < minWithdrawalAmount {
		return "", fmt.Errorf("minimal penarikan adalah Rp %.0f", minWithdrawalAmount)
	}

	// Validasi Payment Method ID
	paymentMethod, err := s.repo.GetPaymentMethodByID(req.PaymentMethodID)
	if err != nil {
		log.Printf("Error getting payment method ID %d: %v", req.PaymentMethodID, err)
		return "", errors.New("gagal memvalidasi metode pembayaran")
	}
	if paymentMethod == nil {
		return "", errors.New("metode pembayaran tidak valid")
	}
	if paymentMethod.Status != "Active" {
		return "", errors.New("metode pembayaran tidak aktif")
	}

	// TODO: Validasi Account Number (mungkin cek format dasar)

	// 2. Hitung Total dan Cek Saldo
	totalDeduction := req.Amount + withdrawalFee
	currentBalance, err := s.repo.GetCurrentBalanceByUserID(userID)
	if err != nil {
		return "", fmt.Errorf("gagal memeriksa saldo: %w", err)
	}

	if currentBalance < totalDeduction {
		return "", errors.New("saldo tidak mencukupi untuk penarikan dan biaya admin")
	}

	// 3. Eksekusi Transaksi Database (potong saldo + catat riwayat)
	orderID, err := s.repo.ExecuteWithdrawTransaction(userID, totalDeduction, withdrawalFee, req.PaymentMethodID, req.AccountNumber)
	if err != nil {
		// Error spesifik (saldo tidak cukup, dll) sudah ditangani di repo
		return "", fmt.Errorf("gagal memproses penarikan: %w", err)
	}

	// 4. (Nanti di sini) Panggil API Midtrans

	// --- KIRIM NOTIFIKASI ---
	go func() {
		notifTitle := "Penarikan Saldo Diproses"
		notifBody := fmt.Sprintf("Permintaan penarikan saldo sebesar Rp %.0f sedang diproses.", req.Amount)
		s.notifService.SendNotification(userID, notifTitle, notifBody, "WITHDRAW_PENDING")
	}()

	return orderID, nil // Kembalikan Order ID jika sukses
}

// --- User Top Up Service Method ---

// RequestTopup memproses permintaan top up saldo dengan Midtrans
func (s *Service) RequestTopup(userIDStr string, req TopupRequest) (*TopupResponse, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, errors.New("ID pengguna tidak valid")
	}

	// 1. Validasi Input Dasar
	if req.Amount <= 0 {
		return nil, errors.New("jumlah top up harus lebih besar dari 0")
	}

	if req.Amount < minTopupAmount {
		return nil, fmt.Errorf("minimal top up adalah Rp %.0f", minTopupAmount)
	}

	// 2. Pastikan wallet user ada (fungsi ini otomatis membuat jika belum ada)
	_, err = s.repo.FindOrCreateWalletByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa/membuat wallet: %w", err)
	}

	// 3. Ambil data user untuk customer details
	userData, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data pengguna: %w", err)
	}
	if userData == nil {
		return nil, errors.New("pengguna tidak ditemukan")
	}

	// 4. Generate order_id menggunakan sequence tanpa create record di DB
	// Record akan dibuat saat webhook pending pertama kali (setelah user pilih payment method)
	orderID, err := s.repo.GenerateTopupOrderID()
	if err != nil {
		return nil, fmt.Errorf("gagal membuat order ID: %w", err)
	}

	// 5. Buat Snap transaction di Midtrans menggunakan interface
	// Kita perlu membuat request sesuai dengan yang diharapkan MidtransService
	snapReqMap := map[string]interface{}{
		"order_id":       orderID,
		"amount":         req.Amount,
		"customer_name":  userData.Fullname,
		"customer_email": userData.Email,
	}

	// Panggil CreateSnapTransactionFromMap melalui interface
	snapRespMap, err := s.midtransService.CreateSnapTransactionFromMap(snapReqMap)
	if err != nil {
		log.Printf("Error creating Snap transaction for Order ID %s: %v", orderID, err)
		return nil, fmt.Errorf("gagal membuat transaksi Midtrans: %w", err)
	}

	token, _ := snapRespMap["token"].(string)
	redirectURL, _ := snapRespMap["redirect_url"].(string)

	response := &TopupResponse{
		OrderID:     orderID,
		SnapToken:   token,
		RedirectURL: redirectURL,
	}

	log.Printf("Topup request created successfully. Order ID: %s, Snap Token: %s", orderID, token)
	log.Printf("Note: DB record will be created when user selects payment method (webhook pending)")

	// Jangan kirim notifikasi dulu, tunggu webhook konfirmasi pembayaran berhasil
	// Notifikasi akan dikirim setelah webhook mengkonfirmasi status Completed

	return response, nil
}

// --- User Transfer Service Method ---

// TransferXpoin memproses transfer xpoin antar user
func (s *Service) TransferXpoin(senderUserIDStr string, req TransferRequest) (string, error) {
	senderUserID, err := strconv.Atoi(senderUserIDStr)
	if err != nil {
		return "", errors.New("ID pengirim tidak valid")
	}

	// 1. Validasi Input Dasar
	if req.Amount <= 0 {
		return "", errors.New("jumlah transfer harus lebih besar dari 0")
	}
	// TODO: Mungkin perlu validasi format email lagi di sini? (meskipun binding sudah)

	// 2. Cari ID Penerima berdasarkan Email
	recipientUserID, err := s.repo.FindUserIDByEmail(req.RecipientEmail)
	if err != nil {
		return "", errors.New("gagal mencari penerima")
	}
	if recipientUserID == 0 {
		return "", errors.New("email penerima tidak ditemukan")
	}

	// 3. Pastikan tidak transfer ke diri sendiri
	if senderUserID == recipientUserID {
		return "", errors.New("tidak bisa transfer ke diri sendiri")
	}

	// 4. Pastikan wallet pengirim dan penerima ada (repo akan handle create jika belum ada)
	_, err = s.repo.FindOrCreateWalletByUserID(senderUserID)
	if err != nil {
		return "", fmt.Errorf("gagal memeriksa wallet pengirim: %w", err)
	}
	_, err = s.repo.FindOrCreateWalletByUserID(recipientUserID)
	if err != nil {
		return "", fmt.Errorf("gagal memeriksa/membuat wallet penerima: %w", err)
	}

	// 5. Eksekusi Transaksi Database (kurangi poin pengirim, tambah poin penerima, catat riwayat)
	orderID, err := s.repo.ExecuteTransferTransaction(senderUserID, recipientUserID, req.Amount, req.RecipientEmail)
	if err != nil {
		// Error spesifik (poin tidak cukup, dll) sudah ditangani di repo
		return "", fmt.Errorf("gagal memproses transfer: %w", err)
	}

	// Notifikasi untuk Pengirim
	go func() {
		notifTitle := "Transfer Berhasil"
		notifBody := fmt.Sprintf("Kamu berhasil mentransfer %d Xpoin ke %s.", req.Amount, req.RecipientEmail)
		errNotif := s.notifService.SendNotification(senderUserID, notifTitle, notifBody, "TRANSFER_SENT_SUCCESS")
		if errNotif != nil {
			log.Printf("Gagal mengirim notifikasi transfer (sent) ke user %d: %v", senderUserID, errNotif)
		}
	}()

	// Notifikasi untuk Penerima
	go func(senderID int, recipientID int, amount int) {
		// Ambil data pengirim untuk notifikasi
		senderData, err := s.repo.FindByID(senderID) // Panggil repo untuk data pengirim
		if err != nil || senderData == nil {
			log.Printf("Gagal menemukan data pengirim (ID: %d) untuk notifikasi transfer: %v", senderID, err)
			return // Gagal mengirim notif jika pengirim tidak ditemukan
		}
		senderIdentifier := senderData.Fullname // Gunakan nama lengkap pengirim

		notifTitle := "Xpoin Diterima"
		notifBody := fmt.Sprintf("Kamu menerima %d Xpoin dari %s.", amount, senderIdentifier)
		errNotif := s.notifService.SendNotification(recipientID, notifTitle, notifBody, "TRANSFER_RECEIVED_SUCCESS")
		if errNotif != nil {
			log.Printf("Gagal mengirim notifikasi transfer (received) ke user %d: %v", recipientID, errNotif)
		}
	}(senderUserID, recipientUserID, req.Amount)

	return orderID, nil // Kembalikan Order ID jika sukses
}

// --- Conversion Service Methods ---

func (s *Service) ConvertXpToRp(userIDStr string, req ConversionRequest) (*UserWallet, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, errors.New("ID pengguna tidak valid")
	}

	amountXp := int(req.Amount) // Jumlah XP harus integer
	if float64(amountXp) != req.Amount || amountXp <= 0 {
		return nil, errors.New("jumlah Xpoin harus berupa angka bulat positif")
	}

	// Hitung jumlah Rp yang didapat
	amountRp := float64(amountXp) * conversionRateXpToRp

	// Pastikan wallet ada
	_, err = s.repo.FindOrCreateWalletByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa/membuat wallet: %w", err)
	}

	// Eksekusi transaksi: kurangi Xp (-amountXp), tambah Rp (+amountRp)
	updatedWallet, err := s.repo.ExecuteConversionTransaction(
		userID, -amountXp, amountRp,
		"xp_to_rp", amountXp, amountRp, conversionRateXpToRp,
	)
	if err != nil {
		return nil, err
	} // Error sudah ditangani repo (misal: xpoin tidak cukup)

	go func() {
		notifTitle := "Konversi Berhasil"
		notifBody := fmt.Sprintf("%d Xpoin berhasil dikonversi menjadi Rp %.0f.", amountXp, amountRp)
		s.notifService.SendNotification(userID, notifTitle, notifBody, "CONVERT_XP_RP_SUCCESS")
	}()

	return updatedWallet, nil
}

func (s *Service) ConvertRpToXp(userIDStr string, req ConversionRequest) (*UserWallet, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return nil, errors.New("ID pengguna tidak valid")
	}

	amountRp := req.Amount
	if amountRp <= 0 {
		return nil, errors.New("jumlah Rupiah harus positif")
	}

	// Hitung jumlah Xp yang didapat (bulatkan ke bawah)
	amountXp := int(math.Floor(amountRp / conversionRateXpToRp))
	if amountXp <= 0 {
		return nil, errors.New("jumlah Rupiah terlalu kecil untuk dikonversi menjadi Xpoin")
	}

	// Hitung ulang amountRp yang benar-benar digunakan berdasarkan Xp yang didapat
	// agar balance berkurang dengan jumlah yang pas
	actualAmountRpUsed := float64(amountXp) * conversionRateXpToRp

	// Pastikan wallet ada
	_, err = s.repo.FindOrCreateWalletByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("gagal memeriksa/membuat wallet: %w", err)
	}

	// Eksekusi transaksi: tambah Xp (+amountXp), kurangi Rp (-actualAmountRpUsed)
	updatedWallet, err := s.repo.ExecuteConversionTransaction(
		userID, amountXp, -actualAmountRpUsed,
		"rp_to_xp", amountXp, actualAmountRpUsed, conversionRateXpToRp,
	)
	if err != nil {
		return nil, err
	} // Error sudah ditangani repo (misal: saldo tidak cukup)

	go func() {
		notifTitle := "Konversi Berhasil"
		notifBody := fmt.Sprintf("Rp %.0f berhasil dikonversi menjadi %d Xpoin.", actualAmountRpUsed, amountXp)
		s.notifService.SendNotification(userID, notifTitle, notifBody, "CONVERT_RP_XP_SUCCESS")
	}()

	return updatedWallet, nil
}

// --- QR Token Service Method ---

// GenerateDepositQrToken membuat token sementara untuk deposit QR
func (s *Service) GenerateDepositQrToken(userIDStr string) (string, time.Time, error) {
	// Durasi token 5 menit
	validityDuration := 5 * time.Minute
	return s.tokenStore.CreateToken(userIDStr, validityDuration)
}

// --- User Profile Update Service ---

// UpdateProfile memproses update data profil user
func (s *Service) UpdateProfile(userIDStr string, req UpdateUserProfileRequest) error {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return errors.New("ID pengguna tidak valid")
	}

	// Cek apakah ada data yang diupdate
	if req.Fullname == "" && req.Email == "" && req.Phone == "" {
		return errors.New("tidak ada data untuk diupdate")
	}
	// TODO: Tambahkan validasi format email jika perlu

	err = s.repo.UpdateUserProfile(userID, &req)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("pengguna tidak ditemukan")
		}
		return err // Termasuk error email/phone duplikat dari repo
	}
	return nil
}

// UploadProfilePhoto menghandle upload file ke storage lokal (VPS) dan update DB user
func (s *Service) UploadProfilePhoto(userIDStr string, fileHeader *multipart.FileHeader) (string, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return "", errors.New("ID pengguna tidak valid")
	}

	file, err := fileHeader.Open()
	if err != nil {
		log.Printf("Error opening uploaded user photo file: %v", err)
		return "", errors.New("gagal membaca file foto")
	}
	defer file.Close()

	// Tentukan direktori dan nama file di storage lokal
	basePath := config.GetMediaBasePath()
	userDir := filepath.Join(basePath, "profile", "users")
	if err := os.MkdirAll(userDir, 0755); err != nil {
		log.Printf("Error creating user profile photo directory: %v", err)
		return "", errors.New("gagal menyiapkan penyimpanan foto")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == "" {
		ext = ".jpg"
	}
	filename := fmt.Sprintf("%d%s", userID, ext)
	fullPath := filepath.Join(userDir, filename)

	dst, err := os.Create(fullPath)
	if err != nil {
		log.Printf("Error creating destination file for user %d photo: %v", userID, err)
		return "", errors.New("gagal menyimpan file foto")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		log.Printf("Error copying uploaded photo to destination for user %d: %v", userID, err)
		return "", errors.New("gagal menyimpan file foto")
	}

	// Bangun URL publik berbasis CDN
	cdnBase := config.GetCDNBaseURL()
	photoURL := fmt.Sprintf("%s/profile/users/%s", cdnBase, filename)

	// Update URL di database
	err = s.repo.UpdateUserPhotoURL(userID, photoURL) // Panggil repo user
	if err != nil {
		log.Printf("DB update failed after saving user photo for user %d: %v", userID, err)
		return "", err
	}

	log.Printf("User %d profile photo updated to: %s", userID, photoURL)
	return photoURL, nil
}

// --- Google Auth Service Method ---

// verifyGoogleIDToken memvalidasi token ke server Google
func verifyGoogleIDToken(idToken string) (*idtoken.Payload, error) {
	googleClientID := config.GetGoogleClientID()
	ctx := context.Background()

	payload, err := idtoken.Validate(ctx, idToken, googleClientID)
	if err != nil {
		log.Printf("Error validating Google ID Token: %v", err)
		return nil, errors.New("token Google tidak valid atau kedaluwarsa")
	}
	return payload, nil
}

// AuthenticateWithGoogle memproses login/register via Google
func (s *Service) AuthenticateWithGoogle(idToken string) (string, *User, error) {
	// 1. Verifikasi token ke Google
	payload, err := verifyGoogleIDToken(idToken)
	if err != nil {
		return "", nil, err // Error: "token Google tidak valid"
	}

	// 2. Ambil data dari payload Google
	email := payload.Claims["email"].(string)
	fullname := payload.Claims["name"].(string)
	photoURL, _ := payload.Claims["picture"].(string) // Ambil foto, _ jika tidak ada

	if email == "" {
		return "", nil, errors.New("token Google tidak berisi email")
	}

	// 3. Cek apakah user sudah ada di DB kita (Sign In)
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		log.Printf("Error finding user by email %s: %v", email, err)
		return "", nil, errors.New("gagal memeriksa database user")
	}

	if user != nil {
		// --- KASUS SIGN IN ---
		// User ditemukan. Buat token JWT Xetor
		log.Printf("Google Sign-In: User %s (ID: %d) found.", user.Email, user.ID)
		token, err := auth.GenerateToken(user.ID, "user")
		if err != nil {
			return "", nil, errors.New("gagal membuat sesi login")
		}
		user.Password = "" // Hapus hash password
		return token, user, nil
	}

	// --- KASUS SIGN UP ---
	// User tidak ditemukan. Buat user baru.
	log.Printf("Google Sign-Up: User %s not found, creating new user.", email)

	newUser := &User{
		Fullname: fullname,
		Email:    email,
		Phone:    nil, // Phone tidak didapat dari Google
		Password: "",  // Repo akan handle ini (set ke "google_oauth_user")
		Photo:    stringToPtr(photoURL),
	}

	// Simpan user baru ke DB (Repo akan create user + wallet + stats)
	err = s.repo.CreateUserFromGoogle(newUser)
	if err != nil {
		return "", nil, err // Error dari repo (misal: "gagal menyimpan user")
	}

	// 5. Buat token JWT Xetor untuk user baru
	token, err := auth.GenerateToken(newUser.ID, "user")
	if err != nil {
		return "", nil, errors.New("gagal membuat sesi login untuk user baru")
	}

	newUser.Password = "" // Hapus placeholder password
	return token, newUser, nil
}

func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s // Mengambil alamat memori dari string
}
