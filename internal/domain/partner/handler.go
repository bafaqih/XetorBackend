package partner

import (
	"net/http"
	"strings"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PartnerHandler struct {
	service *PartnerService
}

func NewPartnerHandler(service *PartnerService) *PartnerHandler {
	return &PartnerHandler{service: service}
}

// SignUp menangani request registrasi partner
func (h *PartnerHandler) SignUp(c *gin.Context) {
	var req PartnerSignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	partner, err := h.service.RegisterPartner(req)
	if err != nil {
		// Service sudah memberi pesan error yang sesuai (email duplikat, dll)
		if strings.Contains(err.Error(), "sudah terdaftar") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan partner"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Registrasi partner berhasil, menunggu persetujuan admin.",
		"partner": partner, // Kirim data partner yang baru dibuat (tanpa password)
	})
}

// SignIn menangani request login partner
func (h *PartnerHandler) SignIn(c *gin.Context) {
	var req PartnerLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email dan password wajib diisi"})
		return
	}

	// Panggil service yang sekarang mengembalikan 3 nilai
	token, status, err := h.service.LoginPartner(req)
	if err != nil {
		// Jika error karena kredensial tidak valid atau status tidak approved/pending
		if err.Error() == "kredensial tidak valid" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		} else { // Error lain (DB, token generation, dll)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Terjadi kesalahan saat login"})
		}
		return
	}

	// Buat respons minimalis
	response := PartnerLoginResponse{
		Token:  token,
		Status: status, // Sertakan status aktual
	}
	c.JSON(http.StatusOK, response) // Selalu 200 OK jika kredensial benar
}

// GetProfile menangani request get profil partner
func (h *PartnerHandler) GetProfile(c *gin.Context) {
	// Ambil ID dari context (sudah divalidasi middleware)
	partnerIDStr, exists := c.Get("entityID") // Gunakan "entityID" dari middleware
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID partner dari token"})
		return
	}

	partnerProfile, err := h.service.GetProfile(partnerIDStr.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()}) // Service handle not found
		return
	}

	c.JSON(http.StatusOK, partnerProfile)
}

// UpdateProfile menangani request update profil partner
func (h *PartnerHandler) UpdateProfile(c *gin.Context) {
	partnerIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID partner dari token"})
		return
	}

	var req UpdatePartnerProfileRequest
	// Gunakan BindJSON, bukan ShouldBindJSON, agar error jika body kosong tidak terjadi
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data tidak valid: " + err.Error()})
		return
	}

	err := h.service.UpdateProfile(partnerIDStr.(string), req)
	if err != nil {
		errMsg := err.Error()
        if errMsg == "partner tidak ditemukan" {
             c.JSON(http.StatusNotFound, gin.H{"error": errMsg})
        } else if errMsg == "tidak ada data untuk diupdate" || errMsg == "nomor telepon sudah digunakan" {
             c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
        } else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate profil"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diupdate"})
}

// --- Partner Photo Upload Handler ---

// UploadProfilePhoto menangani request upload foto profil
func (h *PartnerHandler) UploadProfilePhoto(c *gin.Context) {
	partnerIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID partner dari token"})
		return
	}

	// Ambil file dari form-data dengan key "photo"
	file, err := c.FormFile("photo")
	if err != nil {
		log.Printf("Error getting form file: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "File foto tidak ditemukan atau format request salah"})
		return
	}

	// Panggil service untuk upload dan update DB
	newPhotoURL, err := h.service.UploadProfilePhoto(partnerIDStr.(string), file)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "partner tidak ditemukan" {
			c.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		} else { // Error lain (baca file, cloudinary, db update)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengunggah foto profil"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Foto profil berhasil diunggah",
		"photo_url": newPhotoURL,
	})
}

// ChangePassword menangani request ganti password partner
func (h *PartnerHandler) ChangePassword(c *gin.Context) {
	partnerIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID partner dari token"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.ChangePassword(partnerIDStr.(string), req)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "konfirmasi password baru tidak cocok" ||
		   errMsg == "password baru minimal 6 karakter" ||
		   errMsg == "password lama salah" {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		} else if errMsg == "partner tidak ditemukan" || errMsg == "partner tidak ditemukan saat update" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Partner tidak ditemukan"})
		} else {
			log.Printf("Internal error changing partner password %s: %v", partnerIDStr.(string), err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengubah password"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil diubah"})
}


