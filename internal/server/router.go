// internal/server/router.go
package server

import (
	"github.com/gin-gonic/gin"
	"xetor.id/backend/internal/domain/admin"
	"xetor.id/backend/internal/domain/user" // Import package user kita
)

func NewRouter(userHandler *user.Handler, adminHandler *admin.AdminHandler) *gin.Engine {
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


	// Grup routing untuk user yang terotentikasi
	userRoutes := r.Group("/user")
	userRoutes.Use(AuthMiddleware())
	{
		// Rute untuk profil user
		userRoutes.GET("/profile", userHandler.GetProfile)
		userRoutes.PUT("/password", userHandler.ChangePassword)

		// Rute untuk User Addresses
		addressRoutes := userRoutes.Group("/addresses")
		{
			addressRoutes.POST("/", userHandler.AddUserAddress)
			addressRoutes.GET("/", userHandler.GetUserAddresses)
			addressRoutes.GET("/:id", userHandler.GetUserAddressByID)
			addressRoutes.PUT("/:id", userHandler.UpdateUserAddress)
			addressRoutes.DELETE("/:id", userHandler.DeleteUserAddress)
		}

		// Rute untuk riwayat transaksi user
		userRoutes.GET("/transactions", userHandler.GetTransactionHistory)
		// Rute untuk menghapus akun user
		userRoutes.DELETE("/account", userHandler.DeleteAccount)


	}


	// Grup routing untuk admin
	adminRoutes := r.Group("/admin")
	{
		// Rute untuk Waste Types
		wasteTypeRoutes := adminRoutes.Group("/waste-types")
		{
			wasteTypeRoutes.POST("/", adminHandler.CreateWasteType)
			wasteTypeRoutes.GET("/", adminHandler.GetAllWasteTypes)
			wasteTypeRoutes.GET("/:id", adminHandler.GetWasteTypeByID)
			wasteTypeRoutes.PUT("/:id", adminHandler.UpdateWasteType)
			wasteTypeRoutes.DELETE("/:id", adminHandler.DeleteWasteType)
		}
		
		// Rute untuk Waste Details
		wasteDetailRoutes := adminRoutes.Group("/waste-details")
		{
			wasteDetailRoutes.POST("/", adminHandler.CreateWasteDetail)
			wasteDetailRoutes.GET("/", adminHandler.GetAllWasteDetails)
			wasteDetailRoutes.GET("/:id", adminHandler.GetWasteDetailByID)
			wasteDetailRoutes.PUT("/:id", adminHandler.UpdateWasteDetail)
			wasteDetailRoutes.DELETE("/:id", adminHandler.DeleteWasteDetail)
		}

		// Rute untuk Payment Methods
		paymentMethodRoutes := adminRoutes.Group("/payment-methods")
		{
			paymentMethodRoutes.POST("/", adminHandler.CreatePaymentMethod)
			paymentMethodRoutes.GET("/", adminHandler.GetAllPaymentMethods)
			paymentMethodRoutes.GET("/:id", adminHandler.GetPaymentMethodByID)
			paymentMethodRoutes.PUT("/:id", adminHandler.UpdatePaymentMethod)
			paymentMethodRoutes.DELETE("/:id", adminHandler.DeletePaymentMethod)
		}

		// Rute untuk Deposit Methods
		depositMethodRoutes := adminRoutes.Group("/deposit-methods")
		{
			depositMethodRoutes.POST("/", adminHandler.CreateDepositMethod)
			depositMethodRoutes.GET("/", adminHandler.GetAllDepositMethods)
			depositMethodRoutes.GET("/:id", adminHandler.GetDepositMethodByID)
			depositMethodRoutes.PUT("/:id", adminHandler.UpdateDepositMethod)
			depositMethodRoutes.DELETE("/:id", adminHandler.DeleteDepositMethod)
		}

		// Rute untuk Promotion Banners
		bannerRoutes := adminRoutes.Group("/banners")
		{
			bannerRoutes.POST("/", adminHandler.CreatePromotionBanner)
			bannerRoutes.GET("/", adminHandler.GetAllPromotionBanners)
			bannerRoutes.GET("/:id", adminHandler.GetPromotionBannerByID)
			bannerRoutes.PUT("/:id", adminHandler.UpdatePromotionBanner)
			bannerRoutes.DELETE("/:id", adminHandler.DeletePromotionBanner)
		}

		// Rute untuk About Xetor
		aboutRoutes := adminRoutes.Group("/about-xetor")
		{
			aboutRoutes.POST("/", adminHandler.CreateAboutXetor)
			aboutRoutes.GET("/", adminHandler.GetAllAboutXetor)
			aboutRoutes.GET("/id/:id", adminHandler.GetAboutXetorByID) // Endpoint by ID
			aboutRoutes.GET("/title/:title", adminHandler.GetAboutXetorByTitle) // Endpoint by Title
			aboutRoutes.PUT("/:id", adminHandler.UpdateAboutXetor)
			aboutRoutes.DELETE("/:id", adminHandler.DeleteAboutXetor)
		}

		// Rute untuk Xetor Partners
		xetorPartnerRoutes := adminRoutes.Group("/xetor-partners")
		{
			xetorPartnerRoutes.POST("/", adminHandler.CreateXetorPartner) // Create manual oleh admin
			xetorPartnerRoutes.GET("/", adminHandler.GetAllXetorPartners)
			xetorPartnerRoutes.GET("/:id", adminHandler.GetXetorPartnerByID)
			xetorPartnerRoutes.PUT("/:id/status", adminHandler.UpdateXetorPartnerStatus) // Endpoint khusus update status
			xetorPartnerRoutes.DELETE("/:id", adminHandler.DeleteXetorPartner)
		}
	}

	return r
}