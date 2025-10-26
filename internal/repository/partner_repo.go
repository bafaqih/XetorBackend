package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"xetor.id/backend/internal/domain/partner"
)

type PartnerRepository struct {
	db *sql.DB
}

func NewPartnerRepository(db *sql.DB) *PartnerRepository {
	return &PartnerRepository{db: db}
}

// SavePartner menyimpan data partner baru ke tabel partners dan xetor_partners
func (r *PartnerRepository) SavePartner(p *partner.Partner) error {
	tx, err := r.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction for partner save: %v", err)
		return errors.New("gagal memulai transaksi database")
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	// 1. Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("gagal hashing password")
	}

	// 2. Ambil URL foto default
	defaultPhotoURL := os.Getenv("DEFAULT_PHOTO_URL")
	var photo sql.NullString
	if defaultPhotoURL != "" {
		photo = sql.NullString{String: defaultPhotoURL, Valid: true}
	}

	// 3. Insert ke tabel partners
	queryPartners := `
		INSERT INTO partners (business_name, email, phone, password, photo)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`
	err = tx.QueryRow(queryPartners, p.BusinessName, p.Email, p.Phone, string(hashedPassword), photo).Scan(
		&p.ID, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		log.Printf("Error saving partner to partners table: %v", err)
		// Cek apakah error karena email/phone duplikat
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			if strings.Contains(err.Error(), "partners_email_key") {
				return errors.New("email sudah terdaftar")
			}
			if strings.Contains(err.Error(), "partners_phone_key") {
				return errors.New("nomor telepon sudah terdaftar")
			}
		}
		return errors.New("gagal menyimpan data partner")
	}
	p.Photo = photo // Update photo di struct

	// 4. Insert ke tabel xetor_partners dengan status Pending
	queryXetorPartners := `INSERT INTO xetor_partners (partner_id, status) VALUES ($1, 'Pending')`
	_, err = tx.Exec(queryXetorPartners, p.ID)
	if err != nil {
		log.Printf("Error saving partner to xetor_partners table: %v", err)
		return errors.New("gagal mendaftarkan partner ke sistem Xetor")
	}

	log.Printf("Partner %s (ID: %d) successfully saved with Pending status.", p.Email, p.ID)
	return err // Akan nil jika commit berhasil
}

// FindPartnerByEmail mencari partner berdasarkan email
func (r *PartnerRepository) FindPartnerByEmail(email string) (*partner.Partner, error) {
	query := `
		SELECT p.id, p.business_name, p.email, p.phone, p.password, p.photo, p.created_at, p.updated_at
		FROM partners p
		WHERE p.email = $1`
	var p partner.Partner
	err := r.db.QueryRow(query, email).Scan(
		&p.ID, &p.BusinessName, &p.Email, &p.Phone, &p.Password, &p.Photo, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Tidak ditemukan
		}
		log.Printf("Error finding partner by email %s: %v", email, err)
		return nil, err
	}
	return &p, nil
}

// FindXetorPartnerStatusByID mencari status partner di tabel xetor_partners
// Kita butuh ini saat login untuk memastikan partner sudah 'Approved'
func (r *PartnerRepository) FindXetorPartnerStatusByID(partnerID int) (string, error) {
	query := "SELECT status FROM xetor_partners WHERE partner_id = $1"
	var status string
	err := r.db.QueryRow(query, partnerID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			// Partner ada di tabel partners tapi tidak di xetor_partners (kasus aneh)
			log.Printf("Inconsistency: Partner ID %d found in partners but not in xetor_partners", partnerID)
			return "Not Registered", nil // Beri status khusus
		}
		log.Printf("Error finding xetor_partner status for partner ID %d: %v", partnerID, err)
		return "", err
	}
	return status, nil
}

// FindPartnerByID mencari partner berdasarkan ID
func (r *PartnerRepository) FindPartnerByID(id int) (*partner.Partner, error) {
	query := `
		SELECT p.id, p.business_name, p.email, p.phone, p.password, p.photo, p.created_at, p.updated_at
		FROM partners p
		WHERE p.id = $1`
	var p partner.Partner
	err := r.db.QueryRow(query, id).Scan(
		&p.ID, &p.BusinessName, &p.Email, &p.Phone, &p.Password, &p.Photo, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} // Tidak ditemukan
		log.Printf("Error finding partner by ID %d: %v", id, err)
		return nil, err
	}
	return &p, nil
}

// UpdatePartnerProfile mengupdate nama bisnis, email, dan/atau telepon partner
func (r *PartnerRepository) UpdatePartnerProfile(id int, req *partner.UpdatePartnerProfileRequest) error {
	fields := []string{}
	args := []interface{}{}
	argId := 1

	if req.BusinessName != "" {
		fields = append(fields, fmt.Sprintf("business_name = $%d", argId))
		args = append(args, req.BusinessName)
		argId++
	}
	if req.Email != "" {
		fields = append(fields, fmt.Sprintf("email = $%d", argId))
		args = append(args, req.Email)
		argId++
	}
	if req.Phone != "" {
		fields = append(fields, fmt.Sprintf("phone = $%d", argId))
		args = append(args, sql.NullString{String: req.Phone, Valid: true})
		argId++
	}

	if len(fields) == 0 {
		return nil
	}
	args = append(args, id)

	query := fmt.Sprintf("UPDATE partners SET %s, updated_at = NOW() WHERE id = $%d", strings.Join(fields, ", "), argId)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		log.Printf("Error updating partner profile ID %d: %v", id, err)
		// Cek error duplikat email atau phone
		if strings.Contains(err.Error(), "partners_email_key") {
			return errors.New("email sudah digunakan")
		}
		if strings.Contains(err.Error(), "partners_phone_key") {
			return errors.New("nomor telepon sudah digunakan")
		}
		return errors.New("gagal mengupdate profil")
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	log.Printf("Partner profile updated for ID: %d", id)
	return nil
}

