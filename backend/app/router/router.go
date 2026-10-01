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

	"github.com/gin-gonic/gin"
	mongo "go.mongodb.org/mongo-driver/v2/mongo"
)

func NewRouter(database *mongo.Database, cfg config.Config) *gin.Engine {
	r := gin.Default()

	r.Static("/Uploads", "./Uploads") // Expose file uploads

	// Dependencies
	userRepository := mongorepo.NewUserRepository(database)

	settingsRepository := mongorepo.NewApplicationSettingsRepository(database)
	settingsService := services.NewSettingsService(settingsRepository)
	refreshTokenRepository := mongorepo.NewRefreshTokenRepository(database)
	authService := services.NewAuthService(userRepository, refreshTokenRepository, settingsService, cfg)
	settingsCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := settingsRepository.EnsureDefaults(settingsCtx); err != nil {
		log.Printf("Failed to initialize application settings: %v", err)
		panic(err)
	}

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

	userService := services.NewUserService(userRepository)
	authHandler := handler.NewAuthHandler(authService, cfg)
	userHandler := handler.NewUserHandler(userService)
	pageRepository := mongorepo.NewPageRepository(database)
	menuRepository := mongorepo.NewMenuRepository(database)
	menuService := services.NewMenuService(
		menuRepository,
		pageRepository,
	)
	pageService := services.NewPageService(pageRepository)
	pageHandler := handler.NewPageHandler(pageService)
	menuHandler := handler.NewMenuHandler(menuService)
	settingsHandler := handler.NewSettingsHandler(settingsService)
	// Global/Public Endpoints
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok":     true,
			"status": "application is up",
		})
	})

	// Authentication routes
	authRoutes := r.Group("/api/v1/auth")
	{
		// Public
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/login", authHandler.Login)
		authRoutes.POST("/refresh", authHandler.Refresh)

		// Authenticated
		protected := authRoutes.Group("")
		protected.Use(authMiddleware)

		protected.PATCH("/password", authHandler.ChangePassword)
		//	protected.POST("/refresh", authHandler.Refresh)
		protected.POST("/logout", authHandler.Logout)
		protected.PATCH("/me", userHandler.UpdateProfile)
		protected.PATCH("/me/image", userHandler.UpdateProfilePic)
		protected.GET("/me", userHandler.Me)
	}

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

	r.GET("/api/v1/settings", settingsHandler.GetPublicSettings)

	protectedSettings := r.Group("/api/v1/admin/settings")
	protectedSettings.Use(authMiddleware, auth.RequireRoles("admin"))
	protectedSettings.PATCH("", settingsHandler.Update)

	api := r.Group("/api/v1")

	// Public / registered pages
	api.GET(
		"/pages/:slug",
		optionalAuthMiddleware,
		pageHandler.GetBySlug,
	)

	// Public menus
	api.GET("/menus/:location", menuHandler.GetPublicByLocation)

	// Admin
	admin := api.Group("/admin")
	admin.Use(authMiddleware, auth.RequireRoles("admin"))

	// Admin pages
	admin.POST("/pages", pageHandler.Create)
	admin.GET("/pages", pageHandler.List)
	admin.PATCH("/pages/:id", pageHandler.Update)
	admin.DELETE("/pages/:id", pageHandler.Delete)

	// Admin menus
	menus := admin.Group("/menus")
	menus.POST("", menuHandler.Create)
	menus.GET("", menuHandler.List)
	menus.GET("/:id", menuHandler.Get)
	menus.PATCH("/:id", menuHandler.Update)
	menus.DELETE("/:id", menuHandler.Delete)

	// menu items
	menus.POST("/:id/items", menuHandler.AddItem)
	menus.PATCH("/:id/items/:itemId", menuHandler.UpdateItem)

	return r
}
