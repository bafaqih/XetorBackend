package partner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"strconv"
	"regexp"
	"strings"
	"math"
	"time"


	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"golang.org/x/crypto/bcrypt"
	"xetor.id/backend/internal/auth" // Import JWT generator
	"xetor.id/backend/internal/config"
)

const conversionRateRpToXp = 1.0 / 5.0 // 1 Rp = 0.2 Xp

// Definisikan interface repository yang dibutuhkan
type PartnerRepository interface {
	SavePartner(p *Partner) error
	FindPartnerByEmail(email string) (*Partner, error)
	FindXetorPartnerStatusByID(partnerID int) (string, error)
	FindPartnerByID(id int) (*Partner, error)
	UpdatePartnerProfile(id int, req *UpdatePartnerProfileRequest) error
	UpdatePartnerPhotoURL(id int, photoURL string) error
	GetCurrentPasswordHashByID(id int) (string, error)
	UpdatePassword(id int, newHashedPassword string) error

	// Alamat partner
	GetAddressByPartnerID(partnerID int) (*PartnerAddress, error)
	UpsertAddress(addr *PartnerAddress) error

	// Jadwal operasional partner
	GetScheduleByPartnerID(partnerID int) (*PartnerSchedule, error) // <-- Ganti nama & return type
	UpsertSchedule(sched *PartnerSchedule) error

	// Harga sampah partner
	FindOrCreateWastePriceHeader(partnerID int) (int, error)
	CreateWastePriceDetail(detail *PartnerWastePriceDetail) error
	GetWastePriceDetailsByPartnerID(partnerID int) ([]PartnerWastePriceDetail, error)
	GetWastePriceDetailByID(detailID int, partnerID int) (*PartnerWastePriceDetail, error)
	UpdateWastePriceDetail(detailID int, partnerID int, detail *PartnerWastePriceDetail) error
	DeleteWastePriceDetail(detailID int, partnerID int) error
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

// ChangePassword memvalidasi dan mengubah password partner
func (s *PartnerService) ChangePassword(partnerIDStr string, req ChangePasswordRequest) error {
	// 1. Validasi input dasar
	if req.NewPassword != req.ConfirmNewPassword {
		return errors.New("konfirmasi password baru tidak cocok")
	}
	if len(req.NewPassword) < 6 {
		return errors.New("password baru minimal 6 karakter")
	}

	// 2. Konversi partnerID
	partnerID, err := strconv.Atoi(partnerIDStr)
	if err != nil { return errors.New("ID partner tidak valid") }

	// 3. Ambil hash password saat ini
	currentPasswordHash, err := s.repo.GetCurrentPasswordHashByID(partnerID)
	if err != nil { return err }
	if currentPasswordHash == "" { return errors.New("partner tidak ditemukan") }

	// 4. Bandingkan password lama
	err = bcrypt.CompareHashAndPassword([]byte(currentPasswordHash), []byte(req.OldPassword))
	if err != nil { return errors.New("password lama salah") }

	// 5. Hash password baru
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil { return errors.New("gagal memproses password baru") }

	// 6. Update password di DB
	err = s.repo.UpdatePassword(partnerID, string(newHashedPassword))
	if err != nil {
		if err == sql.ErrNoRows { return errors.New("partner tidak ditemukan saat update") }
		return err
	}
	return nil // Sukses
}

// --- Partner Address Service Methods ---

// GetAddress mengambil alamat usaha partner
func (s *PartnerService) GetAddress(partnerIDStr string) (*PartnerAddress, error) {
	partnerID, err := strconv.Atoi(partnerIDStr)
	if err != nil { return nil, errors.New("ID partner tidak valid") }

	address, err := s.repo.GetAddressByPartnerID(partnerID)
	if err != nil { return nil, errors.New("gagal mengambil alamat usaha") }
	// Tidak perlu cek nil, handler akan handle 404 jika address == nil
	return address, nil
}

// UpdateAddress membuat atau memperbarui alamat usaha partner
func (s *PartnerService) UpdateAddress(partnerIDStr string, req UpdatePartnerAddressRequest) (*PartnerAddress, error) {
	partnerID, err := strconv.Atoi(partnerIDStr)
	if err != nil { return nil, errors.New("ID partner tidak valid") }

	addr := &PartnerAddress{
		PartnerID:   partnerID,
		Address:     req.Address,
		CityRegency: req.CityRegency,
		Province:    req.Province,
		PostalCode:  sql.NullString{String: req.PostalCode, Valid: req.PostalCode != ""},
	}

	err = s.repo.UpsertAddress(addr)
	if err != nil { return nil, err } // Repo sudah handle error

	// Kembalikan data alamat yang baru disimpan (termasuk ID dan timestamp)
	return addr, nil
}

// --- Partner Schedule Service Methods ---

// GetSchedule mengambil jadwal operasional partner (single row)
func (s *PartnerService) GetSchedule(partnerIDStr string) (*PartnerSchedule, error) {
	partnerID, err := strconv.Atoi(partnerIDStr); if err != nil { return nil, errors.New("ID partner tidak valid") }

	schedule, err := s.repo.GetScheduleByPartnerID(partnerID)
	if err != nil { return nil, errors.New("gagal mengambil jadwal operasional") }

	// Jika belum ada jadwal, kembalikan struct kosong tapi valid, bukan nil
	if schedule == nil {
		return &PartnerSchedule{PartnerID: partnerID, DaysOpen: []string{}, OperatingStatus: "Closed"}, nil
	}
	return schedule, nil
}

// UpdateSchedule membuat/memperbarui jadwal operasional partner (single row)
func (s *PartnerService) UpdateSchedule(partnerIDStr string, req UpdateScheduleRequest) (*PartnerSchedule, error) {
	partnerID, err := strconv.Atoi(partnerIDStr); if err != nil { /* ... */ }

	// 1. Ambil jadwal yang ada SEKARANG (jika ada)
	currentSchedule, err := s.repo.GetScheduleByPartnerID(partnerID)
	if err != nil { return nil, errors.New("gagal mengambil jadwal saat ini") }
	// Jika belum ada, buat struct default
	if currentSchedule == nil {
		currentSchedule = &PartnerSchedule{PartnerID: partnerID, OperatingStatus: "Closed"} // Default jika belum ada
	}


	// 2. Validasi input & tentukan nilai baru (gunakan nilai lama jika request kosong)
	timeRegex := regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)
	validStatuses := map[string]bool{"Open": true, "Closed": true}

	finalDaysOpen := currentSchedule.DaysOpen // Default: pakai yg lama
	if len(req.DaysOpen) > 0 { // Hanya proses jika DaysOpen dikirim
        // Validasi hari (ambil dari kode sebelumnya)
        validDays := map[string]bool{"Senin": true, "Selasa": true, "Rabu": true, "Kamis": true, "Jumat": true, "Sabtu": true, "Minggu": true}
        uniqueDays := make(map[string]bool)
        validDaysList := []string{}
        for _, day := range req.DaysOpen {
            trimmedDay := strings.TrimSpace(day)
            if !validDays[trimmedDay] { return nil, fmt.Errorf("nama hari tidak valid: %s", trimmedDay) }
            if !uniqueDays[trimmedDay] { uniqueDays[trimmedDay] = true; validDaysList = append(validDaysList, trimmedDay) }
        }
        finalDaysOpen = validDaysList
	}


	finalOpenTime := currentSchedule.OpenTime // Default: pakai yg lama
	if req.OpenTime != "" {
		if !timeRegex.MatchString(req.OpenTime) { return nil, errors.New("format jam buka tidak valid (HH:MM)") }
		finalOpenTime = req.OpenTime
	}

	finalCloseTime := currentSchedule.CloseTime // Default: pakai yg lama
	if req.CloseTime != "" {
		if !timeRegex.MatchString(req.CloseTime) { return nil, errors.New("format jam tutup tidak valid (HH:MM)") }
		finalCloseTime = req.CloseTime
	}

    finalOpStatus := currentSchedule.OperatingStatus // Default: pakai yg lama
	if req.OperatingStatus != "" {
		if !validStatuses[req.OperatingStatus] { return nil, errors.New("status operasional tidak valid (Open/Closed)") }
		finalOpStatus = req.OperatingStatus
	}


	// Siapkan data final untuk disimpan
	schedToSave := &PartnerSchedule{
		PartnerID:       partnerID,
		DaysOpen:        finalDaysOpen,
		OpenTime:        finalOpenTime,
		CloseTime:       finalCloseTime,
		OperatingStatus: finalOpStatus,
	}

	// Panggil repo untuk upsert
	err = s.repo.UpsertSchedule(schedToSave)
	if err != nil { return nil, err }

	return schedToSave, nil
}

