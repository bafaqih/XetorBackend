// internal/repository/user_repo.go
package repository

import (
	"database/sql"
	"log"
	"os"
	"fmt"
	"strings"
	"strconv"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"xetor.id/backend/internal/domain/user"
)

type UserRepository struct {
	db *sql.DB
}

// NewUserRepository membuat instance baru dari UserRepository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Save menyimpan user baru ke database
func (r *UserRepository) Save(u *user.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Ambil URL default dari environment variable (lebih baik dari hardcode)
	defaultPhotoURL := os.Getenv("DEFAULT_PHOTO_URL")
	if defaultPhotoURL == "" {
		// Fallback jika tidak ada di .env (sebaiknya selalu ada)
		defaultPhotoURL = "https://res.cloudinary.com/db6vyj3n9/image/upload/v1761057514/onzkl78dvit7fyiqiphx.jpg"
		log.Println("WARNING: DEFAULT_PHOTO_URL not set in .env, using fallback.")
	}

	// --- PERUBAHAN DI SINI ---
	// Tambahkan kolom 'photo' dan $5 ke query
	query := "INSERT INTO users (fullname, email, phone, password, photo) VALUES ($1, $2, $3, $4, $5)"
	// Masukkan defaultPhotoURL sebagai argumen kelima
	_, err = r.db.Exec(query, u.Fullname, u.Email, u.Phone, string(hashedPassword), defaultPhotoURL)
	// -------------------------

	if err != nil {
		log.Printf("Error saving user to DB: %v", err)
		return err
	}

	log.Printf("User with email %s successfully saved to database.", u.Email)
	return nil
}

func (r *UserRepository) FindByEmail(email string) (*user.User, error) {
	// --- PERUBAHAN DI SINI ---
	// Tambahkan 'photo' ke query SELECT
	query := "SELECT id, fullname, email, phone, password, photo, created_at, updated_at FROM users WHERE email = $1"

	var u user.User
	// Tambahkan &u.Photo ke Scan
	err := r.db.QueryRow(query, email).Scan(&u.ID, &u.Fullname, &u.Email, &u.Phone, &u.Password, &u.Photo, &u.CreatedAt, &u.UpdatedAt)
	// -------------------------

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

// FindByID mencari user berdasarkan ID
func (r *UserRepository) FindByID(id int) (*user.User, error) {
	query := "SELECT id, fullname, email, phone, password, photo, created_at, updated_at FROM users WHERE id = $1"

	var u user.User
	err := r.db.QueryRow(query, id).Scan(&u.ID, &u.Fullname, &u.Email, &u.Phone, &u.Password, &u.Photo, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Tidak ditemukan
		}
		log.Printf("Error finding user by ID %d: %v", id, err)
		return nil, err
	}
	return &u, nil
}

// GetCurrentPasswordHashByID mengambil hash password saat ini berdasarkan ID user
func (r *UserRepository) GetCurrentPasswordHashByID(id int) (string, error) {
	query := "SELECT password FROM users WHERE id = $1"
	var currentPasswordHash string
	err := r.db.QueryRow(query, id).Scan(&currentPasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // User tidak ditemukan
		}
		log.Printf("Error getting current password hash for ID %d: %v", id, err)
		return "", err
	}
	return currentPasswordHash, nil
}

// UpdatePassword mengupdate hash password di database
func (r *UserRepository) UpdatePassword(id int, newHashedPassword string) error {
	query := "UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2"
	result, err := r.db.Exec(query, newHashedPassword, id)
	if err != nil {
		log.Printf("Error updating password for ID %d: %v", id, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows // User tidak ditemukan
	}
	log.Printf("Password updated successfully for user ID: %d", id)
	return nil
}

// --- User Address CRUD ---

func (r *UserRepository) CreateAddress(addr *user.UserAddress) error {
	query := `
		INSERT INTO user_addresses (user_id, fullname, phone, address, city_regency, province, postal_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	var postalCode sql.NullString
	if addr.PostalCode.Valid {
		postalCode = addr.PostalCode
	}

	err := r.db.QueryRow(query,
		addr.UserID, addr.Fullname, addr.Phone, addr.Address, addr.CityRegency, addr.Province, postalCode,
	).Scan(&addr.ID, &addr.CreatedAt, &addr.UpdatedAt)

	if err != nil {
		log.Printf("Error creating user address for user ID %d: %v", addr.UserID, err)
		return err
	}
	log.Printf("Address created with ID: %d for User ID: %d", addr.ID, addr.UserID)
	return nil
}

// GetAddressesByUserID mengambil semua alamat milik user tertentu
func (r *UserRepository) GetAddressesByUserID(userID int) ([]user.UserAddress, error) {
	query := `
		SELECT id, user_id, fullname, phone, address, city_regency, province, postal_code, created_at, updated_at
		FROM user_addresses
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		log.Printf("Error getting addresses for user ID %d: %v", userID, err)
		return nil, err
	}
	defer rows.Close()

	var addresses []user.UserAddress
	for rows.Next() {
		var addr user.UserAddress
		err := rows.Scan(
			&addr.ID, &addr.UserID, &addr.Fullname, &addr.Phone, &addr.Address,
			&addr.CityRegency, &addr.Province, &addr.PostalCode, &addr.CreatedAt, &addr.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning address row for user ID %d: %v", userID, err)
			return nil, err
		}
		addresses = append(addresses, addr)
	}
	return addresses, nil
}

