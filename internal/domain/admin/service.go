package admin

import (
	"database/sql"
)

// Definisikan interface agar service tidak bergantung langsung pada implementasi repo
type AdminRepository interface {
	// WasteType methods
	CreateWasteType(wt *WasteType) error
	GetAllWasteTypes() ([]WasteType, error)
	GetWasteTypeByID(id int) (*WasteType, error)
	UpdateWasteType(id int, req *UpdateWasteTypeRequest) error
	DeleteWasteType(id int) error

	// WasteDetail methods
	CreateWasteDetail(wd *WasteDetail) error
	GetAllWasteDetails() ([]WasteDetail, error)
	GetWasteDetailByID(id int) (*WasteDetail, error)
	UpdateWasteDetail(id int, req *UpdateWasteDetailRequest) error
	DeleteWasteDetail(id int) error
	RecalculateAndUpdateWasteDetailXpoin(wasteDetailID int) error

	// PaymentMethod methods
	CreatePaymentMethod(pm *PaymentMethod) error
	GetAllPaymentMethods() ([]PaymentMethod, error)
	GetPaymentMethodByID(id int) (*PaymentMethod, error)
	UpdatePaymentMethod(id int, req *UpdatePaymentMethodRequest) error
	DeletePaymentMethod(id int) error

	// DepositMethod methods
	CreateDepositMethod(dm *DepositMethod) error
	GetAllDepositMethods() ([]DepositMethod, error)
	GetDepositMethodByID(id int) (*DepositMethod, error)
	UpdateDepositMethod(id int, req *UpdateDepositMethodRequest) error
	DeleteDepositMethod(id int) error

	// PromotionBanner methods
	CreatePromotionBanner(pb *PromotionBanner) error
	GetAllPromotionBanners() ([]PromotionBanner, error)
	GetPromotionBannerByID(id int) (*PromotionBanner, error)
	UpdatePromotionBanner(id int, req *UpdatePromotionBannerRequest) error
	DeletePromotionBanner(id int) error

	// AboutXetor methods
	CreateAboutXetor(ax *AboutXetor) error
	GetAllAboutXetor() ([]AboutXetor, error)
	GetAboutXetorByID(id int) (*AboutXetor, error)
	GetAboutXetorByTitle(title string) (*AboutXetor, error) // Tambahan
	UpdateAboutXetor(id int, req *UpdateAboutXetorRequest) error
	DeleteAboutXetor(id int) error

	// XetorPartner methods
	CreateXetorPartner(xp *XetorPartner) error
	GetAllXetorPartners() ([]XetorPartner, error)
	GetXetorPartnerByID(id int) (*XetorPartner, error)
	UpdateXetorPartnerStatus(id int, status string) error // Fungsi khusus update status
	DeleteXetorPartner(id int) error
}

type AdminService struct {
	repo AdminRepository
}

func NewAdminService(repo AdminRepository) *AdminService {
	return &AdminService{repo: repo}
}

// --- Waste Type Service Methods ---

func (s *AdminService) CreateWasteType(req CreateWasteTypeRequest) (*WasteType, error) {
	wt := &WasteType{
		Name:   req.Name,
		Status: req.Status, // Repo akan handle default jika kosong
	}
	err := s.repo.CreateWasteType(wt)
	if err != nil {
		return nil, err
	}
	return wt, nil
}

func (s *AdminService) GetAllWasteTypes() ([]WasteType, error) {
	return s.repo.GetAllWasteTypes()
}

func (s *AdminService) GetWasteTypeByID(id int) (*WasteType, error) {
	return s.repo.GetWasteTypeByID(id)
}

func (s *AdminService) UpdateWasteType(id int, req UpdateWasteTypeRequest) error {
	return s.repo.UpdateWasteType(id, &req)
}

func (s *AdminService) DeleteWasteType(id int) error {
	return s.repo.DeleteWasteType(id)
}

// --- Waste Detail Service Methods ---

func (s *AdminService) CreateWasteDetail(req CreateWasteDetailRequest) (*WasteDetail, error) {
	wd := &WasteDetail{
		Name:                 req.Name,
		ProperDisposalMethod: sql.NullString{String: req.ProperDisposalMethod, Valid: req.ProperDisposalMethod != ""},
		PositiveImpact:       sql.NullString{String: req.PositiveImpact, Valid: req.PositiveImpact != ""},
		DecompositionTime:    sql.NullString{String: req.DecompositionTime, Valid: req.DecompositionTime != ""},
		Xpoin:                req.Xpoin,
		Status:               req.Status,
	}
	if req.WasteTypeID != nil {
		wd.WasteTypeID = sql.NullInt32{Int32: int32(*req.WasteTypeID), Valid: true}
	}
	err := s.repo.CreateWasteDetail(wd)
	if err != nil { return nil, err }
	// Ambil data lengkap (termasuk nama type) setelah create
	return s.repo.GetWasteDetailByID(wd.ID)
}

func (s *AdminService) GetAllWasteDetails() ([]WasteDetail, error) {
	return s.repo.GetAllWasteDetails()
}

func (s *AdminService) GetWasteDetailByID(id int) (*WasteDetail, error) {
	return s.repo.GetWasteDetailByID(id)
}

func (s *AdminService) UpdateWasteDetail(id int, req UpdateWasteDetailRequest) error {
	// Di repo sudah handle update dinamis
	return s.repo.UpdateWasteDetail(id, &req)
}

func (s *AdminService) DeleteWasteDetail(id int) error {
	return s.repo.DeleteWasteDetail(id)
}

