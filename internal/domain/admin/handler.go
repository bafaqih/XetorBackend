package admin

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	service *AdminService
}

func NewAdminHandler(service *AdminService) *AdminHandler {
	return &AdminHandler{service: service}
}

// --- Waste Type Handlers ---

func (h *AdminHandler) CreateWasteType(c *gin.Context) {
	var req CreateWasteTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wasteType, err := h.service.CreateWasteType(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan jenis sampah"})
		return
	}

	c.JSON(http.StatusCreated, wasteType)
}

func (h *AdminHandler) GetAllWasteTypes(c *gin.Context) {
	wasteTypes, err := h.service.GetAllWasteTypes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil jenis sampah"})
		return
	}
	c.JSON(http.StatusOK, wasteTypes)
}

func (h *AdminHandler) GetWasteTypeByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	wasteType, err := h.service.GetWasteTypeByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil jenis sampah"})
		return
	}
	if wasteType == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Jenis sampah tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, wasteType)
}

func (h *AdminHandler) UpdateWasteType(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var req UpdateWasteTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Cek dulu apakah ada isinya
	if req.Name == "" && req.Status == "" {
		 c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada data untuk diupdate"})
		 return
	}


	err = h.service.UpdateWasteType(id, req)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Jenis sampah tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate jenis sampah"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Jenis sampah berhasil diupdate"})
}


func (h *AdminHandler) DeleteWasteType(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	err = h.service.DeleteWasteType(id)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Jenis sampah tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus jenis sampah"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Jenis sampah berhasil dihapus"})
}

// --- Waste Detail Handlers ---

func (h *AdminHandler) CreateWasteDetail(c *gin.Context) {
	var req CreateWasteDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wasteDetail, err := h.service.CreateWasteDetail(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan detail sampah"})
		return
	}
	c.JSON(http.StatusCreated, wasteDetail)
}

func (h *AdminHandler) GetAllWasteDetails(c *gin.Context) {
	details, err := h.service.GetAllWasteDetails()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail sampah"})
		return
	}
	c.JSON(http.StatusOK, details)
}

func (h *AdminHandler) GetWasteDetailByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	detail, err := h.service.GetWasteDetailByID(id); if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail sampah"}); return
	}
	if detail == nil { c.JSON(http.StatusNotFound, gin.H{"error": "Detail sampah tidak ditemukan"}); return }
	c.JSON(http.StatusOK, detail)
}

func (h *AdminHandler) UpdateWasteDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	var req UpdateWasteDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	if req.Name == "" && req.WasteTypeID == nil && req.ProperDisposalMethod == "" &&
	   req.PositiveImpact == "" && req.DecompositionTime == "" && req.Xpoin == nil && req.Status == "" {
		 c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada data untuk diupdate"}); return
	}

	err = h.service.UpdateWasteDetail(id, req); if err != nil {
		if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "Detail sampah tidak ditemukan"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate detail sampah"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Detail sampah berhasil diupdate"})
}

func (h *AdminHandler) DeleteWasteDetail(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	err = h.service.DeleteWasteDetail(id); if err != nil {
		if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "Detail sampah tidak ditemukan"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus detail sampah"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Detail sampah berhasil dihapus"})
}

// --- Payment Method Handlers ---

func (h *AdminHandler) CreatePaymentMethod(c *gin.Context) {
	var req CreatePaymentMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	pm, err := h.service.CreatePaymentMethod(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan metode pembayaran"}); return
	}
	c.JSON(http.StatusCreated, pm)
}

func (h *AdminHandler) GetAllPaymentMethods(c *gin.Context) {
	methods, err := h.service.GetAllPaymentMethods()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil metode pembayaran"}); return
	}
	c.JSON(http.StatusOK, methods)
}

func (h *AdminHandler) GetPaymentMethodByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	method, err := h.service.GetPaymentMethodByID(id); if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil metode pembayaran"}); return
	}
	if method == nil { c.JSON(http.StatusNotFound, gin.H{"error": "Metode pembayaran tidak ditemukan"}); return }
	c.JSON(http.StatusOK, method)
}

