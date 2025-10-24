package repository

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"xetor.id/backend/internal/domain/partner" // Import domain partner
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