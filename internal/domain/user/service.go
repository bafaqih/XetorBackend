// internal/domain/user/service.go
package user

import (
	"errors"
	"log"
	"strconv"
	"database/sql"
	"sort"

	"golang.org/x/crypto/bcrypt"
)

type Repository interface {
	// User-related methods
	Save(user *User) error
	FindByEmail(email string) (*User, error)
	FindByID(id int) (*User, error)
	GetCurrentPasswordHashByID(id int) (string, error)
	UpdatePassword(id int, newHashedPassword string) error
	DeleteUserByID(id int) error

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