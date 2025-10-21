// internal/server/router.go
package server

import (
	"github.com/gin-gonic/gin"
	"xetor.id/backend/internal/domain/user" // Import package user kita
)

func NewRouter(userHandler *user.Handler) *gin.Engine {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Server Go Xetor Backend Berjalan!"})
	})

	// Grup routing untuk otentikasi
	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/register", userHandler.SignUp)
		authRoutes.POST("/login", userHandler.SignIn)
	}

	return r
}