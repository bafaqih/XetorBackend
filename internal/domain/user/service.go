// internal/domain/user/service.go
package user

import (
	"errors"
	"log"
	"strconv"
	"database/sql"
	"sort"
	"fmt"
	"math"

	"golang.org/x/crypto/bcrypt"
)

const (
	minWithdrawalAmount = 10000.0 // Minimal penarikan Rp 10.000
	withdrawalFee       = 2500.0  // Biaya admin Rp 2.500
	minTopupAmount      = 10000.0 // Minimal topup Rp 10.000
)

const conversionRateXpToRp = 5.0 // 1 Xp = 5 Rp

type Repository interface {
	// User-related methods
	Save(user *User) error
	FindByEmail(email string) (*User, error)
	FindByID(id int) (*User, error)
	GetCurrentPasswordHashByID(id int) (string, error)
	UpdatePassword(id int, newHashedPassword string) error
	DeleteUserByID(id int) error
	FindOrCreateWalletByUserID(userID int) (*UserWallet, error)
	FindOrCreateStatisticsByUserID(userID int) (*UserStatistic, error)

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

	// Topup methods
	ExecuteTopupTransaction(userID int, amountToAdd float64, paymentMethodID int) (string, error)

	// Transfer methods
	FindUserByEmail(email string) (int, error)
	ExecuteTransferTransaction(senderUserID, recipientUserID, amount int, recipientEmail string) (string, error)

	// Conversion methods
	ExecuteConversionTransaction(userID int, xpoinChange int, balanceChange float64, conversionType string, amountXpInvolved int, amountRpInvolved float64, rate float64) (*UserWallet, error)
}

type Service struct {
	repo Repository
}