// --- Partner Waste Price Service Methods ---

// calculateXpoin menghitung xpoin dari harga (pembulatan ke bawah)
func calculateXpoin(price float64) int {
	if price <= 0 {
		return 0
	}
	return int(math.Floor(price * conversionRateRpToXp))
}

// uploadWastePriceImage mengupload gambar ke cloudinary
func (s *PartnerService) uploadWastePriceImage(partnerID int, detailID int, fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader == nil {
		return "", nil // Tidak ada file yg diupload
	}
	file, err := fileHeader.Open()
	if err != nil { return "", errors.New("gagal membaca file gambar") }
	defer file.Close()

	cldURL := config.GetCloudinaryURL()
	cld, err := cloudinary.NewFromURL(cldURL)
	if err != nil { return "", errors.New("gagal terhubung ke penyedia penyimpanan foto") }

	// Nama file unik di Cloudinary (misal: waste_price_<partnerID>_<detailID>)
	publicID := fmt.Sprintf("waste_price_%d_%d", partnerID, detailID)
	if detailID == 0 { // Jika ini adalah create (belum punya detailID)
		publicID = fmt.Sprintf("waste_price_%d_new_%d", partnerID, time.Now().UnixNano()) // Gunakan timestamp
	}


	uploadParams := uploader.UploadParams{
		Folder:         "xetor/waste_prices",
		PublicID:       publicID,
		Overwrite:      func() *bool { b := true; return &b }(),
		Format:         "jpg", // Atau biarkan Cloudinary deteksi otomatis
	}

	ctx := context.Background()
	uploadResult, err := cld.Upload.Upload(ctx, file, uploadParams)
	if err != nil {
		log.Printf("Error uploading waste price image to Cloudinary: %v", err)
		return "", errors.New("gagal mengunggah gambar sampah")
	}
	return uploadResult.SecureURL, nil
}


