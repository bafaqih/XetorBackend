package main

import (
	"log"
	"xetor.id/backend/internal/database"
	"xetor.id/backend/internal/domain/admin"
	"xetor.id/backend/internal/domain/user"
	"xetor.id/backend/internal/repository"
	"xetor.id/backend/internal/server"
	"xetor.id/backend/internal/config"
	"xetor.id/backend/internal/domain/midtrans"
	"xetor.id/backend/internal/temporary_token"
	"xetor.id/backend/internal/domain/partner"
)

func main() {
	config.LoadConfig() 
	database.ConnectDB()
	db := database.DB

	// Inisialisasi TokenStore untuk token sementara
	tokenStore := temporary_token.NewTokenStore()

	// Komponen Admin
	adminRepo := repository.NewAdminRepository(db)
	adminService := admin.NewAdminService(adminRepo)
	adminHandler := admin.NewAdminHandler(adminService)

	// Komponen User
	userRepo := repository.NewUserRepository(db)
	userService := user.NewService(userRepo, tokenStore)
	userHandler := user.NewHandler(userService)

	// Komponen Partner
	partnerRepo := repository.NewPartnerRepository(db)
	partnerService := partner.NewPartnerService(partnerRepo, userRepo, tokenStore, adminRepo)
	partnerHandler := partner.NewPartnerHandler(partnerService)

	// Komponen Midtrans
	midtransService := midtrans.NewMidtransService(userRepo)
	midtransHandler := midtrans.NewMidtransHandler(midtransService)

	router := server.NewRouter(userHandler, adminHandler, midtransHandler, partnerHandler)
	err := router.Run(":8080")
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}