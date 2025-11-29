// internal/domain/user/handler.go
package user

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"xetor.id/backend/internal/auth"
)

type Handler struct {
	service *Service
}

// Definisikan struct untuk request login
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// NewHandler membuat instance baru dari Handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// SignUp adalah fungsi yang akan dipanggil oleh router
func (h *Handler) SignUp(c *gin.Context) {
	var req SignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Beri pesan error yang lebih baik di sini juga
		if strings.Contains(err.Error(), "required") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Harap isi semua field yang wajib diisi."})
		} else if strings.Contains(err.Error(), "email") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format email tidak valid."})
		} else if strings.Contains(err.Error(), "min=6") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password minimal 6 karakter."})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data request tidak valid."})
		}
		return
	}

	err := h.service.RegisterUser(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat akun"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Akun berhasil dibuat"})
}

func (h *Handler) SignIn(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if strings.Contains(err.Error(), "required") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email dan password wajib diisi."})
		} else if strings.Contains(err.Error(), "email") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format email tidak valid."})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data request tidak valid."})
		}
		return
	}

	// Panggil service untuk validasi dan dapatkan data user
	user, err := h.service.ValidateLogin(req.Email, req.Password)
	if err != nil {
		// Use golang.org/x/text/cases instead of deprecated strings.Title
		caser := cases.Title(language.English)
		c.JSON(http.StatusUnauthorized, gin.H{"error": caser.String(err.Error())})
		return
	}

	// Jika validasi berhasil, buat token JWT
	token, err := auth.GenerateToken(user.ID, "user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
		return
	}

	// Kirim respons yang berisi TOKEN dan data USER
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"fullname": user.Fullname,
			"email":    user.Email,
		},
	})
}

// GetProfile menangani request untuk mengambil profil user yang sedang login
func (h *Handler) GetProfile(c *gin.Context) {
	// Ambil userID dari context yang sudah divalidasi oleh middleware
	userIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID pengguna dari token"})
		return
	}

	// Panggil service untuk mendapatkan data profil
	userProfile, err := h.service.GetProfile(userIDStr.(string)) // Konversi ke string
	if err != nil {
		// Service akan mengembalikan error jika user tidak ditemukan atau ID tidak valid
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, userProfile)
}

// ChangePassword menangani request untuk mengubah password
func (h *Handler) ChangePassword(c *gin.Context) {
	// Ambil userID dari context (sudah divalidasi middleware)
	userIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID pengguna dari token"})
		return
	}

	// Bind request body
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Panggil service untuk proses ganti password
	err := h.service.ChangePassword(userIDStr.(string), req)
	if err != nil {
		// Service akan mengembalikan pesan error yang sesuai
		// Kita bisa bedakan error (misal: 400 Bad Request vs 500 Internal Error)
		if err.Error() == "konfirmasi password baru tidak cocok" ||
			err.Error() == "password baru minimal 6 karakter" ||
			err.Error() == "password lama salah" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else if err.Error() == "pengguna tidak ditemukan" || err.Error() == "pengguna tidak ditemukan saat update" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengubah password"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password berhasil diubah"})
}

// --- User Address Handlers ---

func (h *Handler) AddUserAddress(c *gin.Context) {
	userIDStr, _ := c.Get("entityID") // Ambil userID dari context

	var req CreateUserAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	address, err := h.service.AddUserAddress(userIDStr.(string), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan alamat"})
		return
	}
	c.JSON(http.StatusCreated, address)
}

func (h *Handler) GetUserAddresses(c *gin.Context) {
	userIDStr, _ := c.Get("entityID")

	addresses, err := h.service.GetUserAddresses(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil alamat"})
		return
	}
	c.JSON(http.StatusOK, addresses)
}

func (h *Handler) GetUserAddressByID(c *gin.Context) {
	userIDStr, _ := c.Get("entityID")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID alamat tidak valid"})
		return
	}

	address, err := h.service.GetUserAddressByID(id, userIDStr.(string))
	if err != nil {
		// Service sudah handle error ID tidak valid
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil alamat"})
		return
	}
	if address == nil { // Termasuk jika bukan milik user
		c.JSON(http.StatusNotFound, gin.H{"error": "Alamat tidak ditemukan atau bukan milik Anda"})
		return
	}
	c.JSON(http.StatusOK, address)
}

func (h *Handler) UpdateUserAddress(c *gin.Context) {
	userIDStr, _ := c.Get("entityID")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID alamat tidak valid"})
		return
	}

	var req UpdateUserAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.service.UpdateUserAddress(id, userIDStr.(string), req)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Alamat tidak ditemukan atau bukan milik Anda"})
			return
		}
		if err.Error() == "tidak ada data untuk diupdate" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate alamat"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Alamat berhasil diupdate"})
}