// GetAddressByID mengambil satu alamat berdasarkan ID-nya, memastikan milik user yang benar
func (r *UserRepository) GetAddressByID(id int, userID int) (*user.UserAddress, error) {
	query := `
		SELECT id, user_id, fullname, phone, address, city_regency, province, postal_code, created_at, updated_at
		FROM user_addresses
		WHERE id = $1 AND user_id = $2` // Filter berdasarkan ID alamat dan ID user

	var addr user.UserAddress
	err := r.db.QueryRow(query, id, userID).Scan(
		&addr.ID, &addr.UserID, &addr.Fullname, &addr.Phone, &addr.Address,
		&addr.CityRegency, &addr.Province, &addr.PostalCode, &addr.CreatedAt, &addr.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Tidak ditemukan atau bukan milik user
		}
		log.Printf("Error getting address ID %d for user ID %d: %v", id, userID, err)
		return nil, err
	}
	return &addr, nil
}

func (r *UserRepository) UpdateAddress(id int, userID int, req *user.UpdateUserAddressRequest) error {
	fields := []string{}
	args := []interface{}{}
	argId := 1

	// Bangun query update dinamis
	if req.Fullname != "" { fields = append(fields, fmt.Sprintf("fullname = $%d", argId)); args = append(args, req.Fullname); argId++ }
	if req.Phone != "" { fields = append(fields, fmt.Sprintf("phone = $%d", argId)); args = append(args, req.Phone); argId++ }
	if req.Address != "" { fields = append(fields, fmt.Sprintf("address = $%d", argId)); args = append(args, req.Address); argId++ }
	if req.CityRegency != "" { fields = append(fields, fmt.Sprintf("city_regency = $%d", argId)); args = append(args, req.CityRegency); argId++ }
	if req.Province != "" { fields = append(fields, fmt.Sprintf("province = $%d", argId)); args = append(args, req.Province); argId++ }
	// Handle postal code (bisa diupdate jadi kosong)
	if req.PostalCode != "" {
		fields = append(fields, fmt.Sprintf("postal_code = $%d", argId)); args = append(args, sql.NullString{String: req.PostalCode, Valid: true}); argId++
	} else {
		// Jika ingin bisa mengosongkan postal code, tambahkan ini
		// fields = append(fields, fmt.Sprintf("postal_code = $%d", argId)); args = append(args, sql.NullString{Valid: false}); argId++
	}


	if len(fields) == 0 { return nil /* No fields to update */ }

	// Tambahkan kondisi WHERE dengan id dan userID
	args = append(args, id, userID)
	query := fmt.Sprintf("UPDATE user_addresses SET %s, updated_at = NOW() WHERE id = $%d AND user_id = $%d",
		strings.Join(fields, ", "), argId, argId+1)


	result, err := r.db.Exec(query, args...)
	if err != nil { log.Printf("Error updating address ID %d for user ID %d: %v", id, userID, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows /* Not found or not owned */ }
	log.Printf("Address updated for ID: %d", id)
	return nil
}

func (r *UserRepository) DeleteAddress(id int, userID int) error {
	query := `DELETE FROM user_addresses WHERE id = $1 AND user_id = $2` // Hanya bisa hapus milik sendiri
	result, err := r.db.Exec(query, id, userID)
	if err != nil { log.Printf("Error deleting address ID %d for user ID %d: %v", id, userID, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows /* Not found or not owned */ }
	log.Printf("Address deleted for ID: %d", id)
	return nil
}

// --- Transaction History Fetching ---

// GetDepositHistoryForUser mengambil riwayat deposit
func (r *UserRepository) GetDepositHistoryForUser(userID int) ([]user.TransactionHistoryItem, error) {
	query := `
		SELECT id, total_points, status, deposit_time
		FROM user_deposit_histories
		WHERE user_id = $1
		ORDER BY deposit_time DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil { return nil, err }
	defer rows.Close()

	var items []user.TransactionHistoryItem
	for rows.Next() {
		var item user.TransactionHistoryItem
		var id int
		var points int
		item.Type = "deposit"
		item.Description = "Deposit Sampah"

		if err := rows.Scan(&id, &points, &item.Status, &item.Timestamp); err != nil { return nil, err }
		// --- PERUBAHAN FORMAT ID ---
		item.ID = fmt.Sprintf("DP%05d", id) // Format: DP diikuti 5 digit angka (padding 0)
		item.Points = sql.NullInt32{Int32: int32(points), Valid: true}
		items = append(items, item)
	}
	return items, nil
}

// GetWithdrawHistoryForUser mengambil riwayat withdraw
func (r *UserRepository) GetWithdrawHistoryForUser(userID int) ([]user.TransactionHistoryItem, error) {
	query := `
		SELECT uwh.id, uwh.amount, uwh.status, uwh.withdraw_time, pm.name as payment_method_name
		FROM user_withdraw_histories uwh
		LEFT JOIN payment_methods pm ON uwh.payment_method_id = pm.id
		WHERE uwh.user_id = $1
		ORDER BY uwh.withdraw_time DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil { return nil, err }
	defer rows.Close()

	var items []user.TransactionHistoryItem
	for rows.Next() {
		var item user.TransactionHistoryItem
		var id int
		var amount float64
		var paymentName sql.NullString
		item.Type = "withdraw"

		if err := rows.Scan(&id, &amount, &item.Status, &item.Timestamp, &paymentName); err != nil { return nil, err }
		// --- PERUBAHAN FORMAT ID ---
		item.ID = fmt.Sprintf("WD%05d", id) // Format: WD diikuti 5 digit angka
		item.Amount = sql.NullString{String: fmt.Sprintf("%.2f", amount), Valid: true}
		item.Description = "Withdraw"
		if paymentName.Valid {
			item.Description += " ke " + paymentName.String
		}
		items = append(items, item)
	}
	return items, nil
}

// GetTopupHistoryForUser mengambil riwayat topup
func (r *UserRepository) GetTopupHistoryForUser(userID int) ([]user.TransactionHistoryItem, error) {
	query := `
		SELECT uth.id, uth.amount, uth.status, uth.topup_time, pm.name as payment_method_name
		FROM user_topup_histories uth
		LEFT JOIN payment_methods pm ON uth.payment_method_id = pm.id
		WHERE uth.user_id = $1
		ORDER BY uth.topup_time DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil { return nil, err }
	defer rows.Close()

	var items []user.TransactionHistoryItem
	for rows.Next() {
		var item user.TransactionHistoryItem
		var id int
		var amount float64
		var paymentName sql.NullString
		item.Type = "topup"

		if err := rows.Scan(&id, &amount, &item.Status, &item.Timestamp, &paymentName); err != nil { return nil, err }
		// --- PERUBAHAN FORMAT ID ---
		item.ID = fmt.Sprintf("TP%05d", id) // Format: TP diikuti 5 digit angka
		item.Amount = sql.NullString{String: fmt.Sprintf("%.2f", amount), Valid: true}
		item.Description = "Top Up"
		if paymentName.Valid {
			item.Description += " via " + paymentName.String
		}
		items = append(items, item)
	}
	return items, nil
}

// GetTransferHistoryForUser mengambil riwayat transfer
func (r *UserRepository) GetTransferHistoryForUser(userID int) ([]user.TransactionHistoryItem, error) {
	query := `
		SELECT id, amount, recipient_email, status, transfer_time
		FROM user_transfer_histories
		WHERE user_id = $1
		ORDER BY transfer_time DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil { return nil, err }
	defer rows.Close()

	var items []user.TransactionHistoryItem
	for rows.Next() {
		var item user.TransactionHistoryItem
		var id int
		var amount float64
		var recipient string
		item.Type = "transfer"

		if err := rows.Scan(&id, &amount, &recipient, &item.Status, &item.Timestamp); err != nil { return nil, err }
		// --- PERUBAHAN FORMAT ID ---
		item.ID = fmt.Sprintf("TF%05d", id) // Format: TF diikuti 5 digit angka
		item.Amount = sql.NullString{String: fmt.Sprintf("%.2f", amount), Valid: true}
		item.Description = "Transfer ke " + recipient
		items = append(items, item)
	}
	return items, nil
}

// DeleteUserByID menghapus user berdasarkan ID
func (r *UserRepository) DeleteUserByID(id int) error {
    query := "DELETE FROM users WHERE id = $1"
    result, err := r.db.Exec(query, id)
    if err != nil {
        log.Printf("Error deleting user ID %d: %v", id, err)
        return err
    }
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        return sql.ErrNoRows // User tidak ditemukan
    }
    log.Printf("User deleted successfully for ID: %d", id)
    // Data terkait akan otomatis terhapus oleh ON DELETE CASCADE
    return nil
}

// --- User Wallet ---

// FindOrCreateWalletByUserID mencari wallet user, atau membuatnya jika belum ada
func (r *UserRepository) FindOrCreateWalletByUserID(userID int) (*user.UserWallet, error) {
	querySelect := `
		SELECT id, user_id, balance, xpoin, created_at, updated_at
		FROM user_wallets
		WHERE user_id = $1`

	var wallet user.UserWallet
	var balance float64 // Baca dari DB sebagai float

	err := r.db.QueryRow(querySelect, userID).Scan(
		&wallet.ID, &wallet.UserID, &balance, &wallet.Xpoin, &wallet.CreatedAt, &wallet.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Wallet belum ada, buat baru
			log.Printf("Wallet not found for user ID %d, creating new one.", userID)
			queryInsert := `
				INSERT INTO user_wallets (user_id, balance, xpoin)
				VALUES ($1, 0.00, 0)
				RETURNING id, user_id, balance, xpoin, created_at, updated_at`

			errInsert := r.db.QueryRow(queryInsert, userID).Scan(
				&wallet.ID, &wallet.UserID, &balance, &wallet.Xpoin, &wallet.CreatedAt, &wallet.UpdatedAt,
			)
			if errInsert != nil {
				log.Printf("Error creating wallet for user ID %d: %v", userID, errInsert)
				return nil, errInsert
			}
			wallet.Balance = fmt.Sprintf("%.2f", balance) // Format ke string
			log.Printf("Wallet created successfully for user ID %d with ID %d", userID, wallet.ID)
			return &wallet, nil
		}
		// Error database lain saat SELECT
		log.Printf("Error finding wallet for user ID %d: %v", userID, err)
		return nil, err
	}

	// Wallet ditemukan
	wallet.Balance = fmt.Sprintf("%.2f", balance) // Format ke string
	return &wallet, nil
}

