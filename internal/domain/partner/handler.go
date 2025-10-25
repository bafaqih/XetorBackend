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

	// Ambil file gambar (opsional)
	imageFile, _ := c.FormFile("image") // Abaikan error jika tidak ada file

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


	imageFile, _ := c.FormFile("image")

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