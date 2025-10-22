// internal/domain/user/handler.go
package user

import (
	"net/http"
	"strconv"
	"database/sql"

	"github.com/gin-gonic/gin"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email dan password wajib diisi"})
		return
	}

	// Panggil service untuk validasi dan dapatkan data user
	user, err := h.service.ValidateLogin(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Jika validasi berhasil, buat token JWT
	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
		return
	}

	// Kirim respons yang berisi TOKEN dan data USER
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":    user.ID,
			"name":  user.Fullname,
			"email": user.Email,
		},
	})
}

// GetProfile menangani request untuk mengambil profil user yang sedang login
func (h *Handler) GetProfile(c *gin.Context) {
	// Ambil userID dari context yang sudah divalidasi oleh middleware
	userIDStr, exists := c.Get("userID")
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
	userIDStr, exists := c.Get("userID")
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
	userIDStr, _ := c.Get("userID") // Ambil userID dari context

	var req CreateUserAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	address, err := h.service.AddUserAddress(userIDStr.(string), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan alamat"}); return
	}
	c.JSON(http.StatusCreated, address)
}

func (h *Handler) GetUserAddresses(c *gin.Context) {
	userIDStr, _ := c.Get("userID")

	addresses, err := h.service.GetUserAddresses(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil alamat"}); return
	}
	c.JSON(http.StatusOK, addresses)
}

func (h *Handler) GetUserAddressByID(c *gin.Context) {
	userIDStr, _ := c.Get("userID")
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID alamat tidak valid"}); return
	}

	address, err := h.service.GetUserAddressByID(id, userIDStr.(string))
	if err != nil {
		// Service sudah handle error ID tidak valid
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil alamat"}); return
	}
	if address == nil { // Termasuk jika bukan milik user
		c.JSON(http.StatusNotFound, gin.H{"error": "Alamat tidak ditemukan atau bukan milik Anda"}); return
	}
	c.JSON(http.StatusOK, address)
}

func (h *Handler) UpdateUserAddress(c *gin.Context) {
	userIDStr, _ := c.Get("userID")
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID alamat tidak valid"}); return
	}

	var req UpdateUserAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	err = h.service.UpdateUserAddress(id, userIDStr.(string), req)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Alamat tidak ditemukan atau bukan milik Anda"}); return
		}
		if err.Error() == "tidak ada data untuk diupdate" {
			 c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate alamat"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Alamat berhasil diupdate"})
}

func (h *Handler) DeleteUserAddress(c *gin.Context) {
	userIDStr, _ := c.Get("userID")
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID alamat tidak valid"}); return
	}

	err = h.service.DeleteUserAddress(id, userIDStr.(string))
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Alamat tidak ditemukan atau bukan milik Anda"}); return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus alamat"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Alamat berhasil dihapus"})
}

// --- Transaction History Handler ---

// GetTransactionHistory menangani request untuk riwayat transaksi gabungan
func (h *Handler) GetTransactionHistory(c *gin.Context) {
	userIDStr, _ := c.Get("userID")

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
    userIDStr, exists := c.Get("userID")
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