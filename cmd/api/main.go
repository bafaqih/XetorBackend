package main

import (
	"log"

	"xetor.id/backend/internal/config"
	"xetor.id/backend/internal/database"
	"xetor.id/backend/internal/domain/admin"
	"xetor.id/backend/internal/domain/midtrans"
	"xetor.id/backend/internal/domain/partner"
	"xetor.id/backend/internal/domain/user"
	"xetor.id/backend/internal/notification"
	"xetor.id/backend/internal/repository"
	"xetor.id/backend/internal/server"
	"xetor.id/backend/internal/temporary_token"
)

func main() {
	config.LoadConfig()
	database.ConnectDB()
	db := database.DB

	// Inisialisasi TokenStore untuk token sementara
	tokenStore := temporary_token.NewTokenStore()

	// Inisialisasi NotificationService
	notifService := notification.NewNotificationService()

	// Komponen Admin
	adminRepo := repository.NewAdminRepository(db)
	adminService := admin.NewAdminService(adminRepo)
	adminHandler := admin.NewAdminHandler(adminService)

	// Komponen User
	userRepo := repository.NewUserRepository(db)
	
	// Komponen Midtrans (dibuat dulu karena UserService butuh ini)
	midtransService := midtrans.NewMidtransService(userRepo, notifService)
	midtransHandler := midtrans.NewMidtransHandler(midtransService)
	
	// UserService sekarang butuh MidtransService dan AdminRepository
	userService := user.NewService(userRepo, adminRepo, tokenStore, notifService, midtransService)
	userHandler := user.NewHandler(userService)

	// Komponen Partner
	partnerRepo := repository.NewPartnerRepository(db)
	partnerService := partner.NewPartnerService(partnerRepo, userRepo, tokenStore, adminRepo, notifService)
	partnerHandler := partner.NewPartnerHandler(partnerService)

	router := server.NewRouter(userHandler, adminHandler, midtransHandler, partnerHandler)
	// Gunakan port 8081 untuk Xetor agar tidak bentrok dengan web portofolio di 8080
	err := router.Run(":8081")
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
