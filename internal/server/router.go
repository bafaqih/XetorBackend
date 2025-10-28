// internal/server/router.go
package server

import (
	"github.com/gin-gonic/gin"
	"xetor.id/backend/internal/domain/admin"
	"xetor.id/backend/internal/domain/user"
	"xetor.id/backend/internal/domain/midtrans"
	"xetor.id/backend/internal/domain/partner"
)

func NewRouter(userHandler *user.Handler, adminHandler *admin.AdminHandler, midtransHandler *midtrans.MidtransHandler, partnerHandler *partner.PartnerHandler) *gin.Engine {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Server Go Xetor Backend Berjalan!"})
	})

	// Grup routing untuk otentikasi
	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/register", userHandler.SignUp)
		authRoutes.POST("/login", userHandler.SignIn)
		authRoutes.POST("/google", userHandler.GoogleAuth)
	}

	partnerAuthRoutes := r.Group("/partners")
	{
		partnerAuthRoutes.POST("/register", partnerHandler.SignUp)
		partnerAuthRoutes.POST("/login", partnerHandler.SignIn)
	}


	// Grup routing untuk user yang terotentikasi
	userRoutes := r.Group("/user")
	userRoutes.Use(AuthMiddleware(), RoleCheckMiddleware("user"))
	{
		// Rute untuk profil dan pengelolaan akun
		userRoutes.GET("/profile", userHandler.GetProfile)
		userRoutes.PUT("/profile", userHandler.UpdateProfile)
		userRoutes.POST("/profile/photo", userHandler.UploadProfilePhoto)
		userRoutes.PUT("/password", userHandler.ChangePassword)
		userRoutes.GET("/transactions", userHandler.GetTransactionHistory)
		userRoutes.DELETE("/account", userHandler.DeleteAccount)
		userRoutes.GET("/wallet", userHandler.GetUserWallet)
		userRoutes.GET("/statistics", userHandler.GetUserStatistics)
		userRoutes.POST("/withdraw", userHandler.RequestWithdrawal)
		userRoutes.POST("/topup", userHandler.RequestTopup)
		userRoutes.POST("/transfer", userHandler.TransferXpoin)
		
		// Rute untuk konversi Xpoin dan Rupiah
		convertRoutes := userRoutes.Group("/convert")
		{
			convertRoutes.POST("/xp-to-rp", userHandler.ConvertXpToRp)
			convertRoutes.POST("/rp-to-xp", userHandler.ConvertRpToXp)
		}

		// Rute untuk User Addresses
		addressRoutes := userRoutes.Group("/addresses")
		{
			addressRoutes.POST("/", userHandler.AddUserAddress)
			addressRoutes.GET("/", userHandler.GetUserAddresses)
			addressRoutes.GET("/:id", userHandler.GetUserAddressByID)
			addressRoutes.PUT("/:id", userHandler.UpdateUserAddress)
			addressRoutes.DELETE("/:id", userHandler.DeleteUserAddress)
		}

		// Rute untuk Deposit via QR Code
		depositRoutes := userRoutes.Group("/deposit")
		{
			depositRoutes.POST("/generate-qr-token", userHandler.GenerateDepositQrToken)
			// Endpoint lain terkait deposit bisa ditambahkan di sini nanti
		}

	}

	// Grup routing untuk partner
	partnerRoutes := r.Group("/partner")
    partnerRoutes.Use(AuthMiddleware(), RoleCheckMiddleware("partner"))
    {
		 // Ruter untuk profil partner
        partnerRoutes.GET("/profile", partnerHandler.GetProfile)
        partnerRoutes.PUT("/profile", partnerHandler.UpdateProfile)
		partnerRoutes.POST("/profile/photo", partnerHandler.UploadProfilePhoto)
		partnerRoutes.PUT("/password", partnerHandler.ChangePassword)
		partnerRoutes.DELETE("/account", partnerHandler.DeleteAccount)
		partnerRoutes.GET("/wallet", partnerHandler.GetPartnerWallet)
		partnerRoutes.GET("/statistics", partnerHandler.GetPartnerStatistics)
		partnerRoutes.POST("/withdraw", partnerHandler.RequestPartnerWithdrawal)
		partnerRoutes.POST("/topup", partnerHandler.RequestPartnerTopup)
		partnerRoutes.POST("/transfer", partnerHandler.TransferXpoin)

		 // Ruter untuk konversi Xpoin dan Rupiah
		convertRoutes := partnerRoutes.Group("/convert")
		{
			convertRoutes.POST("/xp-to-rp", partnerHandler.ConvertXpToRp)
			convertRoutes.POST("/rp-to-xp", partnerHandler.ConvertRpToXp)
		}

		// Ruter untuk alamat partner
		partnerRoutes.GET("/address", partnerHandler.GetAddress)
		partnerRoutes.PUT("/address", partnerHandler.UpdateAddress)

		// Ruter untuk jadwal operasional partner
		partnerRoutes.GET("/schedule", partnerHandler.GetSchedule)
		partnerRoutes.PUT("/schedule", partnerHandler.UpdateSchedule)

		// Ruter untuk harga sampah
		wastePriceRoutes := partnerRoutes.Group("/waste-prices")
		{
			wastePriceRoutes.POST("/", partnerHandler.CreateWastePrice)
			wastePriceRoutes.GET("/", partnerHandler.GetAllWastePrices)
			wastePriceRoutes.GET("/:detail_id", partnerHandler.GetWastePriceByID)
			wastePriceRoutes.PUT("/:detail_id", partnerHandler.UpdateWastePrice)
			wastePriceRoutes.DELETE("/:detail_id", partnerHandler.DeleteWastePrice)
		}

		 // Ruter untuk riwayat transaksi partner
		partnerRoutes.GET("/transactions", partnerHandler.GetFinancialTransactionHistory)
		
		 // Ruter untuk riwayat deposit partner
		depositRoutes := partnerRoutes.Group("/deposit")
		{
			depositRoutes.GET("/history", partnerHandler.GetDepositHistory)
			depositRoutes.POST("/verify-qr-token", partnerHandler.VerifyDepositQrToken)
			depositRoutes.POST("/check-user", partnerHandler.CheckUserByEmail)
			depositRoutes.POST("/create", partnerHandler.CreateDeposit)
		}

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

	// Grup routing untuk Midtrans Webhook
	r.POST("/midtrans/notification", midtransHandler.HandleNotification)
	
	return r
}