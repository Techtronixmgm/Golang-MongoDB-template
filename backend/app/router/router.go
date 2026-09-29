package router

import (
	"basic-app/auth"
	"basic-app/config"

	"basic-app/handler"
	mongorepo "basic-app/repository/mongo"
	"basic-app/services"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	mongo "go.mongodb.org/mongo-driver/v2/mongo"
)

func NewRouter(
	client *mongo.Client,
	database *mongo.Database,
	cfg config.Config,
) *gin.Engine {
	r := gin.Default()

	// -------------------------------------------------------------
	// CORS Configuration
	// -------------------------------------------------------------

	r.Use(cors.New(cors.Config{
		AllowOrigins: cfg.AllowedOrigins,

		AllowMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
		},

		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
		},

		AllowCredentials: true,
	}))

	// Custom middleware.
	// Uncomment these when you want request ID and structured logging.
	r.Use(
		gin.Recovery(),
		// middleware.RequestID(),
		// middleware.Logger(),
	)

	r.Static("/Uploads", "./Uploads")

	// ------------------------------------------------------------------
	// Dependencies
	// ------------------------------------------------------------------

	totpEncryptionService, err := services.NewTOTPEncryptionService(
		cfg.TOTPEncryptionKey,
	)
	if err != nil {
		log.Fatal(err)
	}

	userRepository := mongorepo.NewUserRepository(database)

	settingsRepository := mongorepo.NewApplicationSettingsRepository(database)
	settingsService := services.NewSettingsService(settingsRepository, cfg)
	totpService := services.NewTOTPService(cfg.TOTPIssuer)
	refreshTokenRepository := mongorepo.NewRefreshTokenRepository(database)

	authService := services.NewAuthService(
		userRepository,
		refreshTokenRepository,
		settingsService,
		totpService,
		totpEncryptionService,
		cfg,
	)

	// Initialize application settings.
	settingsCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := settingsRepository.EnsureDefaults(settingsCtx); err != nil {
		log.Printf("failed to initialize application settings: %v", err)
		panic(err)
	}

	// ------------------------------------------------------------------
	// Refresh token callback
	// ------------------------------------------------------------------

	refreshToken := func(
		ctx context.Context,
		refreshToken string,
	) (string, string, time.Time, error) {
		result, err := authService.RefreshToken(ctx, refreshToken)
		if err != nil {
			return "", "", time.Time{}, err
		}

		return result.AccessToken,
			result.RefreshToken,
			result.RefreshExpiry,
			nil
	}

	// ------------------------------------------------------------------
	// Authentication middleware
	// ------------------------------------------------------------------

	authMiddleware := auth.AuthMiddleware(
		cfg.JWTSecret,
		cfg.AuthAccessCookie,
		cfg.AuthRefreshCookie,
		cfg.CookieSecure,
		cfg.JWTExpiryHours,
		refreshToken,
	)

	optionalAuthMiddleware := auth.OptionalAuthMiddleware(
		cfg.JWTSecret,
		cfg.AuthAccessCookie,
		cfg.AuthRefreshCookie,
		cfg.CookieSecure,
		cfg.JWTExpiryHours,
		refreshToken,
	)

	// ------------------------------------------------------------------
	// Services / handlers
	// ------------------------------------------------------------------

	userService := services.NewUserService(userRepository, totpService, totpEncryptionService)
	authHandler := handler.NewAuthHandler(authService, userService, cfg)
	userHandler := handler.NewUserHandler(userService)

	pageRepository := mongorepo.NewPageRepository(database)
	pageService := services.NewPageService(pageRepository)
	pageHandler := handler.NewPageHandler(pageService)

	settingsHandler := handler.NewSettingsHandler(settingsService)

	// ------------------------------------------------------------------
	// Health / readiness
	// ------------------------------------------------------------------

	// Liveness:
	// Only confirms that the application process is running.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"status": "application is up",
		})
	})

	// Readiness:
	// Confirms that the application can currently reach MongoDB.

	r.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			2*time.Second,
		)
		defer cancel()

		if err := client.Ping(ctx, nil); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "database is up",
		})
	})

	// ------------------------------------------------------------------
	// Authentication routes
	// ------------------------------------------------------------------

	authRoutes := r.Group("/api/v1/auth")
	{
		// Public
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/login", authHandler.Login)
		authRoutes.POST("/refresh", authHandler.Refresh)
		authRoutes.POST("2fa/verify-login", authHandler.VerifyTwoFactorLogin)

		// Authenticated
		protected := authRoutes.Group("")
		protected.Use(authMiddleware)

		protected.PATCH("/password", authHandler.ChangePassword)
		protected.POST("/logout", authHandler.Logout)
		protected.PATCH("/me", userHandler.UpdateProfile)
		protected.PATCH("/me/image", userHandler.UpdateProfilePic)
		protected.GET("/me", userHandler.Me)

		// 2FA srtup
		protected.POST("/2fa/setup", authHandler.StartTwoFactorSetup)
		protected.POST("/2fa/verify-setup", authHandler.VerifyTwoFactorSetup)
		protected.POST("/2fa/disable", authHandler.DisableTwoFactor)
	}

	// ------------------------------------------------------------------
	// Customer routes
	// ------------------------------------------------------------------

	customerRoutes := r.Group("/api/v1/customers")

	customerRoutes.Use(
		authMiddleware,
		auth.RequireRoles("admin"),
	)

	customerRoutes.POST("", userHandler.CreateCustomer)
	//--------------------------------------------------------------
	customerRoutes.GET("", userHandler.ListCustomers)
	// List customers.
	//
	// Pagination:
	// GET /api/v1/customers?page=1&limit=20
	//
	// Filter by account status:
	// GET /api/v1/customers?status=true
	// GET /api/v1/customers?status=false
	//
	// Search across first name, last name, username, email,
	// alternate email, and phone:
	// GET /api/v1/customers?search=rahul
	//
	// Filters can be combined:
	// GET /api/v1/customers?page=1&limit=20&status=true&search=rahul

	//--------------------------------------------------------------
	customerRoutes.GET("/:id", userHandler.GetCustomerByID)
	customerRoutes.PATCH("/:id", userHandler.UpdateCustomer)
	customerRoutes.PATCH("/:id/status", userHandler.UpdateUserStatus)
	customerRoutes.PATCH("/:id/password", userHandler.ChangeUserPassword)

	// ------------------------------------------------------------------
	// Application settings
	// ------------------------------------------------------------------

	r.GET("/api/v1/settings", settingsHandler.GetPublicSettings)

	protectedSettings := r.Group("/api/v1/admin/settings")
	protectedSettings.Use(
		authMiddleware,
		auth.RequireRoles("admin"),
	)
	protectedSettings.PATCH("", settingsHandler.Update)
	protectedSettings.GET("", settingsHandler.GetPrivateSettings)

	// ------------------------------------------------------------------
	// Pages
	// ------------------------------------------------------------------

	api := r.Group("/api/v1")

	// Public / registered pages
	api.GET(
		"/pages/:slug",
		optionalAuthMiddleware,
		pageHandler.GetBySlug,
	)

	// Admin pages
	admin := api.Group("/admin")
	admin.Use(
		authMiddleware,
		auth.RequireRoles("admin"),
	)

	admin.POST("/pages", pageHandler.Create)
	admin.GET("/pages", pageHandler.List)
	admin.PATCH("/pages/:id", pageHandler.Update)
	admin.DELETE("/pages/:id", pageHandler.Delete)

	return r
}
