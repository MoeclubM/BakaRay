package main

import (
	"flag"
	"log"
	"os"

	"bakaray/internal/config"
	"bakaray/internal/handlers"
	"bakaray/internal/middleware"
	"bakaray/internal/models"
	"bakaray/internal/repository"
	"bakaray/internal/services"
	"bakaray/routes"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func main() {
	// 命令行参数
	initDB := flag.Bool("init-db", false, "Initialize database with default admin user")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库
	db, err := repository.NewDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// 自动迁移数据库表
	log.Println("Migrating database...")
	if err := repository.AutoMigrate(db); err != nil {
		log.Printf("Warning: Database migration failed: %v", err)
	}

	// 如果指定 --init-db，创建默认管理员
	if *initDB {
		log.Println("Initializing database with default admin user...")
		var count int64
		db.Model(&models.User{}).Count(&count)
		if count == 0 {
			hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
			admin := models.User{
				Username:     "admin",
				PasswordHash: string(hash),
				Balance:      0,
				UserGroupID:  0,
				Role:         "admin",
			}
			if err := db.Create(&admin).Error; err != nil {
				log.Printf("Failed to create admin user: %v", err)
			} else {
				log.Println("Default admin user created: admin / admin123")
			}
		} else {
			log.Println("Admin user already exists, skipping.")
		}
		os.Exit(0)
	}

	// 初始化 Redis (可选)
	redis, err := repository.NewRedis(cfg.Redis)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v (continuing without Redis)", err)
		redis = nil
	} else if redis != nil {
		defer redis.Close()
	}

	// 初始化服务
	userService := services.NewUserService(db, redis)
	nodeService := services.NewNodeService(db, redis)
	ruleService := services.NewRuleService(db, redis)
	paymentService := services.NewPaymentService(db, redis)
	paymentConfigService := services.NewPaymentConfigService(db)
	nodeGroupService := services.NewNodeGroupService(db)
	userGroupService := services.NewUserGroupService(db)

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(userService)
	userHandler := handlers.NewUserHandler(userService, ruleService)
	nodeHandler := handlers.NewNodeHandler(nodeService, ruleService)
	ruleHandler := handlers.NewRuleHandler(ruleService)
	paymentHandler := handlers.NewPaymentHandler(paymentService, paymentConfigService)
	adminHandler := handlers.NewAdminHandler(userService, nodeService, ruleService, paymentService, nodeGroupService, userGroupService)

	// 创建 Gin 引擎
	r := gin.New()

	// 自定义 MIME 类型中间件
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	})

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// 提供前端静态文件
	r.Static("/public", "/app/public")
	r.Static("/assets", "/app/public/assets")
	r.Static("/vite.svg", "/app/public/vite.svg")

	// 首页路由
	r.GET("/", func(c *gin.Context) {
		c.File("/app/public/index.html")
	})

	// Vue SPA 路由支持
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

	// 捕获所有前端路由（兜底）
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) > 4 && path[:4] != "/api" {
			c.File("/app/public/index.html")
		}
	})

	// 注册 API 路由
	routes.Setup(r, authHandler, userHandler, nodeHandler, ruleHandler, paymentHandler, adminHandler, middleware.NewAuthMiddleware(userService))

	// 启动服务器
	log.Printf("Server starting on %s:%s", cfg.Server.Host, cfg.Server.Port)
	if err := r.Run(cfg.Server.Host + ":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
