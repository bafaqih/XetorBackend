package partner

import (
	"net/http"
	"strings"
	"log"

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