// NewService membuat instance baru dari Service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// RegisterUser memproses data dari handler dan menyimpannya
func (s *Service) RegisterUser(req SignUpRequest) error {
	user := &User{
		Fullname: req.Fullname,
		Email:    req.Email,
		Phone:    sql.NullString{String: req.Phone, Valid: req.Phone != ""},
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
		return nil, errors.New("kredensial tidak valid") // User tidak ditemukan
	}

	// Bandingkan password yang diinput dengan hash di database
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	log.Println("Bcrypt comparison error:", err) // Log untuk debugging
	if err != nil {
		return nil, errors.New("kredensial tidak valid") // Password tidak cocok
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
	userID, err := strconv.Atoi(userIDStr); if err != nil {
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
	if err != nil { return nil, err }
	return addr, nil
}

func (s *Service) GetUserAddresses(userIDStr string) ([]UserAddress, error) {
	userID, err := strconv.Atoi(userIDStr); if err != nil {
		return nil, errors.New("ID pengguna tidak valid")
	}
	return s.repo.GetAddressesByUserID(userID)
}

func (s *Service) GetUserAddressByID(id int, userIDStr string) (*UserAddress, error) {
	userID, err := strconv.Atoi(userIDStr); if err != nil {
		return nil, errors.New("ID pengguna tidak valid")
	}
	return s.repo.GetAddressByID(id, userID)
}

func (s *Service) UpdateUserAddress(id int, userIDStr string, req UpdateUserAddressRequest) error {
	userID, err := strconv.Atoi(userIDStr); if err != nil {
		return errors.New("ID pengguna tidak valid")
	}
	// Cek apakah ada data yang diupdate
	if req.Fullname == "" && req.Phone == "" && req.Address == "" && req.CityRegency == "" && req.Province == "" && req.PostalCode == "" {
		 return errors.New("tidak ada data untuk diupdate")
	}
	return s.repo.UpdateAddress(id, userID, &req)
}

func (s *Service) DeleteUserAddress(id int, userIDStr string) error {
	userID, err := strconv.Atoi(userIDStr); if err != nil {
		return errors.New("ID pengguna tidak valid")
	}
	return s.repo.DeleteAddress(id, userID)
}

// --- Transaction History Service Method ---

// GetTransactionHistory menggabungkan dan mengurutkan semua riwayat transaksi user
func (s *Service) GetTransactionHistory(userIDStr string) ([]TransactionHistoryItem, error) {
	userID, err := strconv.Atoi(userIDStr); if err != nil {
		return nil, errors.New("ID pengguna tidak valid")
	}

	allTransactions := make([]TransactionHistoryItem, 0)

	// Ambil data dari masing-masing tabel
	depositHistory, err := s.repo.GetDepositHistoryForUser(userID)
	if err != nil { log.Printf("Error getting deposit history: %v", err); /* Lanjutkan saja */ }
	allTransactions = append(allTransactions, depositHistory...)

	withdrawHistory, err := s.repo.GetWithdrawHistoryForUser(userID)
	if err != nil { log.Printf("Error getting withdraw history: %v", err); /* Lanjutkan saja */ }
	allTransactions = append(allTransactions, withdrawHistory...)

	topupHistory, err := s.repo.GetTopupHistoryForUser(userID)
	if err != nil { log.Printf("Error getting topup history: %v", err); /* Lanjutkan saja */ }
	allTransactions = append(allTransactions, topupHistory...)

	transferHistory, err := s.repo.GetTransferHistoryForUser(userID)
	if err != nil { log.Printf("Error getting transfer history: %v", err); /* Lanjutkan saja */ }
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
	userID, err := strconv.Atoi(userIDStr); if err != nil {
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
	if err != nil { return "", errors.New("ID pengguna tidak valid") }

	// 1. Validasi Input Dasar
	if req.Amount < minWithdrawalAmount {
		return "", fmt.Errorf("minimal penarikan adalah Rp %.0f", minWithdrawalAmount)
	}
	// TODO: Validasi Payment Method ID (cek apakah ID ada di tabel payment_methods)
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

	// 4. (Nanti di sini) Panggil API Midtrans Disbursement dengan 'orderID' sebagai referensi
	// log.Printf("TODO: Call Midtrans Disbursement API for Order ID: %s", orderID)

	return orderID, nil // Kembalikan Order ID jika sukses
}

// --- User Top Up Service Method ---

// RequestTopup memproses permintaan top up saldo (simulasi)
func (s *Service) RequestTopup(userIDStr string, req TopupRequest) (string, error) {
	userID, err := strconv.Atoi(userIDStr)
	if err != nil { return "", errors.New("ID pengguna tidak valid") }

	// 1. Validasi Input Dasar
	if req.Amount <= 0 {
		return "", errors.New("jumlah top up harus lebih besar dari 0")
	}

	if req.Amount < minTopupAmount {
		return "", fmt.Errorf("minimal top up adalah Rp %.0f", minTopupAmount)
	}

	// 2. Pastikan wallet user ada (fungsi ini otomatis membuat jika belum ada)
	_, err = s.repo.FindOrCreateWalletByUserID(userID)
	if err != nil {
		return "", fmt.Errorf("gagal memeriksa/membuat wallet: %w", err)
	}

	// 3. Eksekusi Transaksi Database (tambah saldo + catat riwayat)
	orderID, err := s.repo.ExecuteTopupTransaction(userID, req.Amount, req.PaymentMethodID)
	if err != nil {
		// Error spesifik sudah ditangani di repo
		return "", fmt.Errorf("gagal memproses top up: %w", err)
	}

	// 4. (Nanti di sini) Generate Midtrans Snap Token/URL dan kirim ke user
	// log.Printf("TODO: Generate Midtrans payment details for Order ID: %s", orderID)

	return orderID, nil // Kembalikan Order ID jika sukses
}

// --- User Transfer Service Method ---

// TransferXpoin memproses transfer xpoin antar user
func (s *Service) TransferXpoin(senderUserIDStr string, req TransferRequest) (string, error) {
	senderUserID, err := strconv.Atoi(senderUserIDStr)
	if err != nil { return "", errors.New("ID pengirim tidak valid") }

	// 1. Validasi Input Dasar
	if req.Amount <= 0 {
		return "", errors.New("jumlah transfer harus lebih besar dari 0")
	}
	// TODO: Mungkin perlu validasi format email lagi di sini? (meskipun binding sudah)

	// 2. Cari ID Penerima berdasarkan Email
	recipientUserID, err := s.repo.FindUserByEmail(req.RecipientEmail)
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
	if err != nil { return "", fmt.Errorf("gagal memeriksa wallet pengirim: %w", err) }
	_, err = s.repo.FindOrCreateWalletByUserID(recipientUserID)
	if err != nil { return "", fmt.Errorf("gagal memeriksa/membuat wallet penerima: %w", err) }

	// 5. Eksekusi Transaksi Database (kurangi poin pengirim, tambah poin penerima, catat riwayat)
	orderID, err := s.repo.ExecuteTransferTransaction(senderUserID, recipientUserID, req.Amount, req.RecipientEmail)
	if err != nil {
		// Error spesifik (poin tidak cukup, dll) sudah ditangani di repo
		return "", fmt.Errorf("gagal memproses transfer: %w", err)
	}

	return orderID, nil // Kembalikan Order ID jika sukses
}

// --- Conversion Service Methods ---

func (s *Service) ConvertXpToRp(userIDStr string, req ConversionRequest) (*UserWallet, error) {
	userID, err := strconv.Atoi(userIDStr); if err != nil { return nil, errors.New("ID pengguna tidak valid") }

	amountXp := int(req.Amount) // Jumlah XP harus integer
	if float64(amountXp) != req.Amount || amountXp <= 0 {
		return nil, errors.New("jumlah Xpoin harus berupa angka bulat positif")
	}

	// Hitung jumlah Rp yang didapat
	amountRp := float64(amountXp) * conversionRateXpToRp

	// Pastikan wallet ada
	_, err = s.repo.FindOrCreateWalletByUserID(userID)
	if err != nil { return nil, fmt.Errorf("gagal memeriksa/membuat wallet: %w", err) }

	// Eksekusi transaksi: kurangi Xp (-amountXp), tambah Rp (+amountRp)
	updatedWallet, err := s.repo.ExecuteConversionTransaction(
		userID, -amountXp, amountRp,
		"xp_to_rp", amountXp, amountRp, conversionRateXpToRp,
	)
	if err != nil { return nil, err } // Error sudah ditangani repo (misal: xpoin tidak cukup)

	return updatedWallet, nil
}

func (s *Service) ConvertRpToXp(userIDStr string, req ConversionRequest) (*UserWallet, error) {
	userID, err := strconv.Atoi(userIDStr); if err != nil { return nil, errors.New("ID pengguna tidak valid") }

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
	if err != nil { return nil, fmt.Errorf("gagal memeriksa/membuat wallet: %w", err) }

	// Eksekusi transaksi: tambah Xp (+amountXp), kurangi Rp (-actualAmountRpUsed)
	updatedWallet, err := s.repo.ExecuteConversionTransaction(
		userID, amountXp, -actualAmountRpUsed,
		"rp_to_xp", amountXp, actualAmountRpUsed, conversionRateXpToRp,
	)
	if err != nil { return nil, err } // Error sudah ditangani repo (misal: saldo tidak cukup)

	return updatedWallet, nil
}