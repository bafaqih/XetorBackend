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
	"sort"


	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"golang.org/x/crypto/bcrypt"
	"xetor.id/backend/internal/auth" // Import JWT generator
	"xetor.id/backend/internal/config"
	"xetor.id/backend/internal/domain/user"
	"xetor.id/backend/internal/temporary_token"
)

const conversionRateRpToXp = 1.0 / 5.0 // 1 Rp = 0.2 Xp
const conversionRateXpToRp = 5.0   // 1 Xp = 5 Rp

const (
	minWithdrawalAmount = 10000.0
	withdrawalFee       = 2500.0
	minTopupAmount      = 10000.0
)

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
	DeletePartnerByID(id int) error
	FindOrCreateWalletByPartnerID(partnerID int) (*PartnerWallet, error)
	FindOrCreateStatisticsByPartnerID(partnerID int) (*PartnerStatistic, error)

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

	// Riwayat transaksi partner
	GetWithdrawHistoryForPartner(partnerID int) ([]PartnerTransactionHistoryItem, error)
	GetTopupHistoryForPartner(partnerID int) ([]PartnerTransactionHistoryItem, error)
	GetConversionHistoryForPartner(partnerID int) ([]PartnerTransactionHistoryItem, error)
	GetTransferHistoryForPartner(partnerID int) ([]PartnerTransactionHistoryItem, error)
	GetDepositHistoryByPartnerID(partnerID int) ([]DepositHistoryHeader, error)

	// Withdrawal execution
	GetPartnerCurrentBalanceByID(partnerID int) (float64, error)
	ExecutePartnerWithdrawTransaction(partnerID int, amountToDeduct float64, fee float64, paymentMethodID int, accountNumber string) (string, error)
	
	// Topup execution
	ExecutePartnerTopupTransaction(partnerID int, amountToAdd float64, paymentMethodID int) (string, error)

	// Transfer Xpoin
	ExecutePartnerTransferTransaction(senderPartnerID, amount int, recipientUserID *int, recipientPartnerID *int, recipientEmail string) (string, error)

	// Conversion execution
	ExecutePartnerConversionTransaction(partnerID int, xpoinChange int, balanceChange float64, conversionType string, amountXpInvolved int, amountRpInvolved float64, rate float64) (*PartnerWallet, error)
}

type UserRepositoryForPartner interface {
	 FindByEmail(email string) (*user.User, error)
     FindOrCreateWalletByUserID(userID int) (*user.UserWallet, error)
	 FindUserIDByEmail(email string) (int, error)
     FindByID(id int) (*user.User, error)
}

type PartnerService struct {
	repo PartnerRepository
	userRepo UserRepositoryForPartner
	tokenStore *temporary_token.TokenStore
}