// CreateWastePrice menambahkan item harga sampah baru
func (s *PartnerService) CreateWastePrice(partnerIDStr string, req WastePriceRequest, imageFile *multipart.FileHeader) (*PartnerWastePriceDetail, error) {
	partnerID, err := strconv.Atoi(partnerIDStr); if err != nil { return nil, errors.New("ID partner tidak valid") }

	// Validasi dasar
	if req.Price <= 0 { return nil, errors.New("harga harus positif") }
	// TODO: Validasi unit?

	// Dapatkan atau buat header
	headerID, err := s.repo.FindOrCreateWastePriceHeader(partnerID)
	if err != nil { return nil, err }

	// Hitung Xpoin
	xpoin := calculateXpoin(req.Price)

	// Siapkan data detail (Image URL diisi setelah upload)
	detail := &PartnerWastePriceDetail{
		PartnerWastePriceID: headerID,
		Name:                req.Name,
		Price:               fmt.Sprintf("%.2f", req.Price), // Simpan sbg string di struct
		Unit:                req.Unit,
		Xpoin:               xpoin,
	}

	// Upload gambar jika ada
	imageURL, err := s.uploadWastePriceImage(partnerID, 0, imageFile) // detailID 0 karena belum dibuat
	if err != nil { return nil, err }
	if imageURL != "" {
		detail.Image = sql.NullString{String: imageURL, Valid: true}
	}


	// Simpan detail ke DB
	err = s.repo.CreateWastePriceDetail(detail)
	if err != nil { return nil, err }

	return detail, nil
}

