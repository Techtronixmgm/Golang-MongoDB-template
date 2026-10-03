package router

import (
	"basic-app/auth"
	"basic-app/config"
	"basic-app/middleware"

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
			// "X-Request-ID",
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

	userService := services.NewUserService(userRepository, totpService, cfg)
	authHandler := handler.NewAuthHandler(authService, userService, cfg)
	userHandler := handler.NewUserHandler(userService)

	pageRepository := mongorepo.NewPageRepository(database)
	menuRepository := mongorepo.NewMenuRepository(database)
	menuService := services.NewMenuService(
		menuRepository,
		pageRepository,
		settingsService,
	)
	pageService := services.NewPageService(pageRepository)
	pageHandler := handler.NewPageHandler(pageService)
	menuHandler := handler.NewMenuHandler(menuService)
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
		// Rate limiter config

		loginLimiter := middleware.NewRateLimiter(
			5,
			time.Minute,
		)

		registerLimiter := middleware.NewRateLimiter(
			5,
			time.Minute,
		)

		refreshLimiter := middleware.NewRateLimiter(
			10,
			time.Minute,
		)

		twoFactorLimiter := middleware.NewRateLimiter(
			5,
			time.Minute,
		)

		// Public
		authRoutes.POST(
			"/register",
			registerLimiter.Middleware(middleware.RateLimitKeyByIP),
			authHandler.Register,
		)

		authRoutes.POST(
			"/login",
			loginLimiter.Middleware(middleware.RateLimitKeyByIP),
			authHandler.Login,
		)

		authRoutes.POST(
			"/refresh",
			refreshLimiter.Middleware(middleware.RateLimitKeyByIP),
			authHandler.Refresh,
		)

		authRoutes.POST(
			"/2fa/verify-login",
			twoFactorLimiter.Middleware(middleware.RateLimitKeyByIP),
			authHandler.VerifyTwoFactorLogin,
		)

		// Authenticated
		protected := authRoutes.Group("")
		protected.Use(authMiddleware)

		protected.PATCH("/password", authHandler.ChangePassword)
		protected.POST("/logout", authHandler.Logout)
		protected.PATCH("/me", userHandler.UpdateProfile)
		protected.PATCH("/me/image", userHandler.UpdateProfilePic)
		protected.GET("/me", userHandler.Me)

		// 2FA setup
		protected.POST(
			"/2fa/setup",
			twoFactorLimiter.Middleware(middleware.RateLimitKeyByUserID),
			authHandler.StartTwoFactorSetup,
		)

		protected.POST(
			"/2fa/verify-setup",
			twoFactorLimiter.Middleware(middleware.RateLimitKeyByUserID),
			authHandler.VerifyTwoFactorSetup,
		)

		protected.POST(
			"/2fa/disable",
			twoFactorLimiter.Middleware(middleware.RateLimitKeyByUserID),
			authHandler.DisableTwoFactor,
		)

		protected.POST(
			"/2fa/backup-codes/regenerate",
			twoFactorLimiter.Middleware(middleware.RateLimitKeyByUserID),
			authHandler.RegenerateBackupCodes,
		)
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
	customerRoutes.POST("/:id/2fa/reset", authHandler.AdminResetTwoFactor)
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

	// Admin pages
	admin.POST("/pages", pageHandler.Create)
	admin.GET("/pages", pageHandler.List)
	admin.PATCH("/pages/:id", pageHandler.Update)
	admin.DELETE("/pages/:id", pageHandler.Delete)

	// Public menus
	api.GET("/menus/:location", menuHandler.GetPublicByLocation)

	// Admin menus
	menus := admin.Group("/menus")
	menus.POST("", menuHandler.Create)
	menus.GET("", menuHandler.List)
	menus.GET("/:id", menuHandler.Get)
	menus.PATCH("/:id", menuHandler.Update)
	menus.DELETE("/:id", menuHandler.Delete)

	// Menu items
	menus.POST("/:id/items", menuHandler.AddItem)
	menus.PATCH("/:id/items/:itemId", menuHandler.UpdateItem)
	menus.DELETE("/:id/items/:itemId", menuHandler.DeleteItem)
	menus.PATCH("/:id/items/:itemId/order", menuHandler.MoveItem)

	return r
}