// UpdatePartnerPhotoURL mengupdate URL foto di database
func (r *PartnerRepository) UpdatePartnerPhotoURL(id int, photoURL string) error {
	query := "UPDATE partners SET photo = $1, updated_at = NOW() WHERE id = $2"
	result, err := r.db.Exec(query, photoURL, id)
	if err != nil {
		log.Printf("Error updating partner photo URL for ID %d: %v", id, err)
		return errors.New("gagal mengupdate URL foto")
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows // Partner tidak ditemukan
	}
	log.Printf("Partner photo URL updated for ID: %d", id)
	return nil
}

// GetCurrentPasswordHashByID mengambil hash password partner saat ini berdasarkan ID
func (r *PartnerRepository) GetCurrentPasswordHashByID(id int) (string, error) {
	query := "SELECT password FROM partners WHERE id = $1"
	var currentPasswordHash string
	err := r.db.QueryRow(query, id).Scan(&currentPasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // Partner tidak ditemukan
		}
		log.Printf("Error getting current partner password hash for ID %d: %v", id, err)
		return "", err
	}
	return currentPasswordHash, nil
}

// UpdatePassword mengupdate hash password partner di database
func (r *PartnerRepository) UpdatePassword(id int, newHashedPassword string) error {
	query := "UPDATE partners SET password = $1, updated_at = NOW() WHERE id = $2"
	result, err := r.db.Exec(query, newHashedPassword, id)
	if err != nil {
		log.Printf("Error updating partner password for ID %d: %v", id, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows // Partner tidak ditemukan
	}
	log.Printf("Partner password updated successfully for ID: %d", id)
	return nil
}

// --- Partner Address Functions ---

// GetAddressByPartnerID mengambil alamat usaha partner
func (r *PartnerRepository) GetAddressByPartnerID(partnerID int) (*partner.PartnerAddress, error) {
	query := `
		SELECT id, partner_id, address, city_regency, province, postal_code, created_at, updated_at
		FROM partner_addresses
		WHERE partner_id = $1`

	var addr partner.PartnerAddress
	err := r.db.QueryRow(query, partnerID).Scan(
		&addr.ID, &addr.PartnerID, &addr.Address, &addr.CityRegency,
		&addr.Province, &addr.PostalCode, &addr.CreatedAt, &addr.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Belum punya alamat, bukan error
		}
		log.Printf("Error getting address for partner ID %d: %v", partnerID, err)
		return nil, err
	}
	return &addr, nil
}

// UpsertAddress membuat atau mengupdate alamat partner (INSERT ON CONFLICT)
func (r *PartnerRepository) UpsertAddress(addr *partner.PartnerAddress) error {
	query := `
		INSERT INTO partner_addresses (partner_id, address, city_regency, province, postal_code, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (partner_id) DO UPDATE SET -- Jika partner_id sudah ada, update saja
			address = EXCLUDED.address,
			city_regency = EXCLUDED.city_regency,
			province = EXCLUDED.province,
			postal_code = EXCLUDED.postal_code,
			updated_at = NOW()
		RETURNING id, created_at, updated_at` // Kembalikan ID dan timestamp

	var postalCode sql.NullString
	if addr.PostalCode.Valid {
		postalCode = addr.PostalCode
	}

	err := r.db.QueryRow(query,
		addr.PartnerID, addr.Address, addr.CityRegency, addr.Province, postalCode,
	).Scan(&addr.ID, &addr.CreatedAt, &addr.UpdatedAt) // Scan ID dan timestamp baru/update

	if err != nil {
		log.Printf("Error upserting address for partner ID %d: %v", addr.PartnerID, err)
		return errors.New("gagal menyimpan alamat usaha")
	}
	log.Printf("Address upserted with ID: %d for Partner ID: %d", addr.ID, addr.PartnerID)
	return nil
}

// --- Partner Schedule Functions ---

// GetScheduleByPartnerID mengambil satu baris jadwal operasional partner
func (r *PartnerRepository) GetScheduleByPartnerID(partnerID int) (*partner.PartnerSchedule, error) {
	query := `
		SELECT id, partner_id, days_open,
		       COALESCE(to_char(open_time, 'HH24:MI'), '') as open_time, -- Handle NULL time
		       COALESCE(to_char(close_time, 'HH24:MI'), '') as close_time,-- Handle NULL time
		       operating_status, created_at, updated_at
		FROM partner_schedules
		WHERE partner_id = $1`

	var ps partner.PartnerSchedule
	var daysOpenDB sql.NullString // Baca days_open sebagai NullString

	err := r.db.QueryRow(query, partnerID).Scan(
		&ps.ID, &ps.PartnerID, &daysOpenDB, &ps.OpenTime, &ps.CloseTime,
		&ps.OperatingStatus, &ps.CreatedAt, &ps.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Belum punya jadwal
		}
		log.Printf("Error getting schedule for partner ID %d: %v", partnerID, err)
		return nil, err
	}

	// Konversi string DB ke slice
	ps.DaysOpen = partner.DaysOpenFromString(daysOpenDB)

	return &ps, nil
}