// --- User Statistics ---

// FindOrCreateStatisticsByUserID mencari statistik user, atau membuatnya jika belum ada
func (r *UserRepository) FindOrCreateStatisticsByUserID(userID int) (*user.UserStatistic, error) {
	querySelect := `
		SELECT id, user_id, waste, energy, co2, water, tree, created_at, updated_at
		FROM user_statistics
		WHERE user_id = $1`

	var stats user.UserStatistic
	var waste, energy, co2, water float64 // Baca DECIMAL sebagai float64

	err := r.db.QueryRow(querySelect, userID).Scan(
		&stats.ID, &stats.UserID, &waste, &energy, &co2, &water, &stats.Tree, &stats.CreatedAt, &stats.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Statistik belum ada, buat baru
			log.Printf("Statistics not found for user ID %d, creating new entry.", userID)
			queryInsert := `
				INSERT INTO user_statistics (user_id, waste, energy, co2, water, tree)
				VALUES ($1, 0.00, 0.00, 0.00, 0.00, 0)
				RETURNING id, user_id, waste, energy, co2, water, tree, created_at, updated_at`

			errInsert := r.db.QueryRow(queryInsert, userID).Scan(
				&stats.ID, &stats.UserID, &waste, &energy, &co2, &water, &stats.Tree, &stats.CreatedAt, &stats.UpdatedAt,
			)
			if errInsert != nil {
				log.Printf("Error creating statistics for user ID %d: %v", userID, errInsert)
				return nil, errInsert
			}
			// Format ke string setelah insert
			stats.Waste = fmt.Sprintf("%.2f", waste)
			stats.Energy = fmt.Sprintf("%.2f", energy)
			stats.CO2 = fmt.Sprintf("%.2f", co2)
			stats.Water = fmt.Sprintf("%.2f", water)
			log.Printf("Statistics created successfully for user ID %d with ID %d", userID, stats.ID)
			return &stats, nil
		}
		// Error database lain saat SELECT
		log.Printf("Error finding statistics for user ID %d: %v", userID, err)
		return nil, err
	}

	// Statistik ditemukan
	stats.Waste = fmt.Sprintf("%.2f", waste)
	stats.Energy = fmt.Sprintf("%.2f", energy)
	stats.CO2 = fmt.Sprintf("%.2f", co2)
	stats.Water = fmt.Sprintf("%.2f", water)
	return &stats, nil
}