// --- Payment Method Service Methods ---

func (s *AdminService) CreatePaymentMethod(req CreatePaymentMethodRequest) (*PaymentMethod, error) {
	pm := &PaymentMethod{
		Name:   req.Name,
		Type:   req.Type,
		Logo:   req.Logo,
		Code:   req.Code,
		Status: req.Status,
	}
	err := s.repo.CreatePaymentMethod(pm)
	if err != nil { return nil, err }
	return pm, nil
}

func (s *AdminService) GetAllPaymentMethods() ([]PaymentMethod, error) {
	return s.repo.GetAllPaymentMethods()
}

func (s *AdminService) GetPaymentMethodByID(id int) (*PaymentMethod, error) {
	return s.repo.GetPaymentMethodByID(id)
}

func (s *AdminService) UpdatePaymentMethod(id int, req UpdatePaymentMethodRequest) error {
	return s.repo.UpdatePaymentMethod(id, &req)
}

func (s *AdminService) DeletePaymentMethod(id int) error {
	return s.repo.DeletePaymentMethod(id)
}

// --- Deposit Method Service Methods ---

func (s *AdminService) CreateDepositMethod(req CreateDepositMethodRequest) (*DepositMethod, error) {
	dm := &DepositMethod{
		Name:   req.Name,
		Status: req.Status,
	}
	err := s.repo.CreateDepositMethod(dm)
	if err != nil { return nil, err }
	return dm, nil
}

func (s *AdminService) GetAllDepositMethods() ([]DepositMethod, error) {
	return s.repo.GetAllDepositMethods()
}

func (s *AdminService) GetDepositMethodByID(id int) (*DepositMethod, error) {
	return s.repo.GetDepositMethodByID(id)
}

func (s *AdminService) UpdateDepositMethod(id int, req UpdateDepositMethodRequest) error {
	return s.repo.UpdateDepositMethod(id, &req)
}

func (s *AdminService) DeleteDepositMethod(id int) error {
	return s.repo.DeleteDepositMethod(id)
}

// --- Promotion Banner Service Methods ---

func (s *AdminService) CreatePromotionBanner(req CreatePromotionBannerRequest) (*PromotionBanner, error) {
	pb := &PromotionBanner{
		Name:   req.Name,
		Image:  req.Image,
		Link:   sql.NullString{String: req.Link, Valid: req.Link != ""},
		Status: req.Status,
	}
	err := s.repo.CreatePromotionBanner(pb)
	if err != nil { return nil, err }
	return pb, nil
}

func (s *AdminService) GetAllPromotionBanners() ([]PromotionBanner, error) {
	return s.repo.GetAllPromotionBanners()
}

func (s *AdminService) GetPromotionBannerByID(id int) (*PromotionBanner, error) {
	return s.repo.GetPromotionBannerByID(id)
}

func (s *AdminService) UpdatePromotionBanner(id int, req UpdatePromotionBannerRequest) error {
	return s.repo.UpdatePromotionBanner(id, &req)
}

func (s *AdminService) DeletePromotionBanner(id int) error {
	return s.repo.DeletePromotionBanner(id)
}

// --- About Xetor Service Methods ---

func (s *AdminService) CreateAboutXetor(req CreateAboutXetorRequest) (*AboutXetor, error) {
	ax := &AboutXetor{
		Title:   req.Title,
		Content: req.Content,
	}
	err := s.repo.CreateAboutXetor(ax)
	if err != nil { return nil, err }
	return ax, nil
}

func (s *AdminService) GetAllAboutXetor() ([]AboutXetor, error) {
	return s.repo.GetAllAboutXetor()
}

func (s *AdminService) GetAboutXetorByID(id int) (*AboutXetor, error) {
	return s.repo.GetAboutXetorByID(id)
}

// GetAboutXetorByTitle - Tambahan
func (s *AdminService) GetAboutXetorByTitle(title string) (*AboutXetor, error) {
	return s.repo.GetAboutXetorByTitle(title)
}

func (s *AdminService) UpdateAboutXetor(id int, req UpdateAboutXetorRequest) error {
	return s.repo.UpdateAboutXetor(id, &req)
}

func (s *AdminService) DeleteAboutXetor(id int) error {
	return s.repo.DeleteAboutXetor(id)
}

// --- Xetor Partner Service Methods ---

func (s *AdminService) CreateXetorPartner(req CreateXetorPartnerRequest) (*XetorPartner, error) {
	xp := &XetorPartner{
		PartnerID: req.PartnerID,
		Status:    req.Status,
	}
	err := s.repo.CreateXetorPartner(xp)
	if err != nil { return nil, err }
	// Ambil data lengkap (termasuk nama partner) setelah create
	return s.repo.GetXetorPartnerByID(xp.ID)
}

func (s *AdminService) GetAllXetorPartners() ([]XetorPartner, error) {
	return s.repo.GetAllXetorPartners()
}

func (s *AdminService) GetXetorPartnerByID(id int) (*XetorPartner, error) {
	return s.repo.GetXetorPartnerByID(id)
}

// UpdateXetorPartnerStatus - Hanya update status
func (s *AdminService) UpdateXetorPartnerStatus(id int, req UpdateXetorPartnerRequest) error {
	// Di sini bisa ditambahkan validasi status (misal: hanya boleh 'Approved' atau 'Rejected')
	return s.repo.UpdateXetorPartnerStatus(id, req.Status)
}

func (s *AdminService) DeleteXetorPartner(id int) error {
	return s.repo.DeleteXetorPartner(id)
}