func (h *AdminHandler) UpdatePaymentMethod(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	var req UpdatePaymentMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	// Cek jika body kosong
	if req.Name == "" && req.Type == "" && req.Logo == "" && req.Code == "" && req.Status == "" {
		 c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada data untuk diupdate"}); return
	}

	err = h.service.UpdatePaymentMethod(id, req); if err != nil {
		if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "Metode pembayaran tidak ditemukan"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate metode pembayaran"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Metode pembayaran berhasil diupdate"})
}

func (h *AdminHandler) DeletePaymentMethod(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	err = h.service.DeletePaymentMethod(id); if err != nil {
		if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "Metode pembayaran tidak ditemukan"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus metode pembayaran"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Metode pembayaran berhasil dihapus"})
}

// --- Deposit Method Handlers ---

func (h *AdminHandler) CreateDepositMethod(c *gin.Context) {
	var req CreateDepositMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	dm, err := h.service.CreateDepositMethod(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan metode deposit"}); return
	}
	c.JSON(http.StatusCreated, dm)
}

func (h *AdminHandler) GetAllDepositMethods(c *gin.Context) {
	methods, err := h.service.GetAllDepositMethods()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil metode deposit"}); return
	}
	c.JSON(http.StatusOK, methods)
}

func (h *AdminHandler) GetDepositMethodByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	method, err := h.service.GetDepositMethodByID(id); if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil metode deposit"}); return
	}
	if method == nil { c.JSON(http.StatusNotFound, gin.H{"error": "Metode deposit tidak ditemukan"}); return }
	c.JSON(http.StatusOK, method)
}

func (h *AdminHandler) UpdateDepositMethod(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	var req UpdateDepositMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	if req.Name == "" && req.Status == "" {
		 c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada data untuk diupdate"}); return
	}

	err = h.service.UpdateDepositMethod(id, req); if err != nil {
		if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "Metode deposit tidak ditemukan"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate metode deposit"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Metode deposit berhasil diupdate"})
}

func (h *AdminHandler) DeleteDepositMethod(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	err = h.service.DeleteDepositMethod(id); if err != nil {
		if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "Metode deposit tidak ditemukan"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus metode deposit"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Metode deposit berhasil dihapus"})
}

// --- Promotion Banner Handlers ---

func (h *AdminHandler) CreatePromotionBanner(c *gin.Context) {
	var req CreatePromotionBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	banner, err := h.service.CreatePromotionBanner(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan banner promosi"}); return
	}
	c.JSON(http.StatusCreated, banner)
}

func (h *AdminHandler) GetAllPromotionBanners(c *gin.Context) {
	banners, err := h.service.GetAllPromotionBanners()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil banner promosi"}); return
	}
	c.JSON(http.StatusOK, banners)
}

func (h *AdminHandler) GetPromotionBannerByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	banner, err := h.service.GetPromotionBannerByID(id); if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil banner promosi"}); return
	}
	if banner == nil { c.JSON(http.StatusNotFound, gin.H{"error": "Banner promosi tidak ditemukan"}); return }
	c.JSON(http.StatusOK, banner)
}

func (h *AdminHandler) UpdatePromotionBanner(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	var req UpdatePromotionBannerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	// Cek jika body kosong
	if req.Name == "" && req.Image == "" && req.Link == "" && req.Status == "" {
		 c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada data untuk diupdate"}); return
	}

	err = h.service.UpdatePromotionBanner(id, req); if err != nil {
		if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "Banner promosi tidak ditemukan"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate banner promosi"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Banner promosi berhasil diupdate"})
}

func (h *AdminHandler) DeletePromotionBanner(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	err = h.service.DeletePromotionBanner(id); if err != nil {
		if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "Banner promosi tidak ditemukan"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus banner promosi"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Banner promosi berhasil dihapus"})
}