// --- Partner Address Handlers ---

// GetAddress menangani request get alamat usaha partner
func (h *PartnerHandler) GetAddress(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists'
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return
	}
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return
	}

	address, err := h.service.GetAddress(partnerIDStrConv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if address == nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Alamat usaha belum diatur"})
		return
	}
	c.JSON(http.StatusOK, address)
}

// UpdateAddress menangani request create/update alamat usaha partner
func (h *PartnerHandler) UpdateAddress(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists'
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return
	}
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return
	}

	var req UpdatePartnerAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedAddress, err := h.service.UpdateAddress(partnerIDStrConv, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updatedAddress)
}

// --- Partner Schedule Handlers ---

// GetSchedule menangani request get jadwal operasional (single row)
func (h *PartnerHandler) GetSchedule(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists'
	if !exists {
		log.Println("GetSchedule Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return
	}
	// Periksa dan konversi tipe 'ok'
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("GetSchedule Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return
	}

	schedules, err := h.service.GetSchedule(partnerIDStrConv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, schedules)
}

// UpdateSchedule menangani request update jadwal operasional (single row)
func (h *PartnerHandler) UpdateSchedule(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists'
	if !exists {
		log.Println("UpdateSchedule Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return
	}
	// Periksa dan konversi tipe 'ok'
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("UpdateSchedule Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return
	}

	var req UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data tidak valid: " + err.Error()})
		return
	}

	updatedSchedule, err := h.service.UpdateSchedule(partnerIDStrConv, req)
	if err != nil {
		if strings.Contains(err.Error(), "tidak valid") || strings.Contains(err.Error(), "muncul lebih dari sekali") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			log.Printf("Internal error updating partner schedule %s: %v", partnerIDStrConv, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan jadwal operasional"})
		}
		return
	}
	c.JSON(http.StatusOK, updatedSchedule)
}

// --- Partner Waste Price Handlers ---

func (h *PartnerHandler) CreateWastePrice(c *gin.Context) {
	partnerIDStr, _ := c.Get("entityID")

	// Karena ini multipart/form-data, bind form biasa, BUKAN JSON
	var req WastePriceRequest
	if err := c.ShouldBind(&req); err != nil { // Gunakan ShouldBind()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data form tidak valid: " + err.Error()})
		return
	}

	wasteDetailIDStr := c.PostForm("waste_detail_id")
	wasteDetailID, errConv := strconv.Atoi(wasteDetailIDStr)
	if errConv != nil || wasteDetailID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "waste_detail_id tidak valid atau kosong"})
		return
	}
	req.WasteDetailID = wasteDetailID // Masukkan ke struct

	imageFile, _ := c.FormFile("image")

	detail, err := h.service.CreateWastePrice(partnerIDStr.(string), req, imageFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, detail)
}

func (h *PartnerHandler) GetAllWastePrices(c *gin.Context) {
	partnerIDStr, _ := c.Get("entityID")
	details, err := h.service.GetAllWastePrices(partnerIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, details)
}

func (h *PartnerHandler) GetWastePriceByID(c *gin.Context) {
	partnerIDStr, _ := c.Get("entityID")
	detailID, err := strconv.Atoi(c.Param("detail_id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID detail tidak valid"}); return
	}

	detail, err := h.service.GetWastePriceByID(detailID, partnerIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Detail harga sampah tidak ditemukan atau bukan milik Anda"}); return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *PartnerHandler) UpdateWastePrice(c *gin.Context) {
	partnerIDStr, _ := c.Get("entityID")
	detailID, err := strconv.Atoi(c.Param("detail_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID detail tidak valid"})
		return
	}

	// Gunakan struct baru untuk binding form-data
	var req UpdateWastePriceRequest // <-- Ganti struct
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data form tidak valid: " + err.Error()})
		return
	}

	// Cek apakah ada data yang dikirim (selain file gambar)
	if req.Name == "" && req.Price <= 0 && req.Unit == "" {
		 imageFile, _ := c.FormFile("image")
		 if imageFile == nil { // Jika gambar juga tidak ada, baru error
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada data untuk diupdate"})
			return
		 }
	}

	wasteDetailIDStr := c.PostForm("waste_detail_id")
	if wasteDetailIDStr != "" { // Hanya proses jika dikirim
		wasteDetailID, errConv := strconv.Atoi(wasteDetailIDStr)
		if errConv != nil || wasteDetailID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "waste_detail_id tidak valid"})
			return
		}
		req.WasteDetailID = &wasteDetailID // Set pointer di struct
	} else {
		req.WasteDetailID = nil // Pastikan nil jika tidak dikirim
	}

	imageFile, _ := c.FormFile("image")
	if req.Name == "" && req.Price <= 0 && req.Unit == "" && req.WasteDetailID == nil && imageFile == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada data untuk diupdate"})
		return
	}

	updatedDetail, err := h.service.UpdateWastePrice(detailID, partnerIDStr.(string), req, imageFile) // <-- Kirim req baru
	if err != nil {
		errMsg := err.Error()
		if errMsg == "detail harga sampah tidak ditemukan atau bukan milik Anda" {
			 c.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate detail harga sampah: " + errMsg})
		}
		return
	}
	c.JSON(http.StatusOK, updatedDetail)
}