// UpsertSchedule membuat atau mengupdate satu baris jadwal partner
func (r *PartnerRepository) UpsertSchedule(sched *partner.PartnerSchedule) error {
	query := `
		INSERT INTO partner_schedules (partner_id, days_open, open_time, close_time, operating_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (partner_id) DO UPDATE SET
			days_open = EXCLUDED.days_open,
			open_time = EXCLUDED.open_time,
			close_time = EXCLUDED.close_time,
			operating_status = EXCLUDED.operating_status,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	// Konversi slice ke string DB
	daysOpenDB := partner.DaysOpenToString(sched.DaysOpen)

	// Konversi string HH:MM ke tipe TIME atau NULL jika kosong/invalid (lebih aman di service)
	var openTime, closeTime interface{} // Gunakan interface{} untuk handle NULL
	if sched.OpenTime == "" {
		openTime = nil
	} else {
		openTime = sched.OpenTime
	}
	if sched.CloseTime == "" {
		closeTime = nil
	} else {
		closeTime = sched.CloseTime
	}

	err := r.db.QueryRow(query,
		sched.PartnerID, daysOpenDB, openTime, closeTime, sched.OperatingStatus,
	).Scan(&sched.ID, &sched.CreatedAt, &sched.UpdatedAt) // Scan ID dan timestamp baru/update

	if err != nil {
		log.Printf("Error upserting schedule for partner ID %d: %v", sched.PartnerID, err)
		return errors.New("gagal menyimpan jadwal usaha")
	}
	log.Printf("Schedule upserted with ID: %d for Partner ID: %d", sched.ID, sched.PartnerID)
	return nil
}

// --- Partner Waste Price Functions ---

// FindOrCreateWastePriceHeader mencari atau membuat baris header di partner_waste_prices
// Mengembalikan ID header.
func (r *PartnerRepository) FindOrCreateWastePriceHeader(partnerID int) (int, error) {
	var headerID int
	querySelect := "SELECT id FROM partner_waste_prices WHERE partner_id = $1"
	err := r.db.QueryRow(querySelect, partnerID).Scan(&headerID)

	if err != nil {
		if err == sql.ErrNoRows {
			// Header belum ada, buat baru
			queryInsert := `
				INSERT INTO partner_waste_prices (partner_id) VALUES ($1)
				RETURNING id`
			errInsert := r.db.QueryRow(queryInsert, partnerID).Scan(&headerID)
			if errInsert != nil {
				log.Printf("Error creating waste price header for partner ID %d: %v", partnerID, errInsert)
				return 0, errors.New("gagal membuat header harga sampah")
			}
			log.Printf("Waste price header created with ID: %d for Partner ID: %d", headerID, partnerID)
			return headerID, nil
		}
		// Error database lain saat SELECT
		log.Printf("Error finding waste price header for partner ID %d: %v", partnerID, err)
		return 0, err
	}
	// Header sudah ada
	return headerID, nil
}

// CreateWastePriceDetail menambahkan item harga sampah baru
func (r *PartnerRepository) CreateWastePriceDetail(detail *partner.PartnerWastePriceDetail) error {
	query := `
		INSERT INTO partner_waste_price_details
			(partner_waste_price_id, image, name, price, unit, xpoin)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	// Konversi price string ke float64 untuk disimpan sbg DECIMAL
	priceFloat, err := strconv.ParseFloat(detail.Price, 64)
	if err != nil {
		log.Printf("Error parsing price string '%s' to float: %v", detail.Price, err)
		return errors.New("format harga tidak valid")
	}

	err = r.db.QueryRow(query,
		detail.PartnerWastePriceID, detail.Image, detail.Name, priceFloat, detail.Unit, detail.Xpoin,
	).Scan(&detail.ID, &detail.CreatedAt, &detail.UpdatedAt)

	if err != nil {
		log.Printf("Error creating waste price detail: %v", err)
		return errors.New("gagal menyimpan detail harga sampah")
	}
	log.Printf("Waste price detail created with ID: %d", detail.ID)
	return nil
}

// GetWastePriceDetailsByPartnerID mengambil semua item harga sampah milik partner
func (r *PartnerRepository) GetWastePriceDetailsByPartnerID(partnerID int) ([]partner.PartnerWastePriceDetail, error) {
	headerID, err := r.FindOrCreateWastePriceHeader(partnerID) // Pastikan header ada
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, partner_waste_price_id, image, name, price, unit, xpoin, created_at, updated_at
		FROM partner_waste_price_details
		WHERE partner_waste_price_id = $1
		ORDER BY name ASC`

	rows, err := r.db.Query(query, headerID)
	if err != nil {
		log.Printf("Error getting waste price details for partner ID %d: %v", partnerID, err)
		return nil, err
	}
	defer rows.Close()

	var details []partner.PartnerWastePriceDetail
	for rows.Next() {
		var pd partner.PartnerWastePriceDetail
		var priceDB float64 // Baca DECIMAL sebagai float
		if err := rows.Scan(
			&pd.ID, &pd.PartnerWastePriceID, &pd.Image, &pd.Name, &priceDB,
			&pd.Unit, &pd.Xpoin, &pd.CreatedAt, &pd.UpdatedAt,
		); err != nil {
			log.Printf("Error scanning waste price detail row for partner ID %d: %v", partnerID, err)
			return nil, err
		}
		pd.Price = fmt.Sprintf("%.2f", priceDB) // Format ke string
		details = append(details, pd)
	}
	return details, nil
}

// GetWastePriceDetailByID mengambil satu item harga (memastikan milik partner)
func (r *PartnerRepository) GetWastePriceDetailByID(detailID int, partnerID int) (*partner.PartnerWastePriceDetail, error) {
	headerID, err := r.FindOrCreateWastePriceHeader(partnerID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, partner_waste_price_id, image, name, price, unit, xpoin, created_at, updated_at
		FROM partner_waste_price_details
		WHERE id = $1 AND partner_waste_price_id = $2`

	var pd partner.PartnerWastePriceDetail
	var priceDB float64
	err = r.db.QueryRow(query, detailID, headerID).Scan(
		&pd.ID, &pd.PartnerWastePriceID, &pd.Image, &pd.Name, &priceDB,
		&pd.Unit, &pd.Xpoin, &pd.CreatedAt, &pd.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		} // Tidak ditemukan / bukan milik partner
		log.Printf("Error getting waste price detail ID %d for partner ID %d: %v", detailID, partnerID, err)
		return nil, err
	}
	pd.Price = fmt.Sprintf("%.2f", priceDB)
	return &pd, nil
}

