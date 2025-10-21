package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // Underscore means we need this for its side effects (registering the driver)
)

var DB *sql.DB // Variabel global untuk menampung koneksi database

// ConnectDB akan membaca file .env dan menghubungkan ke database PostgreSQL
func ConnectDB() {
	// Muat file .env dari root folder
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Buat string koneksi (Data Source Name)
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)

	// Buka koneksi ke database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Lakukan ping untuk memastikan koneksi berhasil
	err = db.Ping()
	if err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	DB = db // Simpan koneksi yang berhasil ke variabel global
	log.Println("Database connected successfully")
}