// --- About Xetor Handlers ---

func (h *AdminHandler) CreateAboutXetor(c *gin.Context) {
	var req CreateAboutXetorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	entry, err := h.service.CreateAboutXetor(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan entri About Xetor"}); return
	}
	c.JSON(http.StatusCreated, entry)
}

func (h *AdminHandler) GetAllAboutXetor(c *gin.Context) {
	entries, err := h.service.GetAllAboutXetor()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil entri About Xetor"}); return
	}
	c.JSON(http.StatusOK, entries)
}

func (h *AdminHandler) GetAboutXetorByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	entry, err := h.service.GetAboutXetorByID(id); if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil entri About Xetor"}); return
	}
	if entry == nil { c.JSON(http.StatusNotFound, gin.H{"error": "Entri About Xetor tidak ditemukan"}); return }
	c.JSON(http.StatusOK, entry)
}

// GetAboutXetorByTitle - Handler tambahan
func (h *AdminHandler) GetAboutXetorByTitle(c *gin.Context) {
	title := c.Param("title") // Ambil title dari URL path
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title tidak boleh kosong"}); return
	}
	entry, err := h.service.GetAboutXetorByTitle(title); if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil entri About Xetor"}); return
	}
	if entry == nil { c.JSON(http.StatusNotFound, gin.H{"error": "Entri About Xetor tidak ditemukan"}); return }
	c.JSON(http.StatusOK, entry)
}


func (h *AdminHandler) UpdateAboutXetor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	var req UpdateAboutXetorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	if req.Title == "" && req.Content == "" {
		 c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada data untuk diupdate"}); return
	}

	err = h.service.UpdateAboutXetor(id, req); if err != nil {
		if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "Entri About Xetor tidak ditemukan"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate entri About Xetor"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Entri About Xetor berhasil diupdate"})
}

func (h *AdminHandler) DeleteAboutXetor(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	err = h.service.DeleteAboutXetor(id); if err != nil {
		if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "Entri About Xetor tidak ditemukan"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus entri About Xetor"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Entri About Xetor berhasil dihapus"})
}

// --- Xetor Partner Handlers ---

func (h *AdminHandler) CreateXetorPartner(c *gin.Context) {
	var req CreateXetorPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	partner, err := h.service.CreateXetorPartner(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan Xetor partner"}); return
	}
	c.JSON(http.StatusCreated, partner)
}

func (h *AdminHandler) GetAllXetorPartners(c *gin.Context) {
	partners, err := h.service.GetAllXetorPartners()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil Xetor partner"}); return
	}
	c.JSON(http.StatusOK, partners)
}

func (h *AdminHandler) GetXetorPartnerByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	partner, err := h.service.GetXetorPartnerByID(id); if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil Xetor partner"}); return
	}
	if partner == nil { c.JSON(http.StatusNotFound, gin.H{"error": "Xetor partner tidak ditemukan"}); return }
	c.JSON(http.StatusOK, partner)
}

// UpdateXetorPartnerStatus - Handler khusus update status
func (h *AdminHandler) UpdateXetorPartnerStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	var req UpdateXetorPartnerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	// Validasi status sederhana (bisa diperketat)
	if req.Status != "Approved" && req.Status != "Rejected" && req.Status != "Pending" && req.Status != "Inactive" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status tidak valid"}); return
	}

	err = h.service.UpdateXetorPartnerStatus(id, req); if err != nil {
		if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "Xetor partner tidak ditemukan"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate status Xetor partner"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Status Xetor partner berhasil diupdate"})
}


func (h *AdminHandler) DeleteXetorPartner(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id")); if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"}); return
	}
	err = h.service.DeleteXetorPartner(id); if err != nil {
		if err == sql.ErrNoRows { c.JSON(http.StatusNotFound, gin.H{"error": "Xetor partner tidak ditemukan"}); return }
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus Xetor partner"}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Xetor partner berhasil dihapus"})
}