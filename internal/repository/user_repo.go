// internal/repository/user_repo.go
package repository

import (
	"database/sql"
	"log"
	"os"
	"fmt"
	"strings"

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
	query := "SELECT id, fullname, email, phone, password, photo FROM users WHERE email = $1"

	var u user.User
	// Tambahkan &u.Photo ke Scan
	err := r.db.QueryRow(query, email).Scan(&u.ID, &u.Fullname, &u.Email, &u.Phone, &u.Password, &u.Photo)
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
	query := "SELECT id, fullname, email, phone, password, photo FROM users WHERE id = $1"

	var u user.User
	err := r.db.QueryRow(query, id).Scan(&u.ID, &u.Fullname, &u.Email, &u.Phone, &u.Password, &u.Photo)
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