func NewPartnerService(repo PartnerRepository, userRepo UserRepositoryForPartner, tokenStore *temporary_token.TokenStore) *PartnerService {
	return &PartnerService{repo: repo, userRepo: userRepo, tokenStore: tokenStore}
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
	xpoin := calculateXpoin(req.Price)

	detail := &PartnerWastePriceDetail{
		PartnerWastePriceID: headerID,
		WasteDetailID:       sql.NullInt32{Int32: int32(req.WasteDetailID), Valid: true}, // Set WasteDetailID
		Name:                req.Name,
		Price:               fmt.Sprintf("%.2f", req.Price),
		Unit:                req.Unit,
		Xpoin:               xpoin,
	}

	imageURL, err := s.uploadWastePriceImage(partnerID, 0, imageFile)
	if err != nil { return nil, err }
	if imageURL != "" { detail.Image = sql.NullString{String: imageURL, Valid: true} }

	err = s.repo.CreateWastePriceDetail(detail)
	if err != nil { return nil, err }

	// Ambil data lagi untuk response agar WasteDetailID terisi jika NULL
    return s.repo.GetWastePriceDetailByID(detail.ID, partnerID)
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
		ID:    detailID, // Untuk repo
		Image: sql.NullString{Valid: false}, // Default: jangan update image
	}
	needsUpdate := false

	if req.Name != "" { updateData.Name = req.Name; needsUpdate = true }
	if req.Unit != "" { updateData.Unit = req.Unit; needsUpdate = true }
	if req.WasteDetailID != nil { // Jika WasteDetailID dikirim
		// TODO: Validasi apakah *req.WasteDetailID valid?
		updateData.WasteDetailID = sql.NullInt32{Int32: int32(*req.WasteDetailID), Valid: true}
		needsUpdate = true
	}
	if req.Price > 0 {
		updateData.Price = fmt.Sprintf("%.2f", req.Price) // Update harga
		updateData.Xpoin = calculateXpoin(req.Price)      // Hitung ulang Xpoin HANYA jika harga diupdate
		needsUpdate = true
	} else {
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

// --- Partner Financial Transaction History Service ---

// GetFinancialTransactionHistory menggabungkan riwayat withdraw, topup, convert, transfer
func (s *PartnerService) GetFinancialTransactionHistory(partnerIDStr string) ([]PartnerTransactionHistoryItem, error) {
	partnerID, err := strconv.Atoi(partnerIDStr)
	if err != nil { return nil, errors.New("ID partner tidak valid") }

	allTransactions := make([]PartnerTransactionHistoryItem, 0)

	// Ambil data dari masing-masing tabel (handle error per jenis)
	withdrawHistory, errW := s.repo.GetWithdrawHistoryForPartner(partnerID)
	if errW != nil { log.Printf("Error getting withdraw history for partner %d: %v", partnerID, errW) }
	allTransactions = append(allTransactions, withdrawHistory...)

	topupHistory, errT := s.repo.GetTopupHistoryForPartner(partnerID)
	if errT != nil { log.Printf("Error getting topup history for partner %d: %v", partnerID, errT) }
	allTransactions = append(allTransactions, topupHistory...)

	convertHistory, errC := s.repo.GetConversionHistoryForPartner(partnerID)
	if errC != nil { log.Printf("Error getting conversion history for partner %d: %v", partnerID, errC) }
	allTransactions = append(allTransactions, convertHistory...)

	transferHistory, errTr := s.repo.GetTransferHistoryForPartner(partnerID)
	if errTr != nil { log.Printf("Error getting transfer history for partner %d: %v", partnerID, errTr) }
	allTransactions = append(allTransactions, transferHistory...)


	// Urutkan semua transaksi berdasarkan waktu (terbaru dulu)
	sort.SliceStable(allTransactions, func(i, j int) bool {
		return allTransactions[i].Timestamp.After(allTransactions[j].Timestamp)
	})

	return allTransactions, nil // Tidak return error utama jika salah satu gagal
}

// --- Partner Deposit History Service ---

// GetDepositHistory mengambil riwayat setoran sampah partner
func (s *PartnerService) GetDepositHistory(partnerIDStr string) ([]DepositHistoryHeader, error) {
	partnerID, err := strconv.Atoi(partnerIDStr); if err != nil { return nil, errors.New("ID partner tidak valid") }

	history, err := s.repo.GetDepositHistoryByPartnerID(partnerID)
	if err != nil { return nil, errors.New("gagal mengambil riwayat deposit") }

	// Repo sudah handle jika kosong (return array kosong)
	return history, nil
}

// DeleteAccount menghapus akun partner
func (s *PartnerService) DeleteAccount(partnerIDStr string) error {
	partnerID, err := strconv.Atoi(partnerIDStr)
	if err != nil { return errors.New("ID partner tidak valid") }

	err = s.repo.DeletePartnerByID(partnerID)
	if err != nil {
		 if err == sql.ErrNoRows { return errors.New("partner tidak ditemukan") }
		return err // Error teknis repo
	}
	return nil // Sukses
}

// --- Partner Wallet Service Method ---

// GetPartnerWallet mengambil data wallet partner (membuat jika belum ada)
func (s *PartnerService) GetPartnerWallet(partnerIDStr string) (*PartnerWallet, error) {
	partnerID, err := strconv.Atoi(partnerIDStr)
	if err != nil { return nil, errors.New("ID partner tidak valid") }

	wallet, err := s.repo.FindOrCreateWalletByPartnerID(partnerID)
	if err != nil {
		return nil, errors.New("gagal mengambil atau membuat wallet partner")
	}
	return wallet, nil
}

// --- Partner Statistics Service Method ---

// GetPartnerStatistics mengambil data statistik partner (membuat jika belum ada)
func (s *PartnerService) GetPartnerStatistics(partnerIDStr string) (*PartnerStatistic, error) {
	partnerID, err := strconv.Atoi(partnerIDStr)
	if err != nil { return nil, errors.New("ID partner tidak valid") }

	stats, err := s.repo.FindOrCreateStatisticsByPartnerID(partnerID)
	if err != nil {
		return nil, errors.New("gagal mengambil atau membuat statistik partner")
	}
	return stats, nil
}

// --- Partner Withdraw Service Method ---

// RequestPartnerWithdrawal memproses permintaan penarikan saldo partner
func (s *PartnerService) RequestPartnerWithdrawal(partnerIDStr string, req PartnerWithdrawRequest) (string, error) {
	partnerID, err := strconv.Atoi(partnerIDStr); if err != nil { return "", errors.New("ID partner tidak valid") }

	// 1. Validasi Input Dasar
	if req.Amount < minWithdrawalAmount {
		return "", fmt.Errorf("minimal penarikan adalah Rp %.0f", minWithdrawalAmount)
	}
	// TODO: Validasi Payment Method ID
	// TODO: Validasi Account Number (mungkin berdasarkan Payment Method)

	// 2. Hitung Total dan Cek Saldo
	totalDeduction := req.Amount + withdrawalFee
	currentBalance, err := s.repo.GetPartnerCurrentBalanceByID(partnerID)
	if err != nil {
		return "", fmt.Errorf("gagal memeriksa saldo partner: %w", err)
	}

	if currentBalance < totalDeduction {
		return "", errors.New("saldo partner tidak mencukupi untuk penarikan dan biaya admin")
	}

	// 3. Eksekusi Transaksi Database
	orderID, err := s.repo.ExecutePartnerWithdrawTransaction(partnerID, totalDeduction, withdrawalFee, req.PaymentMethodID, req.AccountNumber)
	if err != nil {
		return "", fmt.Errorf("gagal memproses penarikan partner: %w", err)
	}

	// 4. (Nanti di sini) Panggil API Midtrans Disbursement
	// log.Printf("TODO: Call Midtrans Disbursement API for Partner Withdraw Order ID: %s", orderID)

	return orderID, nil
}

// --- Partner Top Up Service Method ---

// RequestPartnerTopup memproses permintaan top up saldo partner (simulasi)
func (s *PartnerService) RequestPartnerTopup(partnerIDStr string, req PartnerTopupRequest) (string, error) {
	partnerID, err := strconv.Atoi(partnerIDStr); if err != nil { return "", errors.New("ID partner tidak valid") }

	// 1. Validasi Input Dasar
	if req.Amount <= 0 {
		return "", errors.New("jumlah top up harus lebih besar dari 0")
	}
	if req.Amount < minTopupAmount {
		return "", fmt.Errorf("minimal top up adalah Rp %.0f", minTopupAmount)
	}
	// TODO: Validasi Payment Method ID

	// 2. Pastikan wallet partner ada (fungsi ini otomatis membuat jika belum ada)
	_, err = s.repo.FindOrCreateWalletByPartnerID(partnerID)
	if err != nil {
		return "", fmt.Errorf("gagal memeriksa/membuat wallet partner: %w", err)
	}

	// 3. Eksekusi Transaksi Database
	orderID, err := s.repo.ExecutePartnerTopupTransaction(partnerID, req.Amount, req.PaymentMethodID)
	if err != nil {
		return "", fmt.Errorf("gagal memproses top up partner: %w", err)
	}

	// 4. (Nanti di sini) Generate Midtrans Snap Token/URL
	// log.Printf("TODO: Generate Midtrans payment details for Partner Topup Order ID: %s", orderID)

	return orderID, nil
}

// --- Partner Transfer Service Method ---

func (s *PartnerService) TransferXpoin(senderPartnerIDStr string, req PartnerTransferRequest) (string, error) {
    senderPartnerID, err := strconv.Atoi(senderPartnerIDStr)
    if err != nil { return "", errors.New("ID pengirim tidak valid") }

    // 1. Validasi Input Dasar
    if req.Amount <= 0 { return "", errors.New("jumlah transfer harus positif") }

    // 2. Cari Penerima (Cek Partner dulu, baru User)
    var recipientUserID *int
    var recipientPartnerID *int

    // Cek di tabel partners
    recipientPartner, errP := s.repo.FindPartnerByEmail(req.RecipientEmail)
    if errP != nil && errP != sql.ErrNoRows {
        return "", errors.New("gagal mencari partner penerima")
    }

    if recipientPartner != nil {
        // Ditemukan sebagai partner
        if senderPartnerID == recipientPartner.ID { // Cek transfer ke diri sendiri
            return "", errors.New("tidak bisa transfer ke diri sendiri")
        }
        // Pastikan wallet penerima ada
         _, errWallet := s.repo.FindOrCreateWalletByPartnerID(recipientPartner.ID); if errWallet != nil {
             return "", fmt.Errorf("gagal memeriksa/membuat wallet partner penerima: %w", errWallet)
         }
        id := recipientPartner.ID // Copy id ke var baru agar bisa diambil addressnya
        recipientPartnerID = &id
    } else {
        // Jika tidak ketemu di partner, cek di tabel users
        recipUserID, errU := s.userRepo.FindUserIDByEmail(req.RecipientEmail) // Panggil userRepo
        if errU != nil && errU != sql.ErrNoRows {
            return "", errors.New("gagal mencari user penerima")
        }
        if recipUserID == 0 { // Jika tidak ketemu di user juga
            return "", errors.New("email penerima tidak ditemukan (baik user maupun partner)")
        }
        // Ditemukan sebagai user
         // Pastikan wallet user penerima ada
         _, errWallet := s.userRepo.FindOrCreateWalletByUserID(recipUserID); if errWallet != nil {
             return "", fmt.Errorf("gagal memeriksa/membuat wallet user penerima: %w", errWallet)
         }
        recipientUserID = &recipUserID // Ambil addressnya
    }


    // 3. Pastikan wallet pengirim ada
     _, err = s.repo.FindOrCreateWalletByPartnerID(senderPartnerID); if err != nil {
         return "", fmt.Errorf("gagal memeriksa/membuat wallet pengirim: %w", err)
     }


    // 4. Eksekusi Transaksi Database
    orderID, err := s.repo.ExecutePartnerTransferTransaction(senderPartnerID, req.Amount, recipientUserID, recipientPartnerID, req.RecipientEmail)
    if err != nil {
        // Error spesifik (poin tidak cukup, dll) sudah ditangani di repo
        return "", fmt.Errorf("gagal memproses transfer: %w", err)
    }

    return orderID, nil
}

// --- Partner Conversion Service Methods ---

func (s *PartnerService) ConvertXpToRp(partnerIDStr string, req PartnerConversionRequest) (*PartnerWallet, error) {
	partnerID, err := strconv.Atoi(partnerIDStr); if err != nil { return nil, errors.New("ID partner tidak valid") }

	amountXp := int(req.Amount)
	if float64(amountXp) != req.Amount || amountXp <= 0 {
		return nil, errors.New("jumlah Xpoin harus berupa angka bulat positif")
	}

	amountRp := float64(amountXp) * conversionRateXpToRp

	_, err = s.repo.FindOrCreateWalletByPartnerID(partnerID) // Pastikan wallet ada
	if err != nil { return nil, fmt.Errorf("gagal memeriksa/membuat wallet: %w", err) }

	updatedWallet, err := s.repo.ExecutePartnerConversionTransaction(
		partnerID, -amountXp, amountRp,
		"xp_to_rp", amountXp, amountRp, conversionRateXpToRp,
	)
	if err != nil { return nil, err }
	return updatedWallet, nil
}

func (s *PartnerService) ConvertRpToXp(partnerIDStr string, req PartnerConversionRequest) (*PartnerWallet, error) {
	partnerID, err := strconv.Atoi(partnerIDStr); if err != nil { return nil, errors.New("ID partner tidak valid") }

	amountRp := req.Amount
	if amountRp <= 0 { return nil, errors.New("jumlah Rupiah harus positif") }

	amountXp := int(math.Floor(amountRp / conversionRateXpToRp))
	if amountXp <= 0 { return nil, errors.New("jumlah Rupiah terlalu kecil untuk dikonversi") }

	actualAmountRpUsed := float64(amountXp) * conversionRateXpToRp

	_, err = s.repo.FindOrCreateWalletByPartnerID(partnerID) // Pastikan wallet ada
	if err != nil { return nil, fmt.Errorf("gagal memeriksa/membuat wallet: %w", err) }

	updatedWallet, err := s.repo.ExecutePartnerConversionTransaction(
		partnerID, amountXp, -actualAmountRpUsed,
		"rp_to_xp", amountXp, actualAmountRpUsed, conversionRateXpToRp,
	)
	if err != nil { return nil, err }
	return updatedWallet, nil
}

// --- Verify QR Token Service Method ---

// VerifyDepositQrToken memvalidasi token QR dan mengembalikan data user
func (s *PartnerService) VerifyDepositQrToken(req VerifyQrTokenRequest) (*VerifyQrTokenResponse, error) {
	// 1. Validasi token menggunakan TokenStore
	userID, err := s.tokenStore.ValidateToken(req.Token)
	if err != nil {
		// Error bisa "token tidak ditemukan" atau "token sudah kedaluwarsa"
		return nil, err
	}

	// 2. Jika token valid, ambil data user dari UserRepository
	userData, err := s.userRepo.FindByID(userID) // Gunakan fungsi FindByID yang sudah ada di userRepo
	if err != nil {
		log.Printf("Error fetching user data after QR token validation for user ID %d: %v", userID, err)
		return nil, errors.New("gagal mengambil data pengguna terkait token")
	}
	if userData == nil {
		return nil, errors.New("pengguna terkait token tidak ditemukan")
	}

	// 3. Siapkan respons
	response := &VerifyQrTokenResponse{
		UserID:   userData.ID,
		Fullname: userData.Fullname,
		Email:    userData.Email,
	}

	return response, nil
}

// --- Check User Service Method ---

// CheckUserByEmail mencari user berdasarkan email
func (s *PartnerService) CheckUserByEmail(req CheckUserRequest) (*CheckUserResponse, error) {
    // Panggil userRepo yang sudah di-inject
    userData, err := s.userRepo.FindByEmail(req.Email)
    if err != nil {
        // Hanya log error teknis, jangan kirim detailnya
        log.Printf("Error checking user by email %s: %v", req.Email, err)
        return nil, errors.New("gagal memeriksa pengguna")
    }
    if userData == nil {
        return nil, errors.New("pengguna dengan email tersebut tidak ditemukan")
    }

    // Siapkan respons
    response := &CheckUserResponse{
        UserID:   userData.ID,
        Fullname: userData.Fullname,
        Email:    userData.Email,
    }
    return response, nil
}