func (h *Handler) DeleteUserAddress(c *gin.Context) {
	userIDStr, _ := c.Get("entityID")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID alamat tidak valid"})
		return
	}

	err = h.service.DeleteUserAddress(id, userIDStr.(string))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Alamat tidak ditemukan atau bukan milik Anda"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus alamat"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Alamat berhasil dihapus"})
}

// --- Transaction History Handler ---

// GetTransactionHistory menangani request untuk riwayat transaksi gabungan
func (h *Handler) GetTransactionHistory(c *gin.Context) {
	userIDStr, _ := c.Get("entityID")

	transactions, err := h.service.GetTransactionHistory(userIDStr.(string))
	if err != nil {
		// Service sudah handle error ID tidak valid
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat transaksi"})
		return
	}

	// Kembalikan array kosong jika tidak ada transaksi, bukan error
	if transactions == nil {
		transactions = []TransactionHistoryItem{}
	}

	c.JSON(http.StatusOK, transactions)
}

// DeleteAccount menangani request hapus akun
func (h *Handler) DeleteAccount(c *gin.Context) {
	userIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID pengguna dari token"})
		return
	}

	err := h.service.DeleteAccount(userIDStr.(string))
	if err != nil {
		if err.Error() == "pengguna tidak ditemukan" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus akun"})
		}
		return
	}

	// TODO: Mungkin perlu invalidate token JWT di sisi client/server? (Opsional)
	c.JSON(http.StatusOK, gin.H{"message": "Akun berhasil dihapus"})
}

// --- User Wallet Handler ---

// GetUserWallet menangani request untuk mengambil data wallet user
func (h *Handler) GetUserWallet(c *gin.Context) {
	userIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID pengguna dari token"})
		return
	}

	wallet, err := h.service.GetUserWallet(userIDStr.(string))
	if err != nil {
		// Service sudah handle error ID tidak valid atau kegagalan DB
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, wallet)
}

// --- User Statistics Handler ---

// GetUserStatistics menangani request untuk mengambil data statistik user
func (h *Handler) GetUserStatistics(c *gin.Context) {
	userIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID pengguna dari token"})
		return
	}

	stats, err := h.service.GetUserStatistics(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// --- Payment Methods Handler ---

// GetActivePaymentMethods menangani request untuk mengambil daftar payment methods aktif
func (h *Handler) GetActivePaymentMethods(c *gin.Context) {
	methods, err := h.service.GetAllActivePaymentMethods()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil metode pembayaran"})
		return
	}

	c.JSON(http.StatusOK, methods)
}

// GetActivePromotionBanners menangani request untuk mengambil daftar promotion banners aktif
func (h *Handler) GetActivePromotionBanners(c *gin.Context) {
	banners, err := h.service.GetAllActivePromotionBanners()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil banner promosi"})
		return
	}

	c.JSON(http.StatusOK, banners)
}

// --- User Withdraw Handler ---

// RequestWithdrawal menangani request penarikan saldo
func (h *Handler) RequestWithdrawal(c *gin.Context) {
	userIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID pengguna dari token"})
		return
	}

	var req WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID, err := h.service.RequestWithdrawal(userIDStr.(string), req)
	if err != nil {
		// Service akan memberikan pesan error yang sesuai (saldo tidak cukup, dll.)
		// Kita bisa bedakan error 400 (Bad Request) vs 500 (Internal)
		errMsg := err.Error()
		if strings.Contains(errMsg, "minimal penarikan") || strings.Contains(errMsg, "saldo tidak mencukupi") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses permintaan penarikan"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Permintaan penarikan sedang diproses",
		"order_id": orderID, // Kirim Order ID kembali (berguna untuk tracking)
	})
}

// --- User Top Up Handler ---

// RequestTopup menangani request top up saldo
func (h *Handler) RequestTopup(c *gin.Context) {
	userIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID pengguna dari token"})
		return
	}

	var req TopupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	topupResp, err := h.service.RequestTopup(userIDStr.(string), req)
	if err != nil {
		// Service akan memberikan pesan error yang sesuai
		errMsg := err.Error()
		if strings.Contains(errMsg, "harus lebih besar dari 0") || strings.Contains(errMsg, "minimal top up") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses permintaan top up"})
		}
		return
	}

	// Return response dengan Snap token untuk frontend
	c.JSON(http.StatusOK, gin.H{
		"message":      "Permintaan top up berhasil dibuat",
		"order_id":     topupResp.OrderID,
		"snap_token":   topupResp.SnapToken,
		"redirect_url": topupResp.RedirectURL,
	})
}

