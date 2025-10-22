package repository

import (
	"database/sql"
	"log"
	"strings" // Untuk update dinamis

	"fmt"
	"xetor.id/backend/internal/domain/admin"
)

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// --- Waste Type CRUD ---

func (r *AdminRepository) CreateWasteType(wt *admin.WasteType) error {
	query := `INSERT INTO waste_types (name, status) VALUES ($1, $2) RETURNING id, created_at, updated_at`
	// Jika status kosong, gunakan default 'Active'
	status := wt.Status
	if status == "" {
		status = "Active"
	}
	err := r.db.QueryRow(query, wt.Name, status).Scan(&wt.ID, &wt.CreatedAt, &wt.UpdatedAt)
	if err != nil {
		log.Printf("Error creating waste type: %v", err)
		return err
	}
	wt.Status = status // Pastikan status di struct terupdate
	log.Printf("Waste type created with ID: %d", wt.ID)
	return nil
}

func (r *AdminRepository) GetAllWasteTypes() ([]admin.WasteType, error) {
	query := `SELECT id, name, status, created_at, updated_at FROM waste_types ORDER BY name ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		log.Printf("Error getting all waste types: %v", err)
		return nil, err
	}
	defer rows.Close()

	var wasteTypes []admin.WasteType
	for rows.Next() {
		var wt admin.WasteType
		if err := rows.Scan(&wt.ID, &wt.Name, &wt.Status, &wt.CreatedAt, &wt.UpdatedAt); err != nil {
			log.Printf("Error scanning waste type row: %v", err)
			return nil, err
		}
		wasteTypes = append(wasteTypes, wt)
	}
	return wasteTypes, nil
}

func (r *AdminRepository) GetWasteTypeByID(id int) (*admin.WasteType, error) {
	query := `SELECT id, name, status, created_at, updated_at FROM waste_types WHERE id = $1`
	var wt admin.WasteType
	err := r.db.QueryRow(query, id).Scan(&wt.ID, &wt.Name, &wt.Status, &wt.CreatedAt, &wt.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Tidak ditemukan, bukan error
		}
		log.Printf("Error getting waste type by ID %d: %v", id, err)
		return nil, err
	}
	return &wt, nil
}

func (r *AdminRepository) UpdateWasteType(id int, req *admin.UpdateWasteTypeRequest) error {
	// Query update dinamis: hanya update field yang diisi
	fields := []string{}
	args := []interface{}{}
	argId := 1

	if req.Name != "" {
		fields = append(fields, fmt.Sprintf("name = $%d", argId))
		args = append(args, req.Name)
		argId++
	}
	if req.Status != "" {
		fields = append(fields, fmt.Sprintf("status = $%d", argId))
		args = append(args, req.Status)
		argId++
	}

	if len(fields) == 0 {
		return nil // Tidak ada yang diupdate
	}

	// Tambahkan ID ke argumen terakhir
	args = append(args, id)

	query := fmt.Sprintf("UPDATE waste_types SET %s, updated_at = NOW() WHERE id = $%d", strings.Join(fields, ", "), argId)

	result, err := r.db.Exec(query, args...)
	if err != nil {
		log.Printf("Error updating waste type ID %d: %v", id, err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows // ID tidak ditemukan
	}

	log.Printf("Waste type updated for ID: %d", id)
	return nil
}


func (r *AdminRepository) DeleteWasteType(id int) error {
	query := `DELETE FROM waste_types WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil {
		log.Printf("Error deleting waste type ID %d: %v", id, err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows // ID tidak ditemukan
	}

	log.Printf("Waste type deleted for ID: %d", id)
	return nil
}

// --- Waste Detail CRUD ---

func (r *AdminRepository) CreateWasteDetail(wd *admin.WasteDetail) error {
	query := `
		INSERT INTO waste_details
			(name, waste_type_id, proper_disposal_method, positive_impact, decomposition_time, xpoin, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	// Handle nullable fields
	var wasteTypeID sql.NullInt32
	if wd.WasteTypeID.Valid {
		wasteTypeID.Int32 = wd.WasteTypeID.Int32
		wasteTypeID.Valid = true
	}
	var properDisposal, positiveImpact, decompTime sql.NullString
	if wd.ProperDisposalMethod.Valid { properDisposal = wd.ProperDisposalMethod }
	if wd.PositiveImpact.Valid { positiveImpact = wd.PositiveImpact }
	if wd.DecompositionTime.Valid { decompTime = wd.DecompositionTime }

	status := wd.Status
	if status == "" {
		status = "Active"
	}

	err := r.db.QueryRow(query,
		wd.Name, wasteTypeID, properDisposal, positiveImpact, decompTime, wd.Xpoin, status,
	).Scan(&wd.ID, &wd.CreatedAt, &wd.UpdatedAt)

	if err != nil {
		log.Printf("Error creating waste detail: %v", err)
		return err
	}
	wd.Status = status // Update status in struct
	log.Printf("Waste detail created with ID: %d", wd.ID)
	return nil
}

func (r *AdminRepository) GetAllWasteDetails() ([]admin.WasteDetail, error) {
	// Query dengan LEFT JOIN untuk mendapatkan nama Waste Type
	query := `
		SELECT
			wd.id, wd.name, wd.waste_type_id, wt.name as waste_type_name,
			wd.proper_disposal_method, wd.positive_impact, wd.decomposition_time,
			wd.xpoin, wd.status, wd.created_at, wd.updated_at
		FROM waste_details wd
		LEFT JOIN waste_types wt ON wd.waste_type_id = wt.id
		ORDER BY wd.name ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		log.Printf("Error getting all waste details: %v", err)
		return nil, err
	}
	defer rows.Close()

	var details []admin.WasteDetail
	for rows.Next() {
		var wd admin.WasteDetail
		// Scan semua kolom, termasuk waste_type_name
		err := rows.Scan(
			&wd.ID, &wd.Name, &wd.WasteTypeID, &wd.WasteTypeName, &wd.ProperDisposalMethod,
			&wd.PositiveImpact, &wd.DecompositionTime, &wd.Xpoin, &wd.Status,
			&wd.CreatedAt, &wd.UpdatedAt,
		)
		if err != nil {
			log.Printf("Error scanning waste detail row: %v", err)
			return nil, err
		}
		details = append(details, wd)
	}
	return details, nil
}

