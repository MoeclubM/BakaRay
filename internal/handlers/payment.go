package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

// GetPackages 获取套餐列表
func (h *PaymentHandler) GetPackages(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, _ := h.paymentService.GetUserByID(userID)

	packages, err := h.paymentService.ListPackages(user.UserGroupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取套餐失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": packages,
	})
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	PackageID uint   `json:"package_id" binding:"required"`
	PayType   string `json:"pay_type" binding:"required"`
}

// CreateOrder 创建订单
func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	pkg, err := h.paymentService.GetPackageByID(req.PackageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "套餐不存在"})
		return
	}

	order, err := h.paymentService.CreateOrder(userID, req.PackageID, pkg.Price, req.PayType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建订单失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"order_id": order.ID,
			"trade_no": order.TradeNo,
			"amount":   order.Amount,
		},
	})
}

// GetOrders 获取我的订单
func (h *PaymentHandler) GetOrders(c *gin.Context) {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	orders, total := h.paymentService.ListOrders(userID, page, pageSize)

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
	userID := middleware.GetUserID(c)
	var req DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 获取订单信息
	order, err := h.paymentService.GetOrderByID(req.OrderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "订单不存在"})
		return
	}

	// 验证订单归属
	if order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权操作此订单"})
		return
	}

	// 获取支付配置
	config, err := h.paymentConfigService.GetPaymentConfigByType(req.PayType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "支付渠道不存在"})
		return
	}

	// 创建支付提供商
	provider := createPaymentProvider(config)

	// 创建支付订单
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建支付订单失败"})
		return
	}

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
	// 合并 GET query + POST form 参数
	_ = c.Request.ParseForm()
	params := make(map[string]string, len(c.Request.Form))
	for key, values := range c.Request.Form {
		if len(values) > 0 {
			params[key] = values[0]
		}
	}

	if len(params) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少参数"})
		return
	}

	// 获取支付配置（优先使用 pay_type，其次尝试 pid/默认配置）
	config, err := h.paymentConfigService.GetPaymentConfigForCallback(params)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "支付渠道不存在"})
		return
	}

	// 创建支付提供商并验证回调
	provider := createPaymentProvider(config)
	result, err := provider.VerifyCallback(params)
	if err != nil {
		if c.Request.Method == http.MethodPost {
			c.String(http.StatusOK, "fail")
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 获取订单
	order, err := h.paymentService.GetOrderByTradeNo(result.TradeNo)
	if err != nil {
		if c.Request.Method == http.MethodPost {
			c.String(http.StatusOK, "fail")
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "订单不存在"})
		return
	}

	// 验证金额
	if result.Amount != order.Amount {
		if c.Request.Method == http.MethodPost {
			c.String(http.StatusOK, "fail")
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "金额不匹配"})
		return
	}

	// 检查订单状态（防止重复处理）
	if order.Status != "pending" {
		if c.Request.Method == http.MethodPost {
			c.String(http.StatusOK, "success")
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "订单已处理"})
		return
	}

	// 检查支付状态
	if strings.EqualFold(result.Status, "TRADE_SUCCESS") || strings.EqualFold(result.Status, "TRADE_FINISHED") {
		pkg, err := h.paymentService.GetPackageByID(order.PackageID)
		if err != nil {
			if c.Request.Method == http.MethodPost {
				c.String(http.StatusOK, "fail")
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取套餐失败"})
			return
		}

		if err := h.paymentService.CompleteOrder(result.TradeNo, order.UserID, pkg.Traffic); err != nil {
			if c.Request.Method == http.MethodPost {
				c.String(http.StatusOK, "fail")
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新订单失败"})
			return
		}

		if c.Request.Method == http.MethodPost {
			c.String(http.StatusOK, "success")
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "处理成功"})
		return
	}

	_ = h.paymentService.UpdateOrderStatus(result.TradeNo, "failed")
	if c.Request.Method == http.MethodPost {
		c.String(http.StatusOK, "success")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 400, "message": "支付失败"})
}

// GetEnabledPayments 获取可用支付方式（不返回敏感密钥）
func (h *PaymentHandler) GetEnabledPayments(c *gin.Context) {
	configs, err := h.paymentConfigService.ListPaymentConfigs()
	if err != nil {
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

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": methods})
}

// --- 支付配置管理 ---

// GetPaymentConfigs 获取支付配置列表
func (h *PaymentHandler) GetPaymentConfigs(c *gin.Context) {
	configs, err := h.paymentConfigService.ListPaymentConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": configs})
}

// CreatePaymentConfig 创建支付配置
func (h *PaymentHandler) CreatePaymentConfig(c *gin.Context) {
	var cfg models.PaymentConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.paymentConfigService.CreatePaymentConfig(&cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": gin.H{"id": cfg.ID}})
}

// UpdatePaymentConfig 更新支付配置
func (h *PaymentHandler) UpdatePaymentConfig(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.paymentConfigService.UpdatePaymentConfig(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeletePaymentConfig 删除支付配置
func (h *PaymentHandler) DeletePaymentConfig(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.paymentConfigService.DeletePaymentConfig(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

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
