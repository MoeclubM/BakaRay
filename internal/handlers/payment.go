package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"bakaray/internal/logger"
	"bakaray/internal/middleware"
	"bakaray/internal/models"
	"bakaray/internal/providers"
	"bakaray/internal/services"

	"github.com/gin-gonic/gin"
)

// PaymentHandler 支付处理器
type PaymentHandler struct {
	paymentService       *services.PaymentService
	paymentConfigService *services.PaymentConfigService
}

// NewPaymentHandler 创建支付处理器
func NewPaymentHandler(paymentService *services.PaymentService, paymentConfigService *services.PaymentConfigService) *PaymentHandler {
	return &PaymentHandler{
		paymentService:       paymentService,
		paymentConfigService: paymentConfigService,
	}
}

// GetPackages 获取套餐列表（只返回可见的套餐）
func (h *PaymentHandler) GetPackages(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "payment")

	log.Debug("GetPackages request")

	user, err := h.paymentService.GetUserByID(userID)
	if err != nil {
		logger.Error("GetPackages: user not found", err, "user_id", userID, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取套餐失败"})
		return
	}

	packages, err := h.paymentService.ListVisiblePackages(user.UserGroupID)
	if err != nil {
		logger.Error("GetPackages: failed to list packages", err, "user_id", userID, "user_group_id", user.UserGroupID, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取套餐失败"})
		return
	}

	// 构建返回数据，添加用户组名称
	type PackageWithGroup struct {
		models.Package
		UserGroupName string `json:"user_group_name"`
	}

	result := make([]PackageWithGroup, 0, len(packages))
	for _, pkg := range packages {
		pkgWithGroup := PackageWithGroup{
			Package:       pkg,
			UserGroupName: "",
		}
		if pkg.UserGroupID > 0 {
			pkgWithGroup.UserGroupName = fmt.Sprintf("用户组 #%d", pkg.UserGroupID)
		} else {
			pkgWithGroup.UserGroupName = "所有用户"
		}
		result = append(result, pkgWithGroup)
	}

	log.Info("GetPackages success", "package_count", len(result))

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	PackageID uint   `json:"package_id" binding:"required"`
	PayType   string `json:"pay_type" binding:"required"`
}

// CreateOrder 创建订单
func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "payment")

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("CreateOrder: invalid request", "error", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("CreateOrder request", "package_id", req.PackageID, "pay_type", req.PayType)

	pkg, err := h.paymentService.GetPackageByID(req.PackageID)
	if err != nil {
		logger.Warn("CreateOrder: package not found", "package_id", req.PackageID, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "套餐不存在"})
		return
	}

	// 检查套餐是否可见
	if !pkg.Visible {
		logger.Warn("CreateOrder: package not visible", "package_id", req.PackageID, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "套餐不可购买"})
		return
	}

	// 检查用户余额是否充足
	user, err := h.paymentService.GetUserByID(userID)
	if err != nil {
		logger.Error("CreateOrder: user not found", err, "user_id", userID, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取用户信息失败"})
		return
	}

	// 检查是否可续费（非续费套餐只能购买一次）
	if !pkg.Renewable {
		hasPurchased, err := h.paymentService.HasUserPurchasedPackage(userID, req.PackageID)
		if err != nil {
			logger.Error("CreateOrder: failed to check purchase history", err, "user_id", userID, "package_id", req.PackageID, "request_id", requestID)
		}
		if hasPurchased {
			logger.Warn("CreateOrder: package not renewable, user already purchased", "user_id", userID, "package_id", req.PackageID, "request_id", requestID)
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "该套餐不可重复购买"})
			return
		}
	}

	// 判断是否为余额支付
	isBalancePayment := req.PayType == "balance" || req.PayType == "" || req.PayType == "1"

	if isBalancePayment {
		// 余额支付：直接扣款并完成订单
		if user.Balance < pkg.Price {
			logger.Warn("CreateOrder: insufficient balance", "user_id", userID, "balance", user.Balance, "required", pkg.Price, "request_id", requestID)
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "余额不足，请先充值"})
			return
		}

		// 创建已完成的订单并扣款
		order, err := h.paymentService.CreateAndCompleteOrder(userID, req.PackageID, pkg.Price)
		if err != nil {
			logger.Error("CreateOrder: failed to complete order", err, "user_id", userID, "package_id", req.PackageID, "request_id", requestID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建订单失败"})
			return
		}

		log.Info("CreateOrder success (balance payment)", "order_id", order.ID, "trade_no", order.TradeNo, "amount", order.Amount, "user_id", userID)

		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"order_id":  order.ID,
				"trade_no":  order.TradeNo,
				"amount":    order.Amount,
				"status":    "completed",
			},
			"message": "购买成功",
		})
		return
	}

	// 其他支付方式：创建待支付订单
	order, err := h.paymentService.CreateOrder(userID, req.PackageID, pkg.Price, req.PayType)
	if err != nil {
		logger.Error("CreateOrder: failed to create order", err, "user_id", userID, "package_id", req.PackageID, "pay_type", req.PayType, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建订单失败"})
		return
	}

	log.Info("CreateOrder success", "order_id", order.ID, "trade_no", order.TradeNo, "amount", order.Amount)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"order_id": order.ID,
			"trade_no": order.TradeNo,
			"amount":   order.Amount,
			"status":   "pending",
		},
	})
}