// --- Transaction Status Update ---

// UpdateWithdrawStatus memperbarui status withdraw berdasarkan orderID (misal: "WD-123")
func (r *UserRepository) UpdateWithdrawStatus(orderID string, newStatus string, transactionID string) error {
    // Ekstrak ID asli dari orderID (misal: "WD-123" -> 123)
    parts := strings.Split(orderID, "-")
    if len(parts) != 2 || parts[0] != "WD" {
        log.Printf("Invalid withdraw order ID format for status update: %s", orderID)
        // Kita return nil agar Midtrans tidak retry, tapi log error
        // Atau return error jika ingin Midtrans coba lagi (hati-hati infinite loop)
        return nil // Atau errors.New("invalid order ID format")
    }
    withdrawID, err := strconv.Atoi(parts[1])
    if err != nil {
        log.Printf("Error converting withdraw ID from order ID %s: %v", orderID, err)
        return nil // Atau errors.New("invalid withdraw ID")
    }

    // TODO: Tambahkan kolom transaction_id dari Midtrans ke tabel user_withdraw_histories jika perlu

    query := `UPDATE user_withdraw_histories SET status = $1, updated_at = NOW() WHERE id = $2 AND status = 'Pending'` // Hanya update jika masih pending
    result, err := r.db.Exec(query, newStatus, withdrawID)
    if err != nil {
        log.Printf("Error updating withdraw status for ID %d: %v", withdrawID, err)
        return err
    }
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
        log.Printf("No pending withdraw found or status already updated for ID %d (Order ID: %s)", withdrawID, orderID)
        // Return nil karena mungkin notifikasi datang telat atau duplikat
        return nil
    }
    log.Printf("Withdraw status updated successfully for ID: %d (Order ID: %s) to %s", withdrawID, orderID, newStatus)
    return nil
}

// --- Withdraw Process Functions ---

// GetCurrentBalanceByUserID mengambil saldo saat ini
func (r *UserRepository) GetCurrentBalanceByUserID(userID int) (float64, error) {
	// Pastikan wallet ada (fungsi ini sudah otomatis membuat jika belum ada)
	wallet, err := r.FindOrCreateWalletByUserID(userID)
	if err != nil {
		return 0, fmt.Errorf("gagal mendapatkan wallet: %w", err)
	}

	// Konversi balance string ke float64
	balance, err := strconv.ParseFloat(wallet.Balance, 64)
	if err != nil {
		log.Printf("Error parsing balance string for user ID %d: %v", userID, err)
		return 0, errors.New("format saldo tidak valid")
	}
	return balance, nil
}