// GetAllWastePrices mengambil semua item harga sampah partner
func (s *PartnerService) GetAllWastePrices(partnerIDStr string) ([]PartnerWastePriceDetail, error) {
	partnerID, err := strconv.Atoi(partnerIDStr); if err != nil { return nil, errors.New("ID partner tidak valid") }
	details, err := s.repo.GetWastePriceDetailsByPartnerID(partnerID)
	if err != nil { return nil, errors.New("gagal mengambil daftar harga sampah") }
	if details == nil { return []PartnerWastePriceDetail{}, nil } // Return array kosong
	return details, nil
}

// GetWastePriceByID mengambil satu item harga
func (s *PartnerService) GetWastePriceByID(detailID int, partnerIDStr string) (*PartnerWastePriceDetail, error) {
	partnerID, err := strconv.Atoi(partnerIDStr); if err != nil { return nil, errors.New("ID partner tidak valid") }
	detail, err := s.repo.GetWastePriceDetailByID(detailID, partnerID)
	if err != nil { return nil, errors.New("gagal mengambil detail harga sampah") }
	// Tidak perlu cek nil, repo sudah handle
	return detail, nil
}

// UpdateWastePrice mengupdate item harga sampah
func (s *PartnerService) UpdateWastePrice(detailID int, partnerIDStr string, req UpdateWastePriceRequest, imageFile *multipart.FileHeader) (*PartnerWastePriceDetail, error) {
	partnerID, err := strconv.Atoi(partnerIDStr)
	if err != nil {
		return nil, errors.New("ID partner tidak valid")
	}

	// Cek apakah item ada dan milik partner
	existingDetail, err := s.repo.GetWastePriceDetailByID(detailID, partnerID)
	if err != nil {
		return nil, errors.New("gagal memeriksa detail harga sampah")
	}
	if existingDetail == nil {
		return nil, errors.New("detail harga sampah tidak ditemukan atau bukan milik Anda")
	}

	// Siapkan data update
	updateData := &PartnerWastePriceDetail{
		ID: detailID, // ID untuk WHERE clause
	}
	needsUpdate := false // Flag untuk cek apakah ada yg diupdate

	if req.Name != "" {
		updateData.Name = req.Name
		needsUpdate = true
	}
	if req.Price > 0 {
		updateData.Price = fmt.Sprintf("%.2f", req.Price) // Update price
		updateData.Xpoin = calculateXpoin(req.Price)      // Hitung ulang Xpoin
		needsUpdate = true
	}
	if req.Unit != "" {
		updateData.Unit = req.Unit
		needsUpdate = true
	}

	imageURL, err := s.uploadWastePriceImage(partnerID, detailID, imageFile)
	if err != nil { return nil, err }
	if imageURL != "" {
		updateData.Image = sql.NullString{String: imageURL, Valid: true}
		needsUpdate = true
	}

	// Hanya panggil repo jika ada perubahan
	if !needsUpdate {
		log.Println("UpdateWastePrice: No fields to update.")
		return existingDetail, nil // Kembalikan data lama jika tidak ada update
	}

	err = s.repo.UpdateWastePriceDetail(detailID, partnerID, updateData)
	if err != nil { return nil, err }

	return s.repo.GetWastePriceDetailByID(detailID, partnerID) // Ambil data terbaru
}


// DeleteWastePrice menghapus item harga sampah
func (s *PartnerService) DeleteWastePrice(detailID int, partnerIDStr string) error {
	partnerID, err := strconv.Atoi(partnerIDStr); if err != nil { return errors.New("ID partner tidak valid") }

	// TODO: Hapus gambar dari Cloudinary sebelum hapus dari DB? (Opsional)
	// detail, _ := s.repo.GetWastePriceDetailByID(detailID, partnerID)
	// if detail != nil && detail.Image.Valid {
	//    cld.Upload.Destroy(...)
	// }

	err = s.repo.DeleteWastePriceDetail(detailID, partnerID)
	if err != nil {
		if err == sql.ErrNoRows { return errors.New("detail harga sampah tidak ditemukan atau bukan milik Anda") }
		return errors.New("gagal menghapus detail harga sampah")
	}
	return nil
}