// GetOrders 获取我的订单
func (h *PaymentHandler) GetOrders(c *gin.Context) {
        requestID := c.GetString("request_id")
        userID := middleware.GetUserID(c)
        log := logger.WithContext(requestID, userID, "payment")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	log.Debug("GetOrders request", "page", page, "page_size", pageSize)

	orders, total := h.paymentService.ListOrders(userID, page, pageSize)

	log.Info("GetOrders success", "order_count", len(orders), "total", total)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  orders,
			"total": total,
			"page":  page,
		},
	})
}

// DepositRequest 发起充值请求
type DepositRequest struct {
	OrderID uint   `json:"order_id" binding:"required"`
	PayType string `json:"pay_type" binding:"required"`
}

// Deposit 发起充值
func (h *PaymentHandler) Deposit(c *gin.Context) {
        requestID := c.GetString("request_id")
        userID := middleware.GetUserID(c)
        log := logger.WithContext(requestID, userID, "payment")

	var req DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Deposit: invalid request", "error", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("Deposit request", "order_id", req.OrderID, "pay_type", req.PayType)

	order, err := h.paymentService.GetOrderByID(req.OrderID)
	if err != nil {
		logger.Warn("Deposit: order not found", "order_id", req.OrderID, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "订单不存在"})
		return
	}

	// 检查订单是否属于当前用户
	if order.UserID != userID {
		logger.Warn("Deposit: unauthorized order access", "order_id", req.OrderID, "order_user_id", order.UserID, "user_id", userID, "request_id", requestID)
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权操作此订单"})
		return
	}

	// 检查订单状态
	if order.Status == "success" {
		logger.Warn("Deposit: order already completed", "order_id", req.OrderID, "trade_no", order.TradeNo, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "订单已完成，无需重复支付"})
		return
	}

	if order.Status != "pending" {
		logger.Warn("Deposit: order status invalid", "order_id", req.OrderID, "status", order.Status, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "订单状态异常"})
		return
	}

	config, err := h.paymentConfigService.GetPaymentConfigByType(req.PayType)
	if err != nil {
		logger.Warn("Deposit: payment config not found", "pay_type", req.PayType, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "支付渠道不存在"})
		return
	}

	provider := createPaymentProvider(config)

	pkg, _ := h.paymentService.GetPackageByID(order.PackageID)
	subject := "BakaRay 套餐购买"
	if pkg.ID > 0 {
		subject = pkg.Name
	}

	resp, err := provider.CreateOrder(&providers.CreateOrderRequest{
		TradeNo:   order.TradeNo,
		Amount:    order.Amount,
		Subject:   subject,
		NotifyURL: config.NotifyURL,
		ReturnURL: fmt.Sprintf("%s/deposit/callback", getSiteDomain(c)),
	})

	if err != nil {
		logger.Error("Deposit: failed to create payment order", err, "order_id", req.OrderID, "trade_no", order.TradeNo, "pay_type", req.PayType, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建支付订单失败"})
		return
	}

	log.Info("Deposit success", "order_id", req.OrderID, "trade_no", order.TradeNo, "amount", order.Amount, "pay_type", req.PayType)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"pay_url":  resp.PayURL,
			"trade_no": resp.TradeNo,
			"amount":   resp.Amount,
		},
	})
}