// ExecuteWithdrawTransaction menjalankan pengurangan saldo dan pencatatan riwayat dalam satu transaksi DB
func (r *UserRepository) ExecuteWithdrawTransaction(userID int, amountToDeduct float64, fee float64, paymentMethodID int, accountNumber string) (string, error) {
	// Mulai transaksi
	tx, err := r.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction for withdraw user ID %d: %v", userID, err)
		return "", errors.New("gagal memulai transaksi database")
	}
	// Pastikan rollback jika ada error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // Re-throw panic setelah rollback
		} else if err != nil {
			tx.Rollback() // Rollback jika ada error
		} else {
			err = tx.Commit() // Commit jika semua OK
			if err != nil {
				log.Printf("Error committing withdraw transaction for user ID %d: %v", userID, err)
			}
		}
	}()

	// 1. Kurangi saldo (dengan validasi saldo >= amountToDeduct)
	queryUpdateWallet := `
		UPDATE user_wallets
		SET balance = balance - $1, updated_at = NOW()
		WHERE user_id = $2 AND balance >= $1
		RETURNING balance` // Kembalikan sisa saldo untuk verifikasi (opsional)

	var remainingBalance float64 // Untuk menampung sisa saldo (opsional)
	err = tx.QueryRow(queryUpdateWallet, amountToDeduct, userID).Scan(&remainingBalance)
	if err != nil {
		if err == sql.ErrNoRows { // Ini terjadi jika saldo tidak cukup (WHERE balance >= $1 gagal)
			log.Printf("Insufficient balance for user ID %d during withdraw attempt.", userID)
			return "", errors.New("saldo tidak mencukupi")
		}
		log.Printf("Error updating wallet balance during withdraw for user ID %d: %v", userID, err)
		return "", errors.New("gagal mengupdate saldo")
	}

	// 2. Catat riwayat penarikan
	queryInsertHistory := `
		INSERT INTO user_withdraw_histories (user_id, payment_method_id, account_number, amount, fee, status, withdraw_time)
		VALUES ($1, $2, $3, $4, $5, 'Pending', NOW())
		RETURNING id` // Kembalikan ID withdraw history

	var withdrawID int
	amountRequested := amountToDeduct - fee // Jumlah yang diminta user (sebelum fee)
	err = tx.QueryRow(queryInsertHistory, userID, paymentMethodID, accountNumber, amountRequested, fee).Scan(&withdrawID)
	if err != nil {
		log.Printf("Error inserting withdraw history for user ID %d: %v", userID, err)
		return "", errors.New("gagal mencatat riwayat penarikan")
	}

	// Buat Order ID unik (misal: WD-<withdrawID>)
	orderID := fmt.Sprintf("WD-%d", withdrawID)
	log.Printf("Withdraw history created with ID %d (Order ID: %s) for user ID %d", withdrawID, orderID, userID)

	// Jika semua berhasil, transaksi akan di-commit oleh defer func
	return orderID, nil
}

// --- Top Up Process Functions ---

// ExecuteTopupTransaction menjalankan penambahan saldo dan pencatatan riwayat top up dalam satu transaksi DB
func (r *UserRepository) ExecuteTopupTransaction(userID int, amountToAdd float64, paymentMethodID int) (string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction for topup user ID %d: %v", userID, err)
		return "", errors.New("gagal memulai transaksi database")
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Printf("Error committing topup transaction for user ID %d: %v", userID, err)
			}
		}
	}()

	// 1. Tambahkan saldo
	queryUpdateWallet := `
		UPDATE user_wallets
		SET balance = balance + $1, updated_at = NOW()
		WHERE user_id = $2
		RETURNING balance`

	var currentBalance float64
	err = tx.QueryRow(queryUpdateWallet, amountToAdd, userID).Scan(&currentBalance)
	// Kita perlu memastikan wallet ada SEBELUM memanggil ini,
	// Fungsi FindOrCreateWalletByUserID bisa dipanggil di service sebelum eksekusi transaksi
	if err != nil {
		log.Printf("Error updating wallet balance during topup for user ID %d: %v", userID, err)
		// Kembalikan error jika user_id tidak ada di wallet (seharusnya tidak terjadi jika FindOrCreate dipanggil dulu)
		if err == sql.ErrNoRows {
			return "", errors.New("wallet pengguna tidak ditemukan")
		}
		return "", errors.New("gagal mengupdate saldo")
	}

	// 2. Catat riwayat top up
	queryInsertHistory := `
		INSERT INTO user_topup_histories (user_id, payment_method_id, amount, status, topup_time)
		VALUES ($1, $2, $3, 'Completed', NOW()) -- Status langsung Completed untuk simulasi
		RETURNING id`

	var topupID int
	err = tx.QueryRow(queryInsertHistory, userID, paymentMethodID, amountToAdd).Scan(&topupID)
	if err != nil {
		log.Printf("Error inserting topup history for user ID %d: %v", userID, err)
		return "", errors.New("gagal mencatat riwayat top up")
	}

	orderID := fmt.Sprintf("TP-%d", topupID)
	log.Printf("Topup history created with ID %d (Order ID: %s) for user ID %d", topupID, orderID, userID)

	return orderID, nil
}

