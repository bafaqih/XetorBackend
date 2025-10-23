// internal/domain/midtrans/handler.go
package midtrans

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MidtransHandler struct {
	service *MidtransService
}

func NewMidtransHandler(service *MidtransService) *MidtransHandler {
	return &MidtransHandler{service: service}
}

// HandleNotification menerima notifikasi webhook dari Midtrans
func (h *MidtransHandler) HandleNotification(c *gin.Context) {
	var notification MidtransTransactionNotification

	// Bind JSON payload dari request body
	if err := c.ShouldBindJSON(&notification); err != nil {
		log.Printf("Error binding Midtrans notification JSON: %v", err)
		// Midtrans mengharapkan response 200 OK meskipun ada error parsing di sisi kita
		// agar tidak mengirim ulang notifikasi terus menerus. Cukup log errornya.
		c.JSON(http.StatusOK, gin.H{"status": "ok"}) // Bisa juga 400 jika ingin lebih strict
		return
	}

	// Proses notifikasi menggunakan service
	err := h.service.ProcessNotification(notification)
	if err != nil {
		// Jika terjadi error saat validasi atau update DB, kita tetap respon OK ke Midtrans
		// tapi error sudah di-log di service.
		log.Printf("Error processing Midtrans notification: %v", err)
		c.JSON(http.StatusOK, gin.H{"status": "ok"}) // Tetap 200 OK ke Midtrans
		return
	}

	// Jika semua sukses
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}