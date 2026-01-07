package main

import (
	"os"

	"bakaray/internal/config"
	"bakaray/internal/handlers"
	"bakaray/internal/logger"
	"bakaray/internal/middleware"
	"bakaray/internal/repository"
	"bakaray/internal/services"
	"bakaray/routes"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	if err := logger.Init(os.Getenv("LOG_LEVEL")); err != nil {
		logger.Error("Failed to initialize logger", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load config", err)
		os.Exit(1)
	}

	db, err := repository.NewDB(cfg.Database)
	if err != nil {
		logger.Error("Failed to connect to database", err,
			"db_type", cfg.Database.Type,
			"db_host", cfg.Database.Host,
			"db_port", cfg.Database.Port,
			"db_name", cfg.Database.Name,
			"db_path", cfg.Database.Path,
		)
		os.Exit(1)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	logger.Info("Migrating database...")
	if err := repository.AutoMigrate(db); err != nil {
		logger.Warn("Database migration failed", "error", err)
	}

	redis, err := repository.NewRedis(cfg.Redis)
	if err != nil {
		logger.Warn("Failed to connect to Redis", "error", err)
		redis = nil
	} else if redis != nil {
		defer redis.Close()
	}

	userService := services.NewUserService(db, redis)
	nodeService := services.NewNodeService(db, redis)
	ruleService := services.NewRuleService(db, redis)
	paymentService := services.NewPaymentService(db, redis)
	paymentConfigService := services.NewPaymentConfigService(db)
	siteConfigService := services.NewSiteConfigService(db)
	nodeGroupService := services.NewNodeGroupService(db)
	userGroupService := services.NewUserGroupService(db)

	authHandler := handlers.NewAuthHandler(userService)
	userHandler := handlers.NewUserHandler(userService, ruleService)
	nodeHandler := handlers.NewNodeHandler(nodeService, ruleService, userService)
	ruleHandler := handlers.NewRuleHandler(ruleService)
	paymentHandler := handlers.NewPaymentHandler(paymentService, paymentConfigService)
	adminHandler := handlers.NewAdminHandler(userService, nodeService, ruleService, paymentService, nodeGroupService, userGroupService, siteConfigService)

	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		requestID := c.GetHeader("X-Request-ID")
		if requestID != "" {
			c.Set("request_id", requestID)
			c.Header("X-Request-ID", requestID)
		}
		c.Next()
	})

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.Static("/public", "/app/public")
	r.Static("/assets", "/app/public/assets")
	r.Static("/vite.svg", "/app/public/vite.svg")

	r.GET("/", func(c *gin.Context) {
		c.File("/app/public/index.html")
	})

	frontendRoutes := []string{
		"/login", "/register", "/dashboard", "/nodes", "/rules",
		"/packages", "/orders", "/profile", "/deposit/callback",
		"/admin", "/admin/dashboard", "/admin/nodes", "/admin/users",
		"/admin/packages", "/admin/orders", "/admin/node-groups",
		"/admin/user-groups", "/admin/payments", "/admin/settings",
	}
	for _, route := range frontendRoutes {
		r.GET(route, func(c *gin.Context) {
			c.File("/app/public/index.html")
		})
	}

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) > 4 && path[:4] != "/api" {
			c.File("/app/public/index.html")
		}
	})

	routes.Setup(r, authHandler, userHandler, nodeHandler, ruleHandler, paymentHandler, adminHandler, middleware.NewAuthMiddleware(userService))

	logger.Info("Server starting", "host", cfg.Server.Host, "port", cfg.Server.Port)
	if err := r.Run(cfg.Server.Host + ":" + cfg.Server.Port); err != nil {
		logger.Error("Failed to start server", err)
		os.Exit(1)
	}
}
