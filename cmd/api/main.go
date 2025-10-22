package main

import (
	"log"
	"xetor.id/backend/internal/database"
	"xetor.id/backend/internal/domain/admin"
	"xetor.id/backend/internal/domain/user"
	"xetor.id/backend/internal/repository"
	"xetor.id/backend/internal/server"
	"xetor.id/backend/internal/config"
)

func main() {
	config.LoadConfig() 
	database.ConnectDB()
	db := database.DB

	// Komponen User
	userRepo := repository.NewUserRepository(db)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	// Komponen Admin
	adminRepo := repository.NewAdminRepository(db)
	adminService := admin.NewAdminService(adminRepo)
	adminHandler := admin.NewAdminHandler(adminService)

	
	router := server.NewRouter(userHandler, adminHandler)
	err := router.Run(":8080")
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}