// PaymentCallback 支付回调
func (h *PaymentHandler) PaymentCallback(c *gin.Context) {
	requestID := c.GetString("request_id")
	log := logger.Log.With("request_id", requestID, "component", "payment")

	_ = c.Request.ParseForm()
	params := make(map[string]string, len(c.Request.Form))
	for key, values := range c.Request.Form {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	if len(params) == 0 {
		logger.Warn("PaymentCallback: missing parameters", "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少参数"})
		return
	}

	log.Debug("PaymentCallback request", "method", c.Request.Method, "params_count", len(params))

	config, err := h.paymentConfigService.GetPaymentConfigForCallback(params)
	if err != nil {
		logger.Warn("PaymentCallback: payment config not found", "error", err, "request_id", requestID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "支付渠道不存在"})
		return
	}

	provider := createPaymentProvider(config)
	result, err := provider.VerifyCallback(params)
	if err != nil {
		logger.Warn("PaymentCallback: callback verification failed", "error", err, "request_id", requestID)
		if c.Request.Method == http.MethodPost {
			c.String(http.StatusOK, "fail")
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	log.Debug("PaymentCallback: verification success", "trade_no", result.TradeNo, "amount", result.Amount, "status", result.Status)

	order, err := h.paymentService.GetOrderByTradeNo(result.TradeNo)
	if err != nil {
		logger.Warn("PaymentCallback: order not found", "trade_no", result.TradeNo, "request_id", requestID)
		if c.Request.Method == http.MethodPost {
			c.String(http.StatusOK, "fail")
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "订单不存在"})
		return
	}

	if result.Amount != order.Amount {
		logger.Warn("PaymentCallback: amount mismatch", "trade_no", result.TradeNo, "expected_amount", order.Amount, "received_amount", result.Amount, "request_id", requestID)
		if c.Request.Method == http.MethodPost {
			c.String(http.StatusOK, "fail")
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "金额不匹配"})
		return
	}

	if order.Status != "pending" {
		logger.Info("PaymentCallback: order already processed", "trade_no", result.TradeNo, "status", order.Status, "request_id", requestID)
		if c.Request.Method == http.MethodPost {
			c.String(http.StatusOK, "success")
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "订单已处理"})
		return
	}

	if strings.EqualFold(result.Status, "TRADE_SUCCESS") || strings.EqualFold(result.Status, "TRADE_FINISHED") {
		// 获取套餐信息
		pkg, err := h.paymentService.GetPackageByID(order.PackageID)
		if err != nil {
			logger.Error("PaymentCallback: failed to get package", err, "trade_no", result.TradeNo, "package_id", order.PackageID, "request_id", requestID)
			if c.Request.Method == http.MethodPost {
				c.String(http.StatusOK, "fail")
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取套餐失败"})
			return
		}

		// 使用幂等性方法完成订单（带分布式锁）
		if err := h.paymentService.CompleteOrderWithLock(result.TradeNo, order.UserID, pkg.Traffic); err != nil {
			logger.Error("PaymentCallback: failed to complete order", err, "trade_no", result.TradeNo, "user_id", order.UserID, "package_id", order.PackageID, "request_id", requestID)
			if c.Request.Method == http.MethodPost {
				c.String(http.StatusOK, "fail")
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新订单失败"})
			return
		}

		log.Info("PaymentCallback: order completed successfully", "trade_no", result.TradeNo, "user_id", order.UserID, "amount", order.Amount, "traffic", pkg.Traffic, "request_id", requestID)

		if c.Request.Method == http.MethodPost {
			c.String(http.StatusOK, "success")
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "处理成功"})
		return
	}

	logger.Info("PaymentCallback: payment failed", "trade_no", result.TradeNo, "status", result.Status, "request_id", requestID)

	_ = h.paymentService.UpdateOrderStatus(result.TradeNo, "failed")
	if c.Request.Method == http.MethodPost {
		c.String(http.StatusOK, "success")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 400, "message": "支付失败"})
}