// UpdateWastePriceDetail mengupdate item harga sampah
func (r *PartnerRepository) UpdateWastePriceDetail(detailID int, partnerID int, detail *partner.PartnerWastePriceDetail) error {
	headerID, err := r.FindOrCreateWastePriceHeader(partnerID)
	if err != nil {
		return err
	}

	fields := []string{}
	args := []interface{}{}
	argId := 1

	// Bangun query update dinamis
	if detail.Image.Valid {
		fields = append(fields, fmt.Sprintf("image = $%d", argId))
		args = append(args, detail.Image)
		argId++
	}
	if detail.Name != "" {
		fields = append(fields, fmt.Sprintf("name = $%d", argId))
		args = append(args, detail.Name)
		argId++
	}
	if detail.Price != "" {
		priceFloat, err := strconv.ParseFloat(detail.Price, 64)
		if err != nil {
			return errors.New("format harga tidak valid")
		}
		fields = append(fields, fmt.Sprintf("price = $%d", argId))
		args = append(args, priceFloat)
		argId++
	}
	if detail.Unit != "" {
		fields = append(fields, fmt.Sprintf("unit = $%d", argId))
		args = append(args, detail.Unit)
		argId++
	}
	// Selalu update xpoin jika price diupdate atau xpoin dikirim (meskipun 0)
	fields = append(fields, fmt.Sprintf("xpoin = $%d", argId))
	args = append(args, detail.Xpoin)
	argId++

	if len(fields) <= 1 { // Jika hanya xpoin atau tidak ada field lain
		if detail.Name == "" && detail.Price == "" && detail.Unit == "" && !detail.Image.Valid {
			log.Println("UpdateWastePriceDetail: No valid fields to update other than xpoin.")
			// Jika hanya Xpoin yg diupdate, tetap jalankan query
			if len(fields) == 1 && strings.HasPrefix(fields[0], "xpoin") {
				// Biarkan query jalan
			} else {
				return nil // Tidak ada yg diupdate
			}
		}
	}

	args = append(args, detailID, headerID) // ID Detail dan ID Header untuk WHERE

	query := fmt.Sprintf("UPDATE partner_waste_price_details SET %s, updated_at = NOW() WHERE id = $%d AND partner_waste_price_id = $%d",
		strings.Join(fields, ", "), argId, argId+1)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		log.Printf("Error updating waste price detail ID %d: %v", detailID, err)
		return errors.New("gagal mengupdate detail harga sampah")
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows /* Not found or not owned */
	}
	log.Printf("Waste price detail updated for ID: %d", detailID)
	return nil
}

// DeleteWastePriceDetail menghapus item harga sampah
func (r *PartnerRepository) DeleteWastePriceDetail(detailID int, partnerID int) error {
	headerID, err := r.FindOrCreateWastePriceHeader(partnerID)
	if err != nil {
		return err
	}

	query := `DELETE FROM partner_waste_price_details WHERE id = $1 AND partner_waste_price_id = $2`
	result, err := r.db.Exec(query, detailID, headerID)
	if err != nil {
		log.Printf("Error deleting waste price detail ID %d: %v", detailID, err)
		return errors.New("gagal menghapus detail harga sampah")
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows /* Not found or not owned */
	}
	log.Printf("Waste price detail deleted for ID: %d", detailID)
	return nil
}

// --- Partner Financial Transaction History Fetching ---

