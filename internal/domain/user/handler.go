// internal/domain/user/handler.go
package user

import (
	"net/http"
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