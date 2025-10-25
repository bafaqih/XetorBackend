package repository

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"fmt"
	"strconv"

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
		if p := recover(); p != nil { tx.Rollback(); panic(p) } else
		if err != nil { tx.Rollback() } else
		{ err = tx.Commit() }
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
		if err == sql.ErrNoRows { return nil, nil } // Tidak ditemukan
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

	if req.BusinessName != "" { fields = append(fields, fmt.Sprintf("business_name = $%d", argId)); args = append(args, req.BusinessName); argId++ }
	if req.Email != "" { fields = append(fields, fmt.Sprintf("email = $%d", argId)); args = append(args, req.Email); argId++ }
	if req.Phone != "" { fields = append(fields, fmt.Sprintf("phone = $%d", argId)); args = append(args, sql.NullString{String: req.Phone, Valid: true}); argId++ }

	if len(fields) == 0 { return nil }
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
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
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
	if sched.OpenTime == "" { openTime = nil } else { openTime = sched.OpenTime }
	if sched.CloseTime == "" { closeTime = nil } else { closeTime = sched.CloseTime }

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
	if err != nil { return nil, err }

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
		if err == sql.ErrNoRows { return nil, nil } // Tidak ditemukan / bukan milik partner
		log.Printf("Error getting waste price detail ID %d for partner ID %d: %v", detailID, partnerID, err)
		return nil, err
	}
	pd.Price = fmt.Sprintf("%.2f", priceDB)
	return &pd, nil
}


// UpdateWastePriceDetail mengupdate item harga sampah
func (r *PartnerRepository) UpdateWastePriceDetail(detailID int, partnerID int, detail *partner.PartnerWastePriceDetail) error {
	headerID, err := r.FindOrCreateWastePriceHeader(partnerID)
	if err != nil { return err }

	fields := []string{}
	args := []interface{}{}
	argId := 1

	// Bangun query update dinamis
	if detail.Image.Valid { fields = append(fields, fmt.Sprintf("image = $%d", argId)); args = append(args, detail.Image); argId++ }
	if detail.Name != "" { fields = append(fields, fmt.Sprintf("name = $%d", argId)); args = append(args, detail.Name); argId++ }
	if detail.Price != "" {
		priceFloat, err := strconv.ParseFloat(detail.Price, 64)
		if err != nil { return errors.New("format harga tidak valid") }
		fields = append(fields, fmt.Sprintf("price = $%d", argId)); args = append(args, priceFloat); argId++
	}
	if detail.Unit != "" { fields = append(fields, fmt.Sprintf("unit = $%d", argId)); args = append(args, detail.Unit); argId++ }
	// Selalu update xpoin jika price diupdate atau xpoin dikirim (meskipun 0)
	fields = append(fields, fmt.Sprintf("xpoin = $%d", argId)); args = append(args, detail.Xpoin); argId++


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
	if err != nil { log.Printf("Error updating waste price detail ID %d: %v", detailID, err); return errors.New("gagal mengupdate detail harga sampah") }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows /* Not found or not owned */ }
	log.Printf("Waste price detail updated for ID: %d", detailID)
	return nil
}


// DeleteWastePriceDetail menghapus item harga sampah
func (r *PartnerRepository) DeleteWastePriceDetail(detailID int, partnerID int) error {
	headerID, err := r.FindOrCreateWastePriceHeader(partnerID)
	if err != nil { return err }

	query := `DELETE FROM partner_waste_price_details WHERE id = $1 AND partner_waste_price_id = $2`
	result, err := r.db.Exec(query, detailID, headerID)
	if err != nil { log.Printf("Error deleting waste price detail ID %d: %v", detailID, err); return errors.New("gagal menghapus detail harga sampah") }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows /* Not found or not owned */ }
	log.Printf("Waste price detail deleted for ID: %d", detailID)
	return nil
}