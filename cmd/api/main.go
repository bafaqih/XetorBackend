package main

import (
	"log"
	"xetor.id/backend/internal/database"
	"xetor.id/backend/internal/domain/user"
	"xetor.id/backend/internal/repository"
	"xetor.id/backend/internal/server"
	"xetor.id/backend/internal/config"
)

func main() {
	// Muat konfigurasi dari .env
	config.LoadConfig() 
	// 1. Hubungkan ke Database
	database.ConnectDB()
	db := database.DB // Ambil koneksi database yang sudah berhasil

	// 2. Inisialisasi semua komponen (Dependency Injection manual)
	userRepo := repository.NewUserRepository(db)
	userService := user.NewService(userRepo)
	userHandler := user.NewHandler(userService)

	// 3. Buat dan jalankan server dengan router yang sudah diisi
	router := server.NewRouter(userHandler)
	err := router.Run(":8080")
	if err != nil {
		log.Fatal("Failed to start server:", err)
	}
}