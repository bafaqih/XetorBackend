package partner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"strconv"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"golang.org/x/crypto/bcrypt"
	"xetor.id/backend/internal/auth" // Import JWT generator
	"xetor.id/backend/internal/config"
)

// Definisikan interface repository yang dibutuhkan
type PartnerRepository interface {
	SavePartner(p *Partner) error
	FindPartnerByEmail(email string) (*Partner, error)
	FindXetorPartnerStatusByID(partnerID int) (string, error)
	FindPartnerByID(id int) (*Partner, error)
	UpdatePartnerProfile(id int, req *UpdatePartnerProfileRequest) error
	UpdatePartnerPhotoURL(id int, photoURL string) error
}

type PartnerService struct {
	repo PartnerRepository
}

func NewPartnerService(repo PartnerRepository) *PartnerService {
	return &PartnerService{repo: repo}
}

// RegisterPartner memproses registrasi partner baru
func (s *PartnerService) RegisterPartner(req PartnerSignUpRequest) (*Partner, error) {
	// Cek apakah email sudah ada (bisa juga ditangani oleh constraint DB)
	existingPartner, err := s.repo.FindPartnerByEmail(req.Email)
	if err != nil && err != sql.ErrNoRows { // Hanya handle error teknis
		return nil, errors.New("gagal memeriksa email")
	}
	if existingPartner != nil {
		return nil, errors.New("email sudah terdaftar")
	}
	// TODO: Cek duplikasi nomor telepon jika perlu

	partner := &Partner{
		BusinessName: req.BusinessName,
		Email:        req.Email,
		Phone:        sql.NullString{String: req.Phone, Valid: req.Phone != ""},
		Password:     req.Password, // Repo akan hash ini
	}

	err = s.repo.SavePartner(partner)
	if err != nil {
		return nil, err // Repo sudah memberi pesan error yang sesuai
	}

	// Jangan kirim password hash kembali
	partner.Password = ""
	return partner, nil
}

// LoginPartner memvalidasi login partner dan membuat token
func (s *PartnerService) LoginPartner(req PartnerLoginRequest) (string, string, error) {
	// 1. Cari partner berdasarkan email
	partner, err := s.repo.FindPartnerByEmail(req.Email)
	if err != nil {
		return "", "", errors.New("gagal mencari partner") // Kembalikan string kosong untuk token & status
	}
	if partner == nil {
		return "", "", errors.New("kredensial tidak valid")
	}

	// 2. Bandingkan password
	err = bcrypt.CompareHashAndPassword([]byte(partner.Password), []byte(req.Password))
	if err != nil {
		return "", "", errors.New("kredensial tidak valid")
	}

	// 3. Cek status approval
	status, err := s.repo.FindXetorPartnerStatusByID(partner.ID)
	if err != nil {
		if err != sql.ErrNoRows && status != "Not Registered" {
			log.Printf("Error checking partner status for ID %d: %v", partner.ID, err)
			return "", "", errors.New("gagal memeriksa status partner")
		}
        if status == "" { status = "Not Registered" }
	}

	// 4. Buat token JWT
	token, err := auth.GenerateToken(partner.ID, "partner")
	if err != nil {
		log.Printf("Error generating token for partner ID %d: %v", partner.ID, err)
		return "", "", errors.New("gagal membuat sesi login")
	}

	// 5. Kembalikan HANYA token dan status aktual
	return token, status, nil
}

func (s *PartnerService) GetProfile(partnerIDStr string) (*Partner, error) {
    partnerID, err := strconv.Atoi(partnerIDStr)
    if err != nil { return nil, errors.New("ID partner tidak valid") }

    partner, err := s.repo.FindPartnerByID(partnerID)
    if err != nil { return nil, errors.New("gagal mengambil profil partner") }
    if partner == nil { return nil, errors.New("partner tidak ditemukan") }

    partner.Password = "" // Jangan kirim hash
    return partner, nil
}

// UpdateProfile memproses update data profil partner
func (s *PartnerService) UpdateProfile(partnerIDStr string, req UpdatePartnerProfileRequest) error {
	partnerID, err := strconv.Atoi(partnerIDStr)
	if err != nil {
		return errors.New("ID partner tidak valid")
	}

	// Cek apakah ada data yang diupdate
	if req.BusinessName == "" && req.Email == "" && req.Phone == "" {
		return errors.New("tidak ada data untuk diupdate")
	}
	// TODO: Tambahkan validasi format email jika perlu di sini

	err = s.repo.UpdatePartnerProfile(partnerID, &req)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("partner tidak ditemukan")
		}
		return err // Termasuk error email/phone duplikat dari repo
	}
	return nil
}

// --- Partner Photo Upload Service Method ---

// UploadProfilePhoto menghandle upload file ke Cloudinary dan update DB
func (s *PartnerService) UploadProfilePhoto(partnerIDStr string, fileHeader *multipart.FileHeader) (string, error) {
	partnerID, err := strconv.Atoi(partnerIDStr)
	if err != nil {
		return "", errors.New("ID partner tidak valid")
	}

	// 1. Buka file yang diupload
	file, err := fileHeader.Open()
	if err != nil {
		log.Printf("Error opening uploaded file: %v", err)
		return "", errors.New("gagal membaca file foto")
	}
	defer file.Close()

	// 2. Setup Cloudinary
	cldURL := config.GetCloudinaryURL()
	cld, err := cloudinary.NewFromURL(cldURL)
	if err != nil {
		log.Printf("Error initializing Cloudinary: %v", err)
		return "", errors.New("gagal terhubung ke penyedia penyimpanan foto")
	}

	// 3. Tentukan parameter upload (termasuk folder)
	overwrite := true
	uploadParams := uploader.UploadParams{
		Folder:    "xetor/partners",                     // Simpan di folder xetor/partners
		PublicID:  fmt.Sprintf("profile_%d", partnerID), // Nama file unik (opsional, Cloudinary bisa generate)
		Overwrite: &overwrite,                           // Timpa file lama jika ada
		Format:    "jpg",                                // Contoh: konversi ke jpg
		// Transformation: "...", // Bisa tambahkan transformasi (resize, crop, dll)
	}

	// 4. Upload ke Cloudinary
	ctx := context.Background()
	uploadResult, err := cld.Upload.Upload(ctx, file, uploadParams)
	if err != nil {
		log.Printf("Error uploading to Cloudinary: %v", err)
		return "", errors.New("gagal mengunggah foto")
	}

	// 5. Update URL foto di database
	photoURL := uploadResult.SecureURL // Gunakan SecureURL (HTTPS)
	err = s.repo.UpdatePartnerPhotoURL(partnerID, photoURL)
	if err != nil {
		// Jika DB gagal, idealnya kita coba hapus file di Cloudinary (rollback manual)
		log.Printf("DB update failed after Cloudinary upload for partner %d: %v", partnerID, err)
		// cld.Upload.Destroy(...) // Implementasi rollback jika perlu
		return "", err // Error dari repo (partner not found atau DB error)
	}

	log.Printf("Partner %d profile photo updated to: %s", partnerID, photoURL)
	return photoURL, nil // Kembalikan URL foto baru
}
