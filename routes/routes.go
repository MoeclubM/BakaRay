package routes

import (
	"bakaray/internal/handlers"
	"bakaray/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 注册所有路由
func Setup(
	r *gin.Engine,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	nodeHandler *handlers.NodeHandler,
	ruleHandler *handlers.RuleHandler,
	paymentHandler *handlers.PaymentHandler,
	adminHandler *handlers.AdminHandler,
	authMiddleware *middleware.AuthMiddleware,
) {
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API 路由组
	api := r.Group("/api")
	{
		// 认证模块
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/refresh", authHandler.Refresh)
		}

		// 需要认证的路由
		protected := api.Group("")
		protected.Use(authMiddleware.Authenticate())
		{
			// 用户模块
			user := protected.Group("/user")
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
			user.POST("/change-password", userHandler.ChangePassword)

			// 节点模块
			nodes := protected.Group("/nodes")
			nodes.GET("", nodeHandler.GetNodes)
			nodes.GET("/:id", nodeHandler.GetNode)

			// 转发规则模块
			rules := protected.Group("/rules")
			rules.GET("", ruleHandler.GetRules)
			rules.POST("", ruleHandler.CreateRule)
			rules.GET("/:id", ruleHandler.GetRule)
			rules.PUT("/:id", ruleHandler.UpdateRule)
			rules.DELETE("/:id", ruleHandler.DeleteRule)

			// 套餐模块
			protected.GET("/packages", paymentHandler.GetPackages)
			// 支付方式（用户可用）
			protected.GET("/payments", paymentHandler.GetEnabledPayments)

			// 订单模块
			protected.GET("/orders", paymentHandler.GetOrders)
			protected.POST("/orders", paymentHandler.CreateOrder)

			// 充值模块
			protected.POST("/deposit", paymentHandler.Deposit)

			// 流量统计
			protected.GET("/statistics/traffic", userHandler.GetTrafficStats)
		}

		// 支付回调（第三方通知/前端回跳，不需要用户认证）
		api.GET("/deposit/callback", paymentHandler.PaymentCallback)
		api.POST("/deposit/callback", paymentHandler.PaymentCallback)

		// 节点通信接口（不需要用户认证，使用节点密钥验证）
		nodeAPI := api.Group("/node")
		{
			nodeAPI.POST("/heartbeat", nodeHandler.NodeHeartbeat)
			nodeAPI.GET("/config", nodeHandler.NodeConfig)
			nodeAPI.POST("/config", nodeHandler.NodeConfig)
			nodeAPI.POST("/report", nodeHandler.NodeReport)
		}

		// 后台管理接口
		admin := api.Group("/admin")
		admin.Use(authMiddleware.Authenticate(), authMiddleware.AdminRequired())
		{
			// 管理员统计
			admin.GET("/stats/overview", adminHandler.GetOverviewStats)
			admin.GET("/rules/count", ruleHandler.CountRules)

			// 站点配置
			site := admin.Group("/site")
			{
				site.GET("", adminHandler.GetSiteConfig)
				site.PUT("", adminHandler.UpdateSiteConfig)
			}

			// 支付配置
			payments := admin.Group("/payments")
			{
				payments.GET("", paymentHandler.GetPaymentConfigs)
				payments.POST("", paymentHandler.CreatePaymentConfig)
				payments.PUT("/:id", paymentHandler.UpdatePaymentConfig)
				payments.DELETE("/:id", paymentHandler.DeletePaymentConfig)
			}

			// 节点组
			nodeGroups := admin.Group("/node-groups")
			{
				nodeGroups.GET("", adminHandler.GetNodeGroups)
				nodeGroups.POST("", adminHandler.CreateNodeGroup)
				nodeGroups.PUT("/:id", adminHandler.UpdateNodeGroup)
				nodeGroups.DELETE("/:id", adminHandler.DeleteNodeGroup)
			}

			// 节点管理
			adminNodes := admin.Group("/nodes")
			{
				adminNodes.GET("", adminHandler.GetAdminNodes)
				adminNodes.POST("", adminHandler.CreateNode)
				adminNodes.GET("/:id", adminHandler.GetAdminNodeDetail)
				adminNodes.PUT("/:id", adminHandler.UpdateNode)
				adminNodes.DELETE("/:id", adminHandler.DeleteNode)
				adminNodes.POST("/:id/reload", adminHandler.ReloadNode)
			}

			// 用户组
			userGroups := admin.Group("/user-groups")
			{
				userGroups.GET("", adminHandler.GetUserGroups)
				userGroups.POST("", adminHandler.CreateUserGroup)
				userGroups.PUT("/:id", adminHandler.UpdateUserGroup)
				userGroups.DELETE("/:id", adminHandler.DeleteUserGroup)
			}

			// 套餐配置
			adminPackages := admin.Group("/packages")
			{
				adminPackages.GET("", adminHandler.GetAdminPackages)
				adminPackages.POST("", adminHandler.CreatePackage)
				adminPackages.PUT("/:id", adminHandler.UpdatePackage)
				adminPackages.DELETE("/:id", adminHandler.DeletePackage)
			}

			// 用户管理
			adminUsers := admin.Group("/users")
			{
				adminUsers.GET("", adminHandler.GetAdminUsers)
				adminUsers.POST("", adminHandler.CreateUser)
				adminUsers.GET("/:id", adminHandler.GetUserDetail)
				adminUsers.PUT("/:id", adminHandler.UpdateUser)
				adminUsers.DELETE("/:id", adminHandler.DeleteUser)
				adminUsers.POST("/:id/balance", adminHandler.AdjustBalance)
			}

			// 订单管理
			adminOrders := admin.Group("/orders")
			{
				adminOrders.GET("", adminHandler.GetAdminOrders)
				adminOrders.PUT("/:id/status", adminHandler.UpdateOrderStatus)
			}
		}
	}
}