func (h *PartnerHandler) DeleteWastePrice(c *gin.Context) {
	partnerIDStr, _ := c.Get("entityID")
	detailID, err := strconv.Atoi(c.Param("detail_id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID detail tidak valid"}); return
	}

	err = h.service.DeleteWastePrice(detailID, partnerIDStr.(string))
	if err != nil {
		errMsg := err.Error()
		if errMsg == "detail harga sampah tidak ditemukan atau bukan milik Anda" {
			 c.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus detail harga sampah: " + errMsg})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Detail harga sampah berhasil dihapus"})
}

// --- Partner Financial Transaction History Handler ---

func (h *PartnerHandler) GetFinancialTransactionHistory(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists'
	if !exists {
		log.Println("GetFinancialTransactionHistory Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return // Hentikan eksekusi
	}
	// Periksa dan konversi tipe 'ok'
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("GetFinancialTransactionHistory Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return // Hentikan eksekusi
	}

	transactions, err := h.service.GetFinancialTransactionHistory(partnerIDStrConv)
	if err != nil {
		// Service idealnya tidak return error utama kecuali fatal
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat transaksi finansial"})
		return
	}

	// Kembalikan array kosong jika tidak ada transaksi
	if transactions == nil {
		transactions = []PartnerTransactionHistoryItem{}
	}

	c.JSON(http.StatusOK, transactions)
}

// --- Partner Deposit History Handler ---

// GetDepositHistory menangani request get riwayat deposit sampah
func (h *PartnerHandler) GetDepositHistory(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists'
	if !exists {
		log.Println("GetDepositHistory Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return // Hentikan eksekusi
	}
	// Periksa dan konversi tipe 'ok'
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("GetDepositHistory Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return // Hentikan eksekusi
	}

	history, err := h.service.GetDepositHistory(partnerIDStrConv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Service sudah handle jika tidak ada riwayat (return array kosong)
	c.JSON(http.StatusOK, history)
}

// DeleteAccount menangani request hapus akun partner
func (h *PartnerHandler) DeleteAccount(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists' dengan benar
	if !exists {
		log.Println("DeleteAccount Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return
	}
	// Periksa dan konversi tipe 'ok' dengan benar
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("DeleteAccount Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return
	}


	err := h.service.DeleteAccount(partnerIDStrConv)
	if err != nil {
		if err.Error() == "partner tidak ditemukan" {
			 c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			log.Printf("Internal error deleting partner account %s: %v", partnerIDStrConv, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus akun"})
		}
		return
	}

	// TODO: Mungkin perlu invalidate token JWT di sisi client/server? (Opsional)
	c.JSON(http.StatusOK, gin.H{"message": "Akun partner berhasil dihapus"})
}

// GetPartnerWallet menangani request untuk mengambil data wallet partner
func (h *PartnerHandler) GetPartnerWallet(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists'
	if !exists {
		log.Println("GetPartnerWallet Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return // Hentikan eksekusi
	}
	// Periksa dan konversi tipe 'ok'
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("GetPartnerWallet Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return // Hentikan eksekusi
	}

	wallet, err := h.service.GetPartnerWallet(partnerIDStrConv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

// GetPartnerStatistics menangani request untuk mengambil data statistik partner
func (h *PartnerHandler) GetPartnerStatistics(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists' dengan benar
	if !exists {
		log.Println("GetPartnerStatistics Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return // Hentikan eksekusi
	}
	// Periksa dan konversi tipe 'ok' dengan benar
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("GetPartnerStatistics Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return // Hentikan eksekusi
	}

	stats, err := h.service.GetPartnerStatistics(partnerIDStrConv)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// --- Partner Withdraw Handler ---

// RequestPartnerWithdrawal menangani request penarikan saldo partner
func (h *PartnerHandler) RequestPartnerWithdrawal(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists' dengan benar
	if !exists {
		log.Println("RequestPartnerWithdrawal Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return // Hentikan eksekusi
	}
	// Periksa dan konversi tipe 'ok' dengan benar
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("RequestPartnerWithdrawal Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return // Hentikan eksekusi
	}

	var req PartnerWithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID, err := h.service.RequestPartnerWithdrawal(partnerIDStrConv, req)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "minimal penarikan") || strings.Contains(errMsg, "saldo partner tidak mencukupi") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		} else {
			log.Printf("Internal error requesting partner withdraw %s: %v", partnerIDStrConv, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses permintaan penarikan"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Permintaan penarikan partner sedang diproses",
		"order_id": orderID,
	})
}

// --- Partner Top Up Handler ---

// RequestPartnerTopup menangani request top up saldo partner
func (h *PartnerHandler) RequestPartnerTopup(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	if !exists {
		log.Println("RequestPartnerTopup Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return
	}
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("RequestPartnerTopup Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return
	}

	var req PartnerTopupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID, err := h.service.RequestPartnerTopup(partnerIDStrConv, req)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "minimal top up") || strings.Contains(errMsg, "harus lebih besar dari 0") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		} else {
			log.Printf("Internal error requesting partner topup %s: %v", partnerIDStrConv, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses permintaan top up"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Top up partner berhasil ditambahkan (simulasi)",
		"order_id": orderID,
		// "payment_details": { ... } // Nanti
	})
}

// --- Partner Transfer Handler ---

// TransferXpoin menangani request transfer xpoin dari partner
func (h *PartnerHandler) TransferXpoin(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists' dengan benar
	if !exists {
		log.Println("TransferXpoin Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return // Hentikan eksekusi
	}
	// Periksa dan konversi tipe 'ok' dengan benar
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("TransferXpoin Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return // Hentikan eksekusi
	}


	var req PartnerTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID, err := h.service.TransferXpoin(partnerIDStrConv, req)
	if err != nil {
		errMsg := err.Error()
		// Tangani error spesifik dari service/repo
		if strings.Contains(errMsg, "tidak mencukupi") ||
		   strings.Contains(errMsg, "tidak ditemukan") ||
		   strings.Contains(errMsg, "diri sendiri") ||
		   strings.Contains(errMsg, "harus positif") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		} else {
			log.Printf("Internal error partner transfer %s: %v", partnerIDStrConv, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses transfer"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Transfer Xpoin berhasil",
		"order_id": orderID,
	})
}

// --- Partner Conversion Handlers ---

func (h *PartnerHandler) ConvertXpToRp(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists' dengan benar
	if !exists {
		log.Println("ConvertXpToRp Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return // Hentikan eksekusi
	}
	// Periksa dan konversi tipe 'ok' dengan benar
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("ConvertXpToRp Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return // Hentikan eksekusi
	}

	var req PartnerConversionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	updatedWallet, err := h.service.ConvertXpToRp(partnerIDStrConv, req)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "mencukupi") || strings.Contains(errMsg, "angka bulat") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg}); return
		}
		log.Printf("Internal error partner ConvertXpToRp %s: %v", partnerIDStrConv, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal melakukan konversi"}); return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Konversi Xpoin ke Rupiah berhasil",
		"wallet": updatedWallet,
	})
}

func (h *PartnerHandler) ConvertRpToXp(c *gin.Context) {
	partnerIDInterface, exists := c.Get("entityID")
	// Periksa 'exists' dengan benar
	if !exists {
		log.Println("ConvertRpToXp Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return // Hentikan eksekusi
	}
	// Periksa dan konversi tipe 'ok' dengan benar
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("ConvertRpToXp Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return // Hentikan eksekusi
	}

	var req PartnerConversionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	updatedWallet, err := h.service.ConvertRpToXp(partnerIDStrConv, req)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "mencukupi") || strings.Contains(errMsg, "terlalu kecil") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg}); return
		}
		log.Printf("Internal error partner ConvertRpToXp %s: %v", partnerIDStrConv, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal melakukan konversi"}); return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Konversi Rupiah ke Xpoin berhasil",
		"wallet": updatedWallet,
	})
}

// --- Verify QR Token Handler ---

// VerifyDepositQrToken menangani request validasi token QR deposit
func (h *PartnerHandler) VerifyDepositQrToken(c *gin.Context) {
	_, exists := c.Get("entityID")
	if !exists {
		log.Println("VerifyDepositQrToken Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return
	}

	var req VerifyQrTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token tidak boleh kosong"})
		return
	}

	userData, err := h.service.VerifyDepositQrToken(req)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "tidak ditemukan") || strings.Contains(errMsg, "kedaluwarsa") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg}) // Kirim pesan error spesifik
		} else {
			log.Printf("Internal error verifying QR token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi token QR"})
		}
		return
	}

	c.JSON(http.StatusOK, userData) // Kirim data user jika token valid
}