func (r *AdminRepository) GetWasteDetailByID(id int) (*admin.WasteDetail, error) {
	query := `
		SELECT
			wd.id, wd.name, wd.waste_type_id, wt.name as waste_type_name,
			wd.proper_disposal_method, wd.positive_impact, wd.decomposition_time,
			wd.xpoin, wd.status, wd.created_at, wd.updated_at
		FROM waste_details wd
		LEFT JOIN waste_types wt ON wd.waste_type_id = wt.id
		WHERE wd.id = $1`

	var wd admin.WasteDetail
	err := r.db.QueryRow(query, id).Scan(
		&wd.ID, &wd.Name, &wd.WasteTypeID, &wd.WasteTypeName, &wd.ProperDisposalMethod,
		&wd.PositiveImpact, &wd.DecompositionTime, &wd.Xpoin, &wd.Status,
		&wd.CreatedAt, &wd.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		log.Printf("Error getting waste detail by ID %d: %v", id, err)
		return nil, err
	}
	return &wd, nil
}

func (r *AdminRepository) UpdateWasteDetail(id int, req *admin.UpdateWasteDetailRequest) error {
	fields := []string{}
	args := []interface{}{}
	argId := 1

	// Handle update fields, including nullable ones
	if req.Name != "" { fields = append(fields, fmt.Sprintf("name = $%d", argId)); args = append(args, req.Name); argId++ }
	if req.WasteTypeID != nil { fields = append(fields, fmt.Sprintf("waste_type_id = $%d", argId)); args = append(args, *req.WasteTypeID); argId++ } // Dereference pointer
	if req.ProperDisposalMethod != "" { fields = append(fields, fmt.Sprintf("proper_disposal_method = $%d", argId)); args = append(args, req.ProperDisposalMethod); argId++ }
	if req.PositiveImpact != "" { fields = append(fields, fmt.Sprintf("positive_impact = $%d", argId)); args = append(args, req.PositiveImpact); argId++ }
	if req.DecompositionTime != "" { fields = append(fields, fmt.Sprintf("decomposition_time = $%d", argId)); args = append(args, req.DecompositionTime); argId++ }
	if req.Xpoin != nil { fields = append(fields, fmt.Sprintf("xpoin = $%d", argId)); args = append(args, *req.Xpoin); argId++ } // Dereference pointer
	if req.Status != "" { fields = append(fields, fmt.Sprintf("status = $%d", argId)); args = append(args, req.Status); argId++ }


	if len(fields) == 0 { return nil /* No fields to update */ }
	args = append(args, id) // Add ID for WHERE clause
	query := fmt.Sprintf("UPDATE waste_details SET %s, updated_at = NOW() WHERE id = $%d", strings.Join(fields, ", "), argId)

	result, err := r.db.Exec(query, args...)
	if err != nil { log.Printf("Error updating waste detail ID %d: %v", id, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
	log.Printf("Waste detail updated for ID: %d", id)
	return nil
}

func (r *AdminRepository) DeleteWasteDetail(id int) error {
	query := `DELETE FROM waste_details WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil { log.Printf("Error deleting waste detail ID %d: %v", id, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
	log.Printf("Waste detail deleted for ID: %d", id)
	return nil
}

// --- Payment Method CRUD ---

func (r *AdminRepository) CreatePaymentMethod(pm *admin.PaymentMethod) error {
	query := `
		INSERT INTO payment_methods (name, type, logo, code, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	status := pm.Status
	if status == "" {
		status = "Active"
	}
	// Handle nullable code
	var code sql.NullString
	if pm.Code != "" {
		code.String = pm.Code
		code.Valid = true
	}

	err := r.db.QueryRow(query, pm.Name, pm.Type, pm.Logo, code, status).Scan(&pm.ID, &pm.CreatedAt, &pm.UpdatedAt)
	if err != nil {
		log.Printf("Error creating payment method: %v", err)
		return err
	}
	pm.Status = status
	log.Printf("Payment method created with ID: %d", pm.ID)
	return nil
}

func (r *AdminRepository) GetAllPaymentMethods() ([]admin.PaymentMethod, error) {
	query := `SELECT id, name, type, logo, code, status, created_at, updated_at FROM payment_methods ORDER BY name ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		log.Printf("Error getting all payment methods: %v", err)
		return nil, err
	}
	defer rows.Close()

	var methods []admin.PaymentMethod
	for rows.Next() {
		var pm admin.PaymentMethod
		var code sql.NullString // Handle possible null code from DB
		if err := rows.Scan(&pm.ID, &pm.Name, &pm.Type, &pm.Logo, &code, &pm.Status, &pm.CreatedAt, &pm.UpdatedAt); err != nil {
			log.Printf("Error scanning payment method row: %v", err)
			return nil, err
		}
		if code.Valid {
			pm.Code = code.String
		}
		methods = append(methods, pm)
	}
	return methods, nil
}

func (r *AdminRepository) GetPaymentMethodByID(id int) (*admin.PaymentMethod, error) {
	query := `SELECT id, name, type, logo, code, status, created_at, updated_at FROM payment_methods WHERE id = $1`
	var pm admin.PaymentMethod
	var code sql.NullString
	err := r.db.QueryRow(query, id).Scan(&pm.ID, &pm.Name, &pm.Type, &pm.Logo, &code, &pm.Status, &pm.CreatedAt, &pm.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows { return nil, nil }
		log.Printf("Error getting payment method by ID %d: %v", id, err)
		return nil, err
	}
	if code.Valid {
		pm.Code = code.String
	}
	return &pm, nil
}

func (r *AdminRepository) UpdatePaymentMethod(id int, req *admin.UpdatePaymentMethodRequest) error {
	fields := []string{}
	args := []interface{}{}
	argId := 1

	if req.Name != "" { fields = append(fields, fmt.Sprintf("name = $%d", argId)); args = append(args, req.Name); argId++ }
	if req.Type != "" { fields = append(fields, fmt.Sprintf("type = $%d", argId)); args = append(args, req.Type); argId++ }
	if req.Logo != "" { fields = append(fields, fmt.Sprintf("logo = $%d", argId)); args = append(args, req.Logo); argId++ }
	// Handle nullable code update - allow setting to empty/null implicitly
	if req.Code != "" {
		fields = append(fields, fmt.Sprintf("code = $%d", argId)); args = append(args, req.Code); argId++
	} else {
		// Optional: Add logic here if you want to explicitly set code to NULL via the request
		// fields = append(fields, fmt.Sprintf("code = $%d", argId)); args = append(args, nil); argId++
	}
	if req.Status != "" { fields = append(fields, fmt.Sprintf("status = $%d", argId)); args = append(args, req.Status); argId++ }

	if len(fields) == 0 { return nil }
	args = append(args, id)
	query := fmt.Sprintf("UPDATE payment_methods SET %s, updated_at = NOW() WHERE id = $%d", strings.Join(fields, ", "), argId)

	result, err := r.db.Exec(query, args...)
	if err != nil { log.Printf("Error updating payment method ID %d: %v", id, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
	log.Printf("Payment method updated for ID: %d", id)
	return nil
}

func (r *AdminRepository) DeletePaymentMethod(id int) error {
	query := `DELETE FROM payment_methods WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil { log.Printf("Error deleting payment method ID %d: %v", id, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
	log.Printf("Payment method deleted for ID: %d", id)
	return nil
}

// --- Deposit Method CRUD ---

func (r *AdminRepository) CreateDepositMethod(dm *admin.DepositMethod) error {
	query := `INSERT INTO deposit_methods (name, status) VALUES ($1, $2) RETURNING id, created_at, updated_at`
	status := dm.Status
	if status == "" {
		status = "Active"
	}
	err := r.db.QueryRow(query, dm.Name, status).Scan(&dm.ID, &dm.CreatedAt, &dm.UpdatedAt)
	if err != nil {
		log.Printf("Error creating deposit method: %v", err)
		return err
	}
	dm.Status = status
	log.Printf("Deposit method created with ID: %d", dm.ID)
	return nil
}

func (r *AdminRepository) GetAllDepositMethods() ([]admin.DepositMethod, error) {
	query := `SELECT id, name, status, created_at, updated_at FROM deposit_methods ORDER BY name ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		log.Printf("Error getting all deposit methods: %v", err)
		return nil, err
	}
	defer rows.Close()

	var methods []admin.DepositMethod
	for rows.Next() {
		var dm admin.DepositMethod
		if err := rows.Scan(&dm.ID, &dm.Name, &dm.Status, &dm.CreatedAt, &dm.UpdatedAt); err != nil {
			log.Printf("Error scanning deposit method row: %v", err)
			return nil, err
		}
		methods = append(methods, dm)
	}
	return methods, nil
}

func (r *AdminRepository) GetDepositMethodByID(id int) (*admin.DepositMethod, error) {
	query := `SELECT id, name, status, created_at, updated_at FROM deposit_methods WHERE id = $1`
	var dm admin.DepositMethod
	err := r.db.QueryRow(query, id).Scan(&dm.ID, &dm.Name, &dm.Status, &dm.CreatedAt, &dm.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows { return nil, nil }
		log.Printf("Error getting deposit method by ID %d: %v", id, err)
		return nil, err
	}
	return &dm, nil
}

func (r *AdminRepository) UpdateDepositMethod(id int, req *admin.UpdateDepositMethodRequest) error {
	fields := []string{}
	args := []interface{}{}
	argId := 1

	if req.Name != "" { fields = append(fields, fmt.Sprintf("name = $%d", argId)); args = append(args, req.Name); argId++ }
	if req.Status != "" { fields = append(fields, fmt.Sprintf("status = $%d", argId)); args = append(args, req.Status); argId++ }

	if len(fields) == 0 { return nil }
	args = append(args, id)
	query := fmt.Sprintf("UPDATE deposit_methods SET %s, updated_at = NOW() WHERE id = $%d", strings.Join(fields, ", "), argId)

	result, err := r.db.Exec(query, args...)
	if err != nil { log.Printf("Error updating deposit method ID %d: %v", id, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
	log.Printf("Deposit method updated for ID: %d", id)
	return nil
}

func (r *AdminRepository) DeleteDepositMethod(id int) error {
	query := `DELETE FROM deposit_methods WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil { log.Printf("Error deleting deposit method ID %d: %v", id, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
	log.Printf("Deposit method deleted for ID: %d", id)
	return nil
}

// --- Promotion Banner CRUD ---

func (r *AdminRepository) CreatePromotionBanner(pb *admin.PromotionBanner) error {
	query := `
		INSERT INTO promotion_banners (name, image, link, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	status := pb.Status
	if status == "" {
		status = "Active"
	}
	// Handle nullable link
	var link sql.NullString
	if pb.Link.Valid {
		link.String = pb.Link.String
		link.Valid = true
	}

	err := r.db.QueryRow(query, pb.Name, pb.Image, link, status).Scan(&pb.ID, &pb.CreatedAt, &pb.UpdatedAt)
	if err != nil {
		log.Printf("Error creating promotion banner: %v", err)
		return err
	}
	pb.Status = status
	log.Printf("Promotion banner created with ID: %d", pb.ID)
	return nil
}

func (r *AdminRepository) GetAllPromotionBanners() ([]admin.PromotionBanner, error) {
	query := `SELECT id, name, image, link, status, created_at, updated_at FROM promotion_banners ORDER BY created_at DESC` // Urutkan berdasarkan terbaru
	rows, err := r.db.Query(query)
	if err != nil {
		log.Printf("Error getting all promotion banners: %v", err)
		return nil, err
	}
	defer rows.Close()

	var banners []admin.PromotionBanner
	for rows.Next() {
		var pb admin.PromotionBanner
		if err := rows.Scan(&pb.ID, &pb.Name, &pb.Image, &pb.Link, &pb.Status, &pb.CreatedAt, &pb.UpdatedAt); err != nil {
			log.Printf("Error scanning promotion banner row: %v", err)
			return nil, err
		}
		banners = append(banners, pb)
	}
	return banners, nil
}

func (r *AdminRepository) GetPromotionBannerByID(id int) (*admin.PromotionBanner, error) {
	query := `SELECT id, name, image, link, status, created_at, updated_at FROM promotion_banners WHERE id = $1`
	var pb admin.PromotionBanner
	err := r.db.QueryRow(query, id).Scan(&pb.ID, &pb.Name, &pb.Image, &pb.Link, &pb.Status, &pb.CreatedAt, &pb.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows { return nil, nil }
		log.Printf("Error getting promotion banner by ID %d: %v", id, err)
		return nil, err
	}
	return &pb, nil
}

func (r *AdminRepository) UpdatePromotionBanner(id int, req *admin.UpdatePromotionBannerRequest) error {
	fields := []string{}
	args := []interface{}{}
	argId := 1

	if req.Name != "" { fields = append(fields, fmt.Sprintf("name = $%d", argId)); args = append(args, req.Name); argId++ }
	if req.Image != "" { fields = append(fields, fmt.Sprintf("image = $%d", argId)); args = append(args, req.Image); argId++ }
	if req.Link != "" { // Allow updating link (can be set to empty or a value)
		fields = append(fields, fmt.Sprintf("link = $%d", argId)); args = append(args, sql.NullString{String: req.Link, Valid: req.Link != ""}); argId++
	} else {
		// Optional: Logic to explicitly set link to NULL if req.Link is empty and intended
		// fields = append(fields, fmt.Sprintf("link = $%d", argId)); args = append(args, nil); argId++
	}
	if req.Status != "" { fields = append(fields, fmt.Sprintf("status = $%d", argId)); args = append(args, req.Status); argId++ }

	if len(fields) == 0 { return nil }
	args = append(args, id)
	query := fmt.Sprintf("UPDATE promotion_banners SET %s, updated_at = NOW() WHERE id = $%d", strings.Join(fields, ", "), argId)

	result, err := r.db.Exec(query, args...)
	if err != nil { log.Printf("Error updating promotion banner ID %d: %v", id, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
	log.Printf("Promotion banner updated for ID: %d", id)
	return nil
}

func (r *AdminRepository) DeletePromotionBanner(id int) error {
	query := `DELETE FROM promotion_banners WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil { log.Printf("Error deleting promotion banner ID %d: %v", id, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
	log.Printf("Promotion banner deleted for ID: %d", id)
	return nil
}

// --- About Xetor CRUD ---

func (r *AdminRepository) CreateAboutXetor(ax *admin.AboutXetor) error {
	query := `INSERT INTO about_xetor (title, content) VALUES ($1, $2) RETURNING id, created_at, updated_at`
	err := r.db.QueryRow(query, ax.Title, ax.Content).Scan(&ax.ID, &ax.CreatedAt, &ax.UpdatedAt)
	if err != nil {
		log.Printf("Error creating about xetor entry: %v", err)
		return err
	}
	log.Printf("About xetor entry created with ID: %d", ax.ID)
	return nil
}

func (r *AdminRepository) GetAllAboutXetor() ([]admin.AboutXetor, error) {
	query := `SELECT id, title, content, created_at, updated_at FROM about_xetor ORDER BY title ASC`
	rows, err := r.db.Query(query)
	if err != nil {
		log.Printf("Error getting all about xetor entries: %v", err)
		return nil, err
	}
	defer rows.Close()

	var entries []admin.AboutXetor
	for rows.Next() {
		var ax admin.AboutXetor
		if err := rows.Scan(&ax.ID, &ax.Title, &ax.Content, &ax.CreatedAt, &ax.UpdatedAt); err != nil {
			log.Printf("Error scanning about xetor row: %v", err)
			return nil, err
		}
		entries = append(entries, ax)
	}
	return entries, nil
}

func (r *AdminRepository) GetAboutXetorByID(id int) (*admin.AboutXetor, error) {
	query := `SELECT id, title, content, created_at, updated_at FROM about_xetor WHERE id = $1`
	var ax admin.AboutXetor
	err := r.db.QueryRow(query, id).Scan(&ax.ID, &ax.Title, &ax.Content, &ax.CreatedAt, &ax.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows { return nil, nil }
		log.Printf("Error getting about xetor by ID %d: %v", id, err)
		return nil, err
	}
	return &ax, nil
}

// GetAboutXetorByTitle - Fungsi tambahan yang mungkin berguna
func (r *AdminRepository) GetAboutXetorByTitle(title string) (*admin.AboutXetor, error) {
	query := `SELECT id, title, content, created_at, updated_at FROM about_xetor WHERE title = $1`
	var ax admin.AboutXetor
	err := r.db.QueryRow(query, title).Scan(&ax.ID, &ax.Title, &ax.Content, &ax.CreatedAt, &ax.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows { return nil, nil }
		log.Printf("Error getting about xetor by title %s: %v", title, err)
		return nil, err
	}
	return &ax, nil
}


func (r *AdminRepository) UpdateAboutXetor(id int, req *admin.UpdateAboutXetorRequest) error {
	fields := []string{}
	args := []interface{}{}
	argId := 1

	if req.Title != "" { fields = append(fields, fmt.Sprintf("title = $%d", argId)); args = append(args, req.Title); argId++ }
	if req.Content != "" { fields = append(fields, fmt.Sprintf("content = $%d", argId)); args = append(args, req.Content); argId++ }

	if len(fields) == 0 { return nil }
	args = append(args, id)
	query := fmt.Sprintf("UPDATE about_xetor SET %s, updated_at = NOW() WHERE id = $%d", strings.Join(fields, ", "), argId)

	result, err := r.db.Exec(query, args...)
	if err != nil { log.Printf("Error updating about xetor ID %d: %v", id, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
	log.Printf("About xetor entry updated for ID: %d", id)
	return nil
}

func (r *AdminRepository) DeleteAboutXetor(id int) error {
	query := `DELETE FROM about_xetor WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil { log.Printf("Error deleting about xetor ID %d: %v", id, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
	log.Printf("About xetor entry deleted for ID: %d", id)
	return nil
}

// --- Xetor Partner (Admin Approval) CRUD ---

// CreateXetorPartner - Mungkin tidak sering dipakai admin, lebih sering otomatis
func (r *AdminRepository) CreateXetorPartner(xp *admin.XetorPartner) error {
	query := `INSERT INTO xetor_partners (partner_id, status) VALUES ($1, $2) RETURNING id, created_at, updated_at`
	status := xp.Status
	if status == "" {
		status = "Pending" // Default saat admin create manual
	}
	err := r.db.QueryRow(query, xp.PartnerID, status).Scan(&xp.ID, &xp.CreatedAt, &xp.UpdatedAt)
	if err != nil {
		log.Printf("Error creating xetor partner entry: %v", err)
		return err
	}
	xp.Status = status
	log.Printf("Xetor partner entry created with ID: %d for Partner ID: %d", xp.ID, xp.PartnerID)
	return nil
}

func (r *AdminRepository) GetAllXetorPartners() ([]admin.XetorPartner, error) {
	query := `
		SELECT xp.id, xp.partner_id, p.business_name as partner_business_name, xp.status, xp.created_at, xp.updated_at
		FROM xetor_partners xp
		JOIN partners p ON xp.partner_id = p.id
		ORDER BY xp.created_at DESC` // Urutkan berdasarkan terbaru
	rows, err := r.db.Query(query)
	if err != nil {
		log.Printf("Error getting all xetor partners: %v", err)
		return nil, err
	}
	defer rows.Close()

	var partners []admin.XetorPartner
	for rows.Next() {
		var xp admin.XetorPartner
		if err := rows.Scan(&xp.ID, &xp.PartnerID, &xp.PartnerBusinessName, &xp.Status, &xp.CreatedAt, &xp.UpdatedAt); err != nil {
			log.Printf("Error scanning xetor partner row: %v", err)
			return nil, err
		}
		partners = append(partners, xp)
	}
	return partners, nil
}

func (r *AdminRepository) GetXetorPartnerByID(id int) (*admin.XetorPartner, error) {
	query := `
		SELECT xp.id, xp.partner_id, p.business_name as partner_business_name, xp.status, xp.created_at, xp.updated_at
		FROM xetor_partners xp
		JOIN partners p ON xp.partner_id = p.id
		WHERE xp.id = $1`
	var xp admin.XetorPartner
	err := r.db.QueryRow(query, id).Scan(&xp.ID, &xp.PartnerID, &xp.PartnerBusinessName, &xp.Status, &xp.CreatedAt, &xp.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows { return nil, nil }
		log.Printf("Error getting xetor partner by ID %d: %v", id, err)
		return nil, err
	}
	return &xp, nil
}

// UpdateXetorPartnerStatus - Fungsi khusus untuk admin approve/reject
func (r *AdminRepository) UpdateXetorPartnerStatus(id int, status string) error {
	query := `UPDATE xetor_partners SET status = $1, updated_at = NOW() WHERE id = $2`
	result, err := r.db.Exec(query, status, id)
	if err != nil { log.Printf("Error updating xetor partner status ID %d: %v", id, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
	log.Printf("Xetor partner status updated for ID: %d to %s", id, status)
	return nil
}


func (r *AdminRepository) DeleteXetorPartner(id int) error {
	query := `DELETE FROM xetor_partners WHERE id = $1`
	result, err := r.db.Exec(query, id)
	if err != nil { log.Printf("Error deleting xetor partner ID %d: %v", id, err); return err }
	rowsAffected, _ := result.RowsAffected(); if rowsAffected == 0 { return sql.ErrNoRows }
	log.Printf("Xetor partner entry deleted for ID: %d", id)
	return nil
}