// --- Transfer Xpoin Functions ---

// FindUserByEmail mencari user berdasarkan email (hanya butuh ID untuk transfer)
func (r *UserRepository) FindUserIDByEmail(email string) (int, error) {
	query := "SELECT id FROM users WHERE email = $1"
	var userID int
	err := r.db.QueryRow(query, email).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // User tidak ditemukan, bukan error teknis
		}
		log.Printf("Error finding user ID by email %s: %v", email, err)
		return 0, err // Error teknis
	}
	return userID, nil
}

// ExecuteTransferTransaction memproses pengurangan poin pengirim, penambahan poin penerima,
// dan pencatatan riwayat dalam satu transaksi DB.
func (r *UserRepository) ExecuteTransferTransaction(senderUserID, recipientUserID, amount int, recipientEmail string) (string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction for transfer from user ID %d: %v", senderUserID, err)
		return "", errors.New("gagal memulai transaksi database")
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Printf("Error committing transfer transaction from user ID %d: %v", senderUserID, err)
			}
		}
	}()

	// 1. Kurangi Xpoin pengirim (validasi xpoin >= amount)
	queryUpdateSender := `
		UPDATE user_wallets
		SET xpoin = xpoin - $1, updated_at = NOW()
		WHERE user_id = $2 AND xpoin >= $1
		RETURNING xpoin` // Kembalikan sisa poin (opsional)

	var senderRemainingXpoin int
	err = tx.QueryRow(queryUpdateSender, amount, senderUserID).Scan(&senderRemainingXpoin)
	if err != nil {
		if err == sql.ErrNoRows { // Terjadi jika poin tidak cukup
			log.Printf("Insufficient xpoin for user ID %d during transfer attempt.", senderUserID)
			return "", errors.New("xpoin tidak mencukupi")
		}
		log.Printf("Error updating sender wallet during transfer for user ID %d: %v", senderUserID, err)
		return "", errors.New("gagal mengupdate xpoin pengirim")
	}

	// 2. Tambah Xpoin penerima
	// Pastikan wallet penerima ada (FindOrCreateWalletByUserID dipanggil di service)
	queryUpdateRecipient := `
		UPDATE user_wallets
		SET xpoin = xpoin + $1, updated_at = NOW()
		WHERE user_id = $2`

	resultRecipient, err := tx.Exec(queryUpdateRecipient, amount, recipientUserID)
	if err != nil {
		log.Printf("Error updating recipient wallet during transfer for user ID %d: %v", recipientUserID, err)
		return "", errors.New("gagal mengupdate xpoin penerima")
	}
	rowsAffected, _ := resultRecipient.RowsAffected()
	if rowsAffected == 0 {
		// Ini seharusnya tidak terjadi jika FindOrCreateWalletByUserID dipanggil dulu
		log.Printf("Recipient wallet not found during transfer update for user ID %d", recipientUserID)
		return "", errors.New("wallet penerima tidak ditemukan")
	}


	// 3. Catat riwayat transfer
	queryInsertHistory := `
		INSERT INTO user_transfer_histories (user_id, amount, recipient_email, status, transfer_time)
		VALUES ($1, $2, $3, 'Completed', NOW())
		RETURNING id`

	var transferID int
	// Catatan: amount di history mungkin lebih baik float64/DECIMAL jika merepresentasikan Rupiah,
	// tapi karena ini transfer Xpoin (integer), kita simpan amount sbg integer saja di history?
	// Untuk konsistensi, kita simpan sbg DECIMAL(12,2) di DB tapi valuenya integer
	err = tx.QueryRow(queryInsertHistory, senderUserID, float64(amount), recipientEmail).Scan(&transferID)
	if err != nil {
		log.Printf("Error inserting transfer history for user ID %d: %v", senderUserID, err)
		return "", errors.New("gagal mencatat riwayat transfer")
	}

	orderID := fmt.Sprintf("TF-%d", transferID)
	log.Printf("Transfer history created with ID %d (Order ID: %s) for user ID %d to %s", transferID, orderID, senderUserID, recipientEmail)

	return orderID, nil
}

// --- Conversion Functions ---