// GetWithdrawHistoryForPartner mengambil riwayat withdraw partner
func (r *PartnerRepository) GetWithdrawHistoryForPartner(partnerID int) ([]partner.PartnerTransactionHistoryItem, error) {
	query := `
		SELECT pwh.id, pwh.amount, pwh.status, pwh.withdraw_time, pm.name as payment_method_name
		FROM partner_withdraw_histories pwh
		LEFT JOIN payment_methods pm ON pwh.payment_method_id = pm.id
		WHERE pwh.partner_id = $1
		ORDER BY pwh.withdraw_time DESC`
	rows, err := r.db.Query(query, partnerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []partner.PartnerTransactionHistoryItem
	for rows.Next() {
		var item partner.PartnerTransactionHistoryItem
		var id int
		var amount float64
		var paymentName sql.NullString
		item.Type = "withdraw"

		if err := rows.Scan(&id, &amount, &item.Status, &item.Timestamp, &paymentName); err != nil {
			return nil, err
		}
		item.ID = fmt.Sprintf("WD%05d", id) // Format ID
		item.Amount = sql.NullString{String: fmt.Sprintf("%.2f", amount), Valid: true}
		item.Description = "Withdraw"
		if paymentName.Valid {
			item.Description += " ke " + paymentName.String
		}
		items = append(items, item)
	}
	return items, nil
}

// GetTopupHistoryForPartner mengambil riwayat topup partner
func (r *PartnerRepository) GetTopupHistoryForPartner(partnerID int) ([]partner.PartnerTransactionHistoryItem, error) {
	query := `
		SELECT pth.id, pth.amount, pth.status, pth.topup_time, pm.name as payment_method_name
		FROM partner_topup_histories pth
		LEFT JOIN payment_methods pm ON pth.payment_method_id = pm.id
		WHERE pth.partner_id = $1
		ORDER BY pth.topup_time DESC`
	rows, err := r.db.Query(query, partnerID)
	if err != nil {
		log.Printf("Error querying topup history for partner ID %d: %v", partnerID, err)
		return nil, err // Kembalikan error jika query gagal
	}
	defer rows.Close()

	var items []partner.PartnerTransactionHistoryItem
	for rows.Next() {
		var item partner.PartnerTransactionHistoryItem
		var id int
		var amount float64
		var paymentName sql.NullString
		item.Type = "topup"

		if err := rows.Scan(&id, &amount, &item.Status, &item.Timestamp, &paymentName); err != nil {
			return nil, err
		}
		item.ID = fmt.Sprintf("TP%05d", id) // Format ID
		item.Amount = sql.NullString{String: fmt.Sprintf("%.2f", amount), Valid: true}
		item.Description = "Top Up"
		if paymentName.Valid {
			item.Description += " via " + paymentName.String
		}
		items = append(items, item)
	}
	return items, nil
}

// GetConversionHistoryForPartner mengambil riwayat konversi partner
func (r *PartnerRepository) GetConversionHistoryForPartner(partnerID int) ([]partner.PartnerTransactionHistoryItem, error) {
	query := `
		SELECT id, type, amount_xp, amount_rp, status, conversion_time -- Asumsikan ada kolom status? Jika tidak, set default
		FROM partner_conversion_histories
		WHERE partner_id = $1
		ORDER BY conversion_time DESC`
	rows, err := r.db.Query(query, partnerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []partner.PartnerTransactionHistoryItem
	for rows.Next() {
		var item partner.PartnerTransactionHistoryItem
		var id, amountXp int
		var amountRp float64
		var status string // Baca status
		item.Type = "convert"

		// Sesuaikan Scan dengan kolom tabel partner_conversion_histories
		if err := rows.Scan(&id, &item.Type, &amountXp, &amountRp, &status, &item.Timestamp); err != nil {
			return nil, err
		}

		item.ID = fmt.Sprintf("CV%05d", id) // Format ID
		item.Status = status                // Gunakan status dari DB
		item.Points = sql.NullInt32{Int32: int32(amountXp), Valid: true}
		item.Amount = sql.NullString{String: fmt.Sprintf("%.2f", amountRp), Valid: true}

		// Buat deskripsi berdasarkan tipe konversi
		if item.Type == "xp_to_rp" {
			item.Description = fmt.Sprintf("Konversi %d Xp ke Rp %.2f", amountXp, amountRp)
		} else if item.Type == "rp_to_xp" {
			item.Description = fmt.Sprintf("Konversi Rp %.2f ke %d Xp", amountRp, amountXp)
		} else {
			item.Description = "Konversi" // Fallback
		}
		item.Type = "convert" // Set tipe utama

		items = append(items, item)
	}
	return items, nil
}

// GetTransferHistoryForPartner mengambil riwayat transfer partner
func (r *PartnerRepository) GetTransferHistoryForPartner(partnerID int) ([]partner.PartnerTransactionHistoryItem, error) {
	query := `
		SELECT id, amount, recipient_email, status, transfer_time
		FROM partner_transfer_histories
		WHERE partner_id = $1
		ORDER BY transfer_time DESC`
	rows, err := r.db.Query(query, partnerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []partner.PartnerTransactionHistoryItem
	for rows.Next() {
		var item partner.PartnerTransactionHistoryItem
		var id int
		var amount float64
		var recipient string
		item.Type = "transfer"

		if err := rows.Scan(&id, &amount, &recipient, &item.Status, &item.Timestamp); err != nil {
			return nil, err
		}
		item.ID = fmt.Sprintf("TF%05d", id) // Format ID
		item.Amount = sql.NullString{String: fmt.Sprintf("%.2f", amount), Valid: true}
		item.Description = "Transfer ke " + recipient
		items = append(items, item)
	}
	return items, nil
}

// --- Partner Deposit History ---

// GetDepositHistoryByPartnerID mengambil riwayat deposit sampah partner beserta detailnya
func (r *PartnerRepository) GetDepositHistoryByPartnerID(partnerID int) ([]partner.DepositHistoryHeader, error) {
	// Query utama untuk mendapatkan header transaksi dan data user
	queryHeader := `
		SELECT
			pdh.id, pdh.partner_id, pdh.user_id, u.fullname as user_name, u.email as user_email,
			pdh.total_weight, pdh.total_xpoin, pdh.transaction_time, pdh.created_at, pdh.updated_at
		FROM partner_deposit_histories pdh
		JOIN users u ON pdh.user_id = u.id -- Join dengan tabel users
		WHERE pdh.partner_id = $1
		ORDER BY pdh.transaction_time DESC`

	rowsHeader, err := r.db.Query(queryHeader, partnerID)
	if err != nil {
		log.Printf("Error querying deposit history headers for partner ID %d: %v", partnerID, err)
		return nil, err
	}
	defer rowsHeader.Close()

	historiesMap := make(map[int]*partner.DepositHistoryHeader) // Map untuk mengelompokkan detail ke header
	historyOrder := []int{}                                     // Slice untuk menjaga urutan header

	// Iterasi hasil query header
	for rowsHeader.Next() {
		var header partner.DepositHistoryHeader
		var totalWeight sql.NullFloat64 // Baca DECIMAL sebagai NullFloat64

		err := rowsHeader.Scan(
			&header.ID, &header.PartnerID, &header.UserID, &header.UserName, &header.UserEmail,
			&totalWeight, &header.TotalXpoin, &header.TransactionTime, &header.CreatedAt, &header.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning deposit history header row for partner ID %d: %v", partnerID, err)
			return nil, err
		}

		if totalWeight.Valid {
			header.TotalWeight = sql.NullString{String: fmt.Sprintf("%.2f", totalWeight.Float64), Valid: true}
		}
		header.Details = []partner.DepositHistoryDetailItem{} // Inisialisasi slice detail

		historiesMap[header.ID] = &header
		historyOrder = append(historyOrder, header.ID) // Simpan urutan ID header
	}
	if err = rowsHeader.Err(); err != nil {
		log.Printf("Error after iterating deposit history header rows for partner ID %d: %v", partnerID, err)
		return nil, err
	}

	// Jika tidak ada riwayat sama sekali, kembalikan slice kosong
	if len(historyOrder) == 0 {
		return []partner.DepositHistoryHeader{}, nil
	}

	// Query kedua untuk mendapatkan SEMUA detail item dari SEMUA header milik partner ini
	// Gunakan JOIN untuk mendapatkan nama waste type
	queryDetails := `
		SELECT
			pdd.id, pdd.partner_deposit_history_id, pdd.waste_type_id, wt.name as waste_name,
			pdd.waste_weight, pdd.xpoin, pdd.photo, pdd.notes, pdd.status
		FROM partner_deposit_history_details pdd
		LEFT JOIN waste_types wt ON pdd.waste_type_id = wt.id
		JOIN partner_deposit_histories pdh ON pdd.partner_deposit_history_id = pdh.id -- Join ke header untuk filter partner
		WHERE pdh.partner_id = $1`

	rowsDetails, err := r.db.Query(queryDetails, partnerID)
	if err != nil {
		log.Printf("Error querying deposit history details for partner ID %d: %v", partnerID, err)
		return nil, err
	}
	defer rowsDetails.Close()

	// Iterasi hasil query detail dan masukkan ke map header yang sesuai
	for rowsDetails.Next() {
		var detail partner.DepositHistoryDetailItem
		var headerID int
		var wasteWeight sql.NullFloat64 // Baca DECIMAL sbg NullFloat64

		err := rowsDetails.Scan(
			&detail.ID, &headerID, &detail.WasteTypeID, &detail.WasteName,
			&wasteWeight, &detail.Xpoin, &detail.Photo, &detail.Notes, &detail.Status,
		)
		if err != nil {
			log.Printf("Error scanning deposit history detail row for partner ID %d: %v", partnerID, err)
			return nil, err
		}
		if wasteWeight.Valid {
			detail.WasteWeight = sql.NullString{String: fmt.Sprintf("%.2f", wasteWeight.Float64), Valid: true}
		}

		// Masukkan detail ke header yang benar di map
		if header, ok := historiesMap[headerID]; ok {
			header.Details = append(header.Details, detail)
		}
	}
	if err = rowsDetails.Err(); err != nil {
		log.Printf("Error after iterating deposit history detail rows for partner ID %d: %v", partnerID, err)
		return nil, err
	}

	// Susun hasil akhir sesuai urutan header
	finalHistories := make([]partner.DepositHistoryHeader, len(historyOrder))
	for i, id := range historyOrder {
		finalHistories[i] = *historiesMap[id] // Dereference pointer
	}

	return finalHistories, nil
}

// DeletePartnerByID menghapus partner berdasarkan ID
func (r *PartnerRepository) DeletePartnerByID(id int) error {
	query := "DELETE FROM partners WHERE id = $1"
	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Printf("Error deleting partner ID %d: %v", id, err)
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows // Partner tidak ditemukan
	}
	log.Printf("Partner deleted successfully for ID: %d", id)
	// Data terkait akan otomatis terhapus oleh ON DELETE CASCADE
	// atau di-SET NULL oleh constraint
	return nil
}

// --- Partner Wallet ---

// FindOrCreateWalletByPartnerID mencari wallet partner, atau membuatnya jika belum ada
func (r *PartnerRepository) FindOrCreateWalletByPartnerID(partnerID int) (*partner.PartnerWallet, error) {
	querySelect := `
		SELECT id, partner_id, balance, xpoin, created_at, updated_at
		FROM partner_wallets
		WHERE partner_id = $1`

	var wallet partner.PartnerWallet
	var balance float64 // Baca DECIMAL sebagai float64

	err := r.db.QueryRow(querySelect, partnerID).Scan(
		&wallet.ID, &wallet.PartnerID, &balance, &wallet.Xpoin, // Pastikan scan xpoin sbg int
		&wallet.CreatedAt, &wallet.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Wallet belum ada, buat baru
			log.Printf("Wallet not found for partner ID %d, creating new one.", partnerID)
			queryInsert := `
				INSERT INTO partner_wallets (partner_id, balance, xpoin)
				VALUES ($1, 0.00, 0)
				RETURNING id, partner_id, balance, xpoin, created_at, updated_at`

			errInsert := r.db.QueryRow(queryInsert, partnerID).Scan(
				&wallet.ID, &wallet.PartnerID, &balance, &wallet.Xpoin, // Pastikan scan xpoin sbg int
				&wallet.CreatedAt, &wallet.UpdatedAt,
			)
			if errInsert != nil {
				log.Printf("Error creating wallet for partner ID %d: %v", partnerID, errInsert)
				return nil, errInsert
			}
			wallet.Balance = fmt.Sprintf("%.2f", balance) // Format ke string
			log.Printf("Partner wallet created successfully for partner ID %d with ID %d", partnerID, wallet.ID)
			return &wallet, nil
		}
		// Error database lain saat SELECT
		log.Printf("Error finding wallet for partner ID %d: %v", partnerID, err)
		return nil, err
	}

	// Wallet ditemukan
	wallet.Balance = fmt.Sprintf("%.2f", balance) // Format ke string
	return &wallet, nil
}

// --- Partner Statistics ---

// FindOrCreateStatisticsByPartnerID mencari statistik partner, atau membuatnya jika belum ada
func (r *PartnerRepository) FindOrCreateStatisticsByPartnerID(partnerID int) (*partner.PartnerStatistic, error) {
	querySelect := `
		SELECT id, partner_id, waste, revenue, customer, transaction, created_at, updated_at
		FROM partner_statistics
		WHERE partner_id = $1`

	var stats partner.PartnerStatistic
	var waste, revenue float64 // Baca DECIMAL sebagai float64

	err := r.db.QueryRow(querySelect, partnerID).Scan(
		&stats.ID, &stats.PartnerID, &waste, &revenue, &stats.Customer, &stats.Transaction,
		&stats.CreatedAt, &stats.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Statistik belum ada, buat baru
			log.Printf("Statistics not found for partner ID %d, creating new entry.", partnerID)
			queryInsert := `
				INSERT INTO partner_statistics (partner_id, waste, revenue, customer, transaction)
				VALUES ($1, 0.00, 0.00, 0, 0)
				RETURNING id, partner_id, waste, revenue, customer, transaction, created_at, updated_at`

			errInsert := r.db.QueryRow(queryInsert, partnerID).Scan(
				&stats.ID, &stats.PartnerID, &waste, &revenue, &stats.Customer, &stats.Transaction,
				&stats.CreatedAt, &stats.UpdatedAt,
			)
			if errInsert != nil {
				log.Printf("Error creating statistics for partner ID %d: %v", partnerID, errInsert)
				return nil, errInsert
			}
			// Format ke string setelah insert
			stats.Waste = fmt.Sprintf("%.2f", waste)
			stats.Revenue = fmt.Sprintf("%.2f", revenue)
			log.Printf("Partner statistics created successfully for partner ID %d with ID %d", partnerID, stats.ID)
			return &stats, nil
		}
		// Error database lain saat SELECT
		log.Printf("Error finding statistics for partner ID %d: %v", partnerID, err)
		return nil, err
	}

	// Statistik ditemukan
	stats.Waste = fmt.Sprintf("%.2f", waste)
	stats.Revenue = fmt.Sprintf("%.2f", revenue)
	return &stats, nil
}