// GetEnabledPayments 获取可用支付方式（不返回敏感密钥）
func (h *PaymentHandler) GetEnabledPayments(c *gin.Context) {
	requestID := c.GetString("request_id")
	log := logger.Log.With("request_id", requestID, "component", "payment")

	log.Debug("GetEnabledPayments request")

	configs, err := h.paymentConfigService.ListPaymentConfigs()
	if err != nil {
		logger.Error("GetEnabledPayments: failed to list payment configs", err, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取失败"})
		return
	}

	type PaymentMethod struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Provider string `json:"provider"`
	}

	methods := make([]PaymentMethod, 0, len(configs))
	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}
		methods = append(methods, PaymentMethod{
			ID:       cfg.ID,
			Name:     cfg.Name,
			Provider: cfg.Provider,
		})
	}

	log.Info("GetEnabledPayments success", "method_count", len(methods))

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": methods})
}

// --- 支付配置管理 ---

// GetPaymentConfigs 获取支付配置列表
func (h *PaymentHandler) GetPaymentConfigs(c *gin.Context) {
        requestID := c.GetString("request_id")
        userID := middleware.GetUserID(c)
        log := logger.WithContext(requestID, userID, "payment")

	log.Debug("GetPaymentConfigs request")

	configs, err := h.paymentConfigService.ListPaymentConfigs()
	if err != nil {
		logger.Error("GetPaymentConfigs: failed to list payment configs", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取失败"})
		return
	}

	log.Info("GetPaymentConfigs success", "config_count", len(configs))

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": configs})
}

// CreatePaymentConfig 创建支付配置
func (h *PaymentHandler) CreatePaymentConfig(c *gin.Context) {
        requestID := c.GetString("request_id")
        userID := middleware.GetUserID(c)
        log := logger.WithContext(requestID, userID, "payment")

	var cfg models.PaymentConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		logger.Warn("CreatePaymentConfig: invalid request", "error", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("CreatePaymentConfig request", "name", cfg.Name, "provider", cfg.Provider)

	if err := h.paymentConfigService.CreatePaymentConfig(&cfg); err != nil {
		logger.Error("CreatePaymentConfig: failed to create payment config", err, "name", cfg.Name, "provider", cfg.Provider, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	log.Info("CreatePaymentConfig success", "config_id", cfg.ID, "name", cfg.Name, "provider", cfg.Provider)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": gin.H{"id": cfg.ID}})
}

// UpdatePaymentConfig 更新支付配置
func (h *PaymentHandler) UpdatePaymentConfig(c *gin.Context) {
        requestID := c.GetString("request_id")
        userID := middleware.GetUserID(c)
        log := logger.WithContext(requestID, userID, "payment")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		logger.Warn("UpdatePaymentConfig: invalid request", "error", err, "config_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("UpdatePaymentConfig request", "config_id", id)

	if err := h.paymentConfigService.UpdatePaymentConfig(uint(id), updates); err != nil {
		logger.Error("UpdatePaymentConfig: failed to update payment config", err, "config_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	log.Info("UpdatePaymentConfig success", "config_id", id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeletePaymentConfig 删除支付配置
func (h *PaymentHandler) DeletePaymentConfig(c *gin.Context) {
        requestID := c.GetString("request_id")
        userID := middleware.GetUserID(c)
        log := logger.WithContext(requestID, userID, "payment")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	log.Debug("DeletePaymentConfig request", "config_id", id)

	if err := h.paymentConfigService.DeletePaymentConfig(uint(id)); err != nil {
		logger.Error("DeletePaymentConfig: failed to delete payment config", err, "config_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	log.Info("DeletePaymentConfig success", "config_id", id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// --- 辅助函数 ---

// createPaymentProvider 创建支付提供商
func createPaymentProvider(config *models.PaymentConfig) providers.PaymentProvider {
	switch config.Provider {
	case "epay":
		return providers.NewEpayProvider(
			config.MerchantID,
			config.MerchantKey,
			config.APIURL,
			config.NotifyURL,
		)
	default:
		return providers.NewEpayProvider(
			config.MerchantID,
			config.MerchantKey,
			config.APIURL,
			config.NotifyURL,
		)
	}
}

// getSiteDomain 获取站点域名
func getSiteDomain(c *gin.Context) string {
	domain := c.Query("site_domain")
	if domain == "" {
		domain = c.Request.Host
	}
	if domain == "" {
		domain = "http://localhost:8080"
	}
	if !strings.HasPrefix(domain, "http://") && !strings.HasPrefix(domain, "https://") {
		domain = "http://" + domain
	}
	return domain
}