// ExecuteConversionTransaction memproses perubahan balance dan xpoin dalam satu transaksi
func (r *UserRepository) ExecuteConversionTransaction(
	userID int,
	xpoinChange int, // Bisa positif (tambah) atau negatif (kurang)
	balanceChange float64, // Bisa positif (tambah) atau negatif (kurang)
	conversionType string,
	amountXpInvolved int,
	amountRpInvolved float64,
	rate float64,
) (*user.UserWallet, error) { // Kembalikan wallet terbaru

	tx, err := r.db.Begin()
	if err != nil { return nil, errors.New("gagal memulai transaksi database") }
	defer func() {
		if p := recover(); p != nil { tx.Rollback(); panic(p) } else
		if err != nil { tx.Rollback() } else
		{ err = tx.Commit() }
	}()

	// 1. Update Wallet (dengan validasi saldo/poin tidak minus)
	// Query ini mengupdate KEDUANYA (xpoin dan balance) dan memvalidasi
	queryUpdateWallet := `
		UPDATE user_wallets
		SET xpoin = xpoin + $1, balance = balance + $2, updated_at = NOW()
		WHERE user_id = $3
		  AND xpoin + $1 >= 0  -- Pastikan xpoin tidak jadi minus
		  AND balance + $2 >= 0 -- Pastikan balance tidak jadi minus
		RETURNING id, user_id, balance, xpoin, created_at, updated_at` // Kembalikan data wallet terbaru

	var updatedWallet user.UserWallet
	var currentBalance float64 // Baca balance dari DB

	err = tx.QueryRow(queryUpdateWallet, xpoinChange, balanceChange, userID).Scan(
		&updatedWallet.ID, &updatedWallet.UserID, &currentBalance, &updatedWallet.Xpoin,
		&updatedWallet.CreatedAt, &updatedWallet.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows { // Terjadi jika saldo/poin tidak cukup
			if xpoinChange < 0 {
				return nil, errors.New("xpoin tidak mencukupi")
			}
			if balanceChange < 0 {
				return nil, errors.New("saldo tidak mencukupi")
			}
			// Jika user belum punya wallet sama sekali (seharusnya tidak terjadi jika FindOrCreate dipanggil dulu)
			return nil, errors.New("wallet pengguna tidak ditemukan atau saldo/xpoin tidak mencukupi")
		}
		log.Printf("Error updating wallet during conversion for user ID %d: %v", userID, err)
		return nil, errors.New("gagal mengupdate wallet")
	}
	updatedWallet.Balance = fmt.Sprintf("%.2f", currentBalance) // Format balance

	// 2. Catat Riwayat Konversi (jika tabelnya ada)
	queryInsertHistory := `
		INSERT INTO user_conversion_histories
			(user_id, type, amount_xp, amount_rp, rate, conversion_time)
		VALUES ($1, $2, $3, $4, $5, NOW())`

	_, err = tx.Exec(queryInsertHistory, userID, conversionType, amountXpInvolved, amountRpInvolved, rate)
	if err != nil {
		// Jangan gagalkan transaksi utama jika log gagal, cukup catat error
		log.Printf("Error inserting conversion history for user ID %d: %v", userID, err)
		// return nil, errors.New("gagal mencatat riwayat konversi") // Jangan return error di sini
	}

	log.Printf("Conversion successful for user ID %d: %s (AmountXp: %d, AmountRp: %.2f)",
		userID, conversionType, amountXpInvolved, amountRpInvolved)

	return &updatedWallet, err // Return wallet terbaru dan error commit (jika ada)
}

// UpdateUserProfile mengupdate nama, email, dan/atau telepon user
func (r *UserRepository) UpdateUserProfile(id int, req *user.UpdateUserProfileRequest) error {
	fields := []string{}
	args := []interface{}{}
	argId := 1

	// Hanya tambahkan field ke query jika nilainya TIDAK kosong di request
	if req.Fullname != "" {
		fields = append(fields, fmt.Sprintf("fullname = $%d", argId))
		args = append(args, req.Fullname)
		argId++
	}
	if req.Email != "" {
		fields = append(fields, fmt.Sprintf("email = $%d", argId))
		args = append(args, req.Email)
		argId++
	}
	
	if req.Phone != "" { // Hanya update jika phone di request tidak kosong
		fields = append(fields, fmt.Sprintf("phone = $%d", argId))
		args = append(args, sql.NullString{String: req.Phone, Valid: true}) // Simpan sebagai string valid
		argId++
	}

	// Jika tidak ada field yang valid untuk diupdate
	if len(fields) == 0 {
		log.Println("UpdateUserProfile: No valid fields to update.")
		return nil // Tidak ada yang diupdate, bukan error
	}

	args = append(args, id) // Tambahkan ID user untuk klausa WHERE

	// query UPDATE
	query := fmt.Sprintf("UPDATE users SET %s, updated_at = NOW() WHERE id = $%d",
		strings.Join(fields, ", "), argId)

	log.Printf("Executing query: %s with args: %v", query, args)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		 log.Printf("Error updating user profile ID %d: %v", id, err)
		 // Cek error duplikat email
		 if strings.Contains(err.Error(), "users_email_key") {
			 return errors.New("email sudah digunakan")
		 }
		
		 return errors.New("gagal mengupdate profil")
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("UpdateUserProfile: User with ID %d not found.", id)
		return sql.ErrNoRows // User tidak ditemukan
	}

	log.Printf("User profile updated for ID: %d", id)
	return nil
}

// UpdateUserPhotoURL mengupdate URL foto user di database
func (r *UserRepository) UpdateUserPhotoURL(id int, photoURL string) error {
	query := "UPDATE users SET photo = $1, updated_at = NOW() WHERE id = $2"
	result, err := r.db.Exec(query, photoURL, id)
	if err != nil {
		log.Printf("Error updating user photo URL for ID %d: %v", id, err)
		return errors.New("gagal mengupdate URL foto")
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows // User tidak ditemukan
	}
	log.Printf("User photo URL updated for ID: %d", id)
	return nil
}