// --- Partner Withdraw Process Functions ---

// GetPartnerCurrentBalanceByID mengambil saldo partner saat ini
func (r *PartnerRepository) GetPartnerCurrentBalanceByID(partnerID int) (float64, error) {
	wallet, err := r.FindOrCreateWalletByPartnerID(partnerID)
	if err != nil {
		return 0, fmt.Errorf("gagal mendapatkan wallet partner: %w", err)
	}
	balance, err := strconv.ParseFloat(wallet.Balance, 64)
	if err != nil {
		log.Printf("Error parsing partner balance string for partner ID %d: %v", partnerID, err)
		return 0, errors.New("format saldo partner tidak valid")
	}
	return balance, nil
}

// ExecutePartnerWithdrawTransaction menjalankan pengurangan saldo dan pencatatan riwayat withdraw partner
func (r *PartnerRepository) ExecutePartnerWithdrawTransaction(partnerID int, amountToDeduct float64, fee float64, paymentMethodID int, accountNumber string) (string, error) {
	tx, err := r.db.Begin()
	if err != nil { /* handle tx begin error */
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit() /* handle commit error */
		}
	}()

	// 1. Kurangi saldo partner
	queryUpdateWallet := `
		UPDATE partner_wallets
		SET balance = balance - $1, updated_at = NOW()
		WHERE partner_id = $2 AND balance >= $1
		RETURNING balance`
	var remainingBalance float64
	err = tx.QueryRow(queryUpdateWallet, amountToDeduct, partnerID).Scan(&remainingBalance)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("saldo partner tidak mencukupi")
		}
		log.Printf("Error updating partner wallet balance during withdraw for partner ID %d: %v", partnerID, err)
		return "", errors.New("gagal mengupdate saldo partner")
	}

	// 2. Catat riwayat penarikan partner
	queryInsertHistory := `
		INSERT INTO partner_withdraw_histories (partner_id, payment_method_id, account_number, amount, fee, status, withdraw_time)
		VALUES ($1, $2, $3, $4, $5, 'Pending', NOW())
		RETURNING id`
	var withdrawID int
	amountRequested := amountToDeduct - fee
	err = tx.QueryRow(queryInsertHistory, partnerID, paymentMethodID, accountNumber, amountRequested, fee).Scan(&withdrawID)
	if err != nil {
		log.Printf("Error inserting partner withdraw history for partner ID %d: %v", partnerID, err)
		return "", errors.New("gagal mencatat riwayat penarikan partner")
	}

	orderID := fmt.Sprintf("WD-%d", withdrawID) // Prefix WD untuk Partner Withdraw
	log.Printf("Partner withdraw history created with ID %d (Order ID: %s) for partner ID %d", withdrawID, orderID, partnerID)

	return orderID, err // err akan nil jika commit berhasil
}