// --- Check User Handler ---

// CheckUserByEmail menangani request pengecekan user berdasarkan email
func (h *PartnerHandler) CheckUserByEmail(c *gin.Context) {
	_, exists := c.Get("entityID")
	// Periksa 'exists' dengan benar
	if !exists {
		log.Println("CheckUserByEmail Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return // Hentikan eksekusi
	}

	var req CheckUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format email tidak valid atau kosong"})
		return
	}

	userData, err := h.service.CheckUserByEmail(req)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "pengguna dengan email tersebut tidak ditemukan" {
			c.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		} else {
			log.Printf("Internal error checking user by email: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa pengguna"})
		}
		return
	}

	c.JSON(http.StatusOK, userData) // Kirim data user jika ditemukan
}

// --- Partner Deposit Creation Handler ---

// CreateDeposit menangani request pembuatan deposit baru
func (h *PartnerHandler) CreateDeposit(c *gin.Context) {
	// 1. Ambil ID Partner dari context (sudah divalidasi middleware)
	partnerIDInterface, exists := c.Get("entityID")
	if !exists {
		log.Println("CreateDeposit Error: entityID not found in context")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: ID partner tidak ada di context"})
		return
	}
	partnerIDStrConv, ok := partnerIDInterface.(string)
	if !ok {
		log.Printf("CreateDeposit Error: Invalid type for entityID: %T", partnerIDInterface)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan internal: Format ID partner tidak valid"})
		return
	}

	// 2. Bind form-data (string fields: items_json, notes)
	var req CreateDepositRequest
	req.ItemsJSON = c.PostForm("items_json")
	req.Notes = c.PostForm("notes")

	// 3. Baca & validasi field integer manual
	userIDStr := c.PostForm("user_id")
	userID, errUser := strconv.Atoi(userIDStr)
	if errUser != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id tidak valid atau kosong"})
		return
	}
	req.UserID = userID

	depositMethodIDStr := c.PostForm("deposit_method_id")
	depositMethodID, errMethod := strconv.Atoi(depositMethodIDStr)
	if errMethod != nil || depositMethodID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deposit_method_id tidak valid atau kosong"})
		return
	}
	req.DepositMethodID = depositMethodID

	// Validasi ItemsJSON tidak boleh kosong
	if req.ItemsJSON == "" {
		 c.JSON(http.StatusBadRequest, gin.H{"error": "items_json tidak boleh kosong"})
		 return
	}

	// 4. Ambil file foto (opsional)
	imageFile, _ := c.FormFile("photo") // Abaikan error jika file tidak ada

	// 5. Panggil service
	createdDepositHeader, err := h.service.CreateDeposit(partnerIDStrConv, req, imageFile)
	if err != nil {
		errMsg := err.Error()
		log.Printf("Error CreateDeposit handler: %v", err) // Log detail error
		// Bedakan error validasi (400) vs internal (500)
		if strings.Contains(errMsg, "tidak valid") || strings.Contains(errMsg, "harus positif") ||
		   strings.Contains(errMsg, "tidak mencukupi") || strings.Contains(errMsg, "tidak ditemukan") ||
		   strings.Contains(errMsg, "minimal harus ada") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses setoran sampah"})
		}
		return
	}

	// 6. Kirim response sukses
	c.JSON(http.StatusCreated, gin.H{
		"message":           "Setoran sampah berhasil dicatat",
		"deposit_header_id": createdDepositHeader.ID, // Kirim ID header deposit baru
		// Atau bisa kirim seluruh objek header jika service mengembalikannya
		// "deposit_header":    createdDepositHeader,
	})
}