package main

import (
	"log"
	"os"

	"bakaray/internal/config"
	"bakaray/internal/handlers"
	"bakaray/internal/middleware"
	"bakaray/internal/repository"
	"bakaray/internal/services"
	"bakaray/routes"

	"github.com/gin-gonic/gin"
)

func main() {
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

	// 初始化 Redis
	redis, err := repository.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redis.Close()

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
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// 注册路由
	routes.Setup(r, authHandler, userHandler, nodeHandler, ruleHandler, paymentHandler, adminHandler, middleware.NewAuthMiddleware(userService))

	// 启动服务器
	log.Printf("Server starting on %s:%s", cfg.Server.Host, cfg.Server.Port)
	if err := r.Run(cfg.Server.Host + ":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func init() {
	// 确保日志目录存在
	os.MkdirAll("logs", 0755)
}