// --- Partner Top Up Process Functions ---

// ExecutePartnerTopupTransaction menjalankan penambahan saldo dan pencatatan riwayat top up partner
func (r *PartnerRepository) ExecutePartnerTopupTransaction(partnerID int, amountToAdd float64, paymentMethodID int) (string, error) {
	tx, err := r.db.Begin()
	if err != nil { /* handle tx begin error */
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit() /* handle commit error */
		}
	}()

	// 1. Tambahkan saldo partner
	queryUpdateWallet := `
		UPDATE partner_wallets
		SET balance = balance + $1, updated_at = NOW()
		WHERE partner_id = $2
		RETURNING balance`
	var currentBalance float64
	err = tx.QueryRow(queryUpdateWallet, amountToAdd, partnerID).Scan(&currentBalance)
	// Pastikan wallet ada SEBELUM memanggil ini (di service)
	if err != nil {
		log.Printf("Error updating partner wallet balance during topup for partner ID %d: %v", partnerID, err)
		if err == sql.ErrNoRows {
			return "", errors.New("wallet partner tidak ditemukan")
		}
		return "", errors.New("gagal mengupdate saldo partner")
	}

	// 2. Catat riwayat top up partner
	queryInsertHistory := `
		INSERT INTO partner_topup_histories (partner_id, payment_method_id, amount, status, topup_time)
		VALUES ($1, $2, $3, 'Completed', NOW()) -- Status langsung Completed (simulasi)
		RETURNING id`
	var topupID int
	err = tx.QueryRow(queryInsertHistory, partnerID, paymentMethodID, amountToAdd).Scan(&topupID)
	if err != nil {
		log.Printf("Error inserting partner topup history for partner ID %d: %v", partnerID, err)
		return "", errors.New("gagal mencatat riwayat top up partner")
	}

	orderID := fmt.Sprintf("TP-%d", topupID) // Prefix TP untuk Partner Topup
	log.Printf("Partner topup history created with ID %d (Order ID: %s) for partner ID %d", topupID, orderID, partnerID)

	return orderID, err // err akan nil jika commit berhasil
}

// --- Partner Transfer Xpoin ---