// --- User Deposit Related Functions ---

// AddDepositHistory: Pastikan fungsi ini ada
func (r *UserRepository) AddDepositHistory(userID, partnerID, totalPoints int, depositTime time.Time) (int, error) { // <-- Ubah return type
	query := `
		INSERT INTO user_deposit_histories (user_id, partner_id, total_points, status, deposit_time)
		VALUES ($1, $2, $3, 'Completed', $4)
		RETURNING id` // <-- Tambahkan RETURNING id

	var userDepositHistoryID int // <-- Variabel untuk menampung ID
	err := r.db.QueryRow(query, userID, partnerID, totalPoints, depositTime).Scan(&userDepositHistoryID) // <-- Scan ID-nya
	if err != nil {
		log.Printf("Error inserting user deposit history for user ID %d: %v", userID, err)
		return 0, errors.New("gagal mencatat riwayat deposit pengguna")
	}
	return userDepositHistoryID, nil // <-- Kembalikan ID
}

// UpdateUserWalletOnDeposit: Pastikan fungsi ini ada
func (r *UserRepository) UpdateUserWalletOnDeposit(userID, pointsToAdd int) error {
	query := `UPDATE user_wallets SET xpoin = xpoin + $1, updated_at = NOW() WHERE user_id = $2`
	result, err := r.db.Exec(query, pointsToAdd, userID);
    if err != nil { return errors.New("gagal update wallet user") }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return errors.New("wallet user tidak ditemukan") }
	return nil
}

// GetWasteDetailFactors: Pastikan fungsi ini ada dan mengembalikan map[int]user.ImpactFactors
func (r *UserRepository) GetWasteDetailFactors(wasteDetailIDs []int) (map[int]user.ImpactFactors, error) { // Return type dari model user
    if len(wasteDetailIDs) == 0 { return make(map[int]user.ImpactFactors), nil }
    placeholders := make([]string, len(wasteDetailIDs)); args := make([]interface{}, len(wasteDetailIDs))
    for i, id := range wasteDetailIDs { placeholders[i] = fmt.Sprintf("$%d", i+1); args[i] = id }
    query := fmt.Sprintf(`SELECT id, COALESCE(energy_factor, 0), COALESCE(co2_factor, 0), COALESCE(water_factor, 0), COALESCE(tree_factor, 0) FROM waste_details WHERE id IN (%s)`, strings.Join(placeholders, ","))

    rows, err := r.db.Query(query, args...); if err != nil { return nil, err }
    defer rows.Close()

    factorsMap := make(map[int]user.ImpactFactors) // Gunakan struct dari model user
    for rows.Next() {
        var detailID int
        var factors user.ImpactFactors // Gunakan struct dari model user
        if err := rows.Scan(&detailID, &factors.Energy, &factors.CO2, &factors.Water, &factors.Tree); err != nil { return nil, err }
        factorsMap[detailID] = factors
    }
    return factorsMap, rows.Err()
}

// UpdateUserStatisticsOnDeposit: Pastikan fungsi ini ada dan benar (tree jadi int)
func (r *UserRepository) UpdateUserStatisticsOnDeposit(userID int, totalWaste float64, energySaved, co2Saved, waterSaved float64, treesSaved int) error {
    query := `UPDATE user_statistics SET waste = waste + $1, energy = energy + $2, co2 = co2 + $3, water = water + $4, tree = tree + $5, updated_at = NOW() WHERE user_id = $6`
    result, err := r.db.Exec(query, totalWaste, energySaved, co2Saved, waterSaved, treesSaved, userID);
    if err != nil { return errors.New("gagal update statistik user") }
    rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return errors.New("statistik user tidak ditemukan") }
    return nil
}

// --- Google Auth Functions ---

// CreateGoogleUser membuat user baru dari data Google Auth
func (r *UserRepository) CreateUserFromGoogle(u *user.User) error {
	randomPassword := uuid.New().String()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
    if err != nil {
        log.Printf("Error generating random hash for Google user: %v", err)
        return errors.New("gagal mengamankan akun Google")
    }

	queryUsers := `
        INSERT INTO users (fullname, email, phone, password, photo, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
        RETURNING id, created_at, updated_at`

    err = r.db.QueryRow(queryUsers, u.Fullname, u.Email, u.Phone, string(hashedPassword), u.Photo).Scan(
        &u.ID, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "users_email_key") {
			return errors.New("email sudah terdaftar (repo)")
		}
		log.Printf("Error creating Google user in users table: %v", err)
		return errors.New("gagal menyimpan user google")
	}
	
	log.Printf("User created from Google Auth with ID: %d", u.ID)

    // 2. Buat Wallet dan Statistik (kita panggil fungsi yg sudah ada)
    _, err = r.FindOrCreateWalletByUserID(u.ID)
    if err != nil {
        log.Printf("Failed to create wallet for Google user ID %d: %v", u.ID, err)
        // Lanjutkan saja, jangan gagalkan registrasi
    }

    _, err = r.FindOrCreateStatisticsByUserID(u.ID)
     if err != nil {
        log.Printf("Failed to create statistics for Google user ID %d: %v", u.ID, err)
        // Lanjutkan saja
    }

	return nil
}