// --- User Transfer Handler ---

// TransferXpoin menangani request transfer xpoin
func (h *Handler) TransferXpoin(c *gin.Context) {
	userIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID pengguna dari token"})
		return
	}

	var req TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	orderID, err := h.service.TransferXpoin(userIDStr.(string), req)
	if err != nil {
		// Service akan memberikan pesan error yang sesuai
		errMsg := err.Error()
		if strings.Contains(errMsg, "tidak mencukupi") ||
			strings.Contains(errMsg, "tidak ditemukan") ||
			strings.Contains(errMsg, "diri sendiri") ||
			strings.Contains(errMsg, "lebih besar dari 0") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses transfer"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Transfer Xpoin berhasil",
		"order_id": orderID,
	})
}

// --- Conversion Handlers ---

func (h *Handler) ConvertXpToRp(c *gin.Context) {
	userIDStr, _ := c.Get("entityID")
	var req ConversionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedWallet, err := h.service.ConvertXpToRp(userIDStr.(string), req)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "mencukupi") || strings.Contains(errMsg, "angka bulat") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal melakukan konversi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Konversi Xpoin ke Rupiah berhasil",
		"wallet":  updatedWallet,
	})
}

func (h *Handler) ConvertRpToXp(c *gin.Context) {
	userIDStr, _ := c.Get("entityID")
	var req ConversionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updatedWallet, err := h.service.ConvertRpToXp(userIDStr.(string), req)
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "mencukupi") || strings.Contains(errMsg, "terlalu kecil") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal melakukan konversi"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Konversi Rupiah ke Xpoin berhasil",
		"wallet":  updatedWallet,
	})
}

// --- QR Token Handler ---

// GenerateDepositQrToken menangani request pembuatan token QR deposit
func (h *Handler) GenerateDepositQrToken(c *gin.Context) {
	userIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID pengguna dari token"})
		return
	}

	token, expiresAt, err := h.service.GenerateDepositQrToken(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := GenerateQrTokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	}
	c.JSON(http.StatusOK, response)
}

// --- User Profile Update Handlers ---

// UpdateProfile menangani request update profil user
func (h *Handler) UpdateProfile(c *gin.Context) {
	userIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID pengguna dari token"})
		return
	}

	var req UpdateUserProfileRequest
	// Gunakan BindJSON agar tidak error jika body kosong tapi valid JSON {}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data tidak valid: " + err.Error()})
		return
	}

	err := h.service.UpdateProfile(userIDStr.(string), req)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "pengguna tidak ditemukan" {
			c.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		} else if errMsg == "tidak ada data untuk diupdate" || strings.Contains(errMsg, "sudah digunakan") {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		} else {
			// Log error internal server
			log.Printf("Internal error updating user profile %s: %v", userIDStr.(string), err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate profil"})
		}
		return // Hentikan eksekusi jika ada error
	}

	// Hanya dieksekusi jika TIDAK ada error
	c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diupdate"})
}

// UploadProfilePhoto menangani request upload foto profil user
func (h *Handler) UploadProfilePhoto(c *gin.Context) {
	userIDStr, exists := c.Get("entityID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Gagal mendapatkan ID pengguna dari token"})
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
	newPhotoURL, err := h.service.UploadProfilePhoto(userIDStr.(string), file)
	if err != nil {
		errMsg := err.Error()
		if errMsg == "pengguna tidak ditemukan" {
			c.JSON(http.StatusNotFound, gin.H{"error": errMsg})
		} else { // Error lain (baca file, penyimpanan, db update)
			// Log error internal server
			log.Printf("Internal error uploading user photo %s: %v", userIDStr.(string), err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengunggah foto profil"})
		}
		return // Hentikan eksekusi jika ada error
	}

	// Hanya dieksekusi jika TIDAK ada error
	c.JSON(http.StatusOK, gin.H{
		"message":   "Foto profil berhasil diunggah",
		"photo_url": newPhotoURL,
	})
}

// --- Google Auth Handler ---

// GoogleAuth menangani request login/register via Google
func (h *Handler) GoogleAuth(c *gin.Context) {
	var req GoogleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id_token tidak boleh kosong"})
		return
	}

	// Panggil service untuk verifikasi dan login/register
	token, user, err := h.service.AuthenticateWithGoogle(req.IDToken)
	if err != nil {
		// Service sudah memberi pesan error yang sesuai
		log.Printf("Google Auth Error: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Kirim respons sukses (sama seperti login manual)
	c.JSON(http.StatusOK, GoogleAuthResponse{
		Token: token,
		User:  user,
	})
}