// ExecutePartnerTransferTransaction memproses transfer xpoin dari partner ke partner lain atau user
func (r *PartnerRepository) ExecutePartnerTransferTransaction(senderPartnerID, amount int, recipientUserID *int, recipientPartnerID *int, recipientEmail string) (string, error) {
	tx, err := r.db.Begin()
	if err != nil { /* handle tx begin error */
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit() /* handle commit error */
		}
	}()

	// 1. Kurangi Xpoin pengirim (partner)
	queryUpdateSender := `
        UPDATE partner_wallets
        SET xpoin = xpoin - $1, updated_at = NOW()
        WHERE partner_id = $2 AND xpoin >= $1
        RETURNING xpoin`
	var senderRemainingXpoin int
	err = tx.QueryRow(queryUpdateSender, amount, senderPartnerID).Scan(&senderRemainingXpoin)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("xpoin partner tidak mencukupi")
		}
		log.Printf("Error updating sender partner wallet during transfer: %v", err)
		return "", errors.New("gagal mengupdate xpoin pengirim")
	}

	// 2. Tambah Xpoin penerima (bisa user atau partner)
	if recipientUserID != nil { // Jika penerima adalah User
		queryUpdateRecipient := `
            UPDATE user_wallets SET xpoin = xpoin + $1, updated_at = NOW() WHERE user_id = $2`
		res, errExec := tx.Exec(queryUpdateRecipient, amount, *recipientUserID)
		if errExec != nil {
			log.Printf("Error updating recipient user wallet during transfer: %v", errExec)
			return "", errors.New("gagal mengupdate xpoin penerima (user)")
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			return "", errors.New("wallet user penerima tidak ditemukan")
		}
	} else if recipientPartnerID != nil { // Jika penerima adalah Partner
		queryUpdateRecipient := `
            UPDATE partner_wallets SET xpoin = xpoin + $1, updated_at = NOW() WHERE partner_id = $2`
		res, errExec := tx.Exec(queryUpdateRecipient, amount, *recipientPartnerID)
		if errExec != nil {
			log.Printf("Error updating recipient partner wallet during transfer: %v", errExec)
			return "", errors.New("gagal mengupdate xpoin penerima (partner)")
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			return "", errors.New("wallet partner penerima tidak ditemukan")
		}
	} else {
		// Ini seharusnya tidak terjadi jika validasi service benar
		return "", errors.New("penerima tidak valid")
	}

	// 3. Catat riwayat transfer partner
	queryInsertHistory := `
        INSERT INTO partner_transfer_histories (partner_id, amount, recipient_email, status, transfer_time)
        VALUES ($1, $2, $3, 'Completed', NOW())
        RETURNING id`
	var transferID int
	// Simpan amount sebagai DECIMAL (meskipun asalnya int xpoin)
	err = tx.QueryRow(queryInsertHistory, senderPartnerID, float64(amount), recipientEmail).Scan(&transferID)
	if err != nil {
		log.Printf("Error inserting partner transfer history: %v", err)
		return "", errors.New("gagal mencatat riwayat transfer partner")
	}

	orderID := fmt.Sprintf("TF-%d", transferID) // Prefix TF for Partner Transfer
	log.Printf("Partner transfer history created ID %d (Order: %s) from %d to %s", transferID, orderID, senderPartnerID, recipientEmail)

	return orderID, err // err akan nil jika commit berhasil
}

// --- Partner Conversion Functions ---

// ExecutePartnerConversionTransaction memproses perubahan balance dan xpoin partner
func (r *PartnerRepository) ExecutePartnerConversionTransaction(
	partnerID int,
	xpoinChange int,
	balanceChange float64,
	conversionType string,
	amountXpInvolved int,
	amountRpInvolved float64,
	rate float64,
) (*partner.PartnerWallet, error) { // Kembalikan wallet terbaru

	tx, err := r.db.Begin()
	if err != nil { return nil, errors.New("gagal memulai transaksi database") }
	defer func() {
		if p := recover(); p != nil { tx.Rollback(); panic(p) } else
		if err != nil { tx.Rollback() } else
		{ err = tx.Commit(); /* handle commit error */ }
	}()

	// 1. Update Wallet Partner (dengan validasi)
	queryUpdateWallet := `
		UPDATE partner_wallets
		SET xpoin = xpoin + $1, balance = balance + $2, updated_at = NOW()
		WHERE partner_id = $3
		  AND xpoin + $1 >= 0
		  AND balance + $2 >= 0
		RETURNING id, partner_id, balance, xpoin, created_at, updated_at`

	var updatedWallet partner.PartnerWallet
	var currentBalance float64

	err = tx.QueryRow(queryUpdateWallet, xpoinChange, balanceChange, partnerID).Scan(
		&updatedWallet.ID, &updatedWallet.PartnerID, &currentBalance, &updatedWallet.Xpoin,
		&updatedWallet.CreatedAt, &updatedWallet.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			if xpoinChange < 0 { return nil, errors.New("xpoin partner tidak mencukupi") }
			if balanceChange < 0 { return nil, errors.New("saldo partner tidak mencukupi") }
			return nil, errors.New("wallet partner tidak ditemukan atau saldo/xpoin tidak mencukupi")
		}
		log.Printf("Error updating partner wallet during conversion for partner ID %d: %v", partnerID, err)
		return nil, errors.New("gagal mengupdate wallet partner")
	}
	updatedWallet.Balance = fmt.Sprintf("%.2f", currentBalance)

	// 2. Catat Riwayat Konversi Partner
	queryInsertHistory := `
		INSERT INTO partner_conversion_histories
			(partner_id, type, amount_xp, amount_rp, rate, conversion_time)
		VALUES ($1, $2, $3, $4, $5, NOW())`

	_, err = tx.Exec(queryInsertHistory, partnerID, conversionType, amountXpInvolved, amountRpInvolved, rate)
	if err != nil {
		log.Printf("Error inserting partner conversion history for partner ID %d: %v", partnerID, err)
		// Jangan gagalkan transaksi utama jika log gagal
	}

	log.Printf("Partner conversion successful for partner ID %d: %s", partnerID, conversionType)
	return &updatedWallet, err // err akan nil jika commit berhasil
}

