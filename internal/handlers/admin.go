package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"bakaray/internal/logger"
	"bakaray/internal/middleware"
	"bakaray/internal/models"
	"bakaray/internal/services"

	"github.com/gin-gonic/gin"
)

// AdminHandler 后台管理处理器
type AdminHandler struct {
	userService       *services.UserService
	nodeService       *services.NodeService
	ruleService       *services.RuleService
	paymentService    *services.PaymentService
	userGroupService  *services.UserGroupService
	siteConfigService *services.SiteConfigService
}

// NewAdminHandler 创建后台管理处理器
func NewAdminHandler(userService *services.UserService, nodeService *services.NodeService, ruleService *services.RuleService, paymentService *services.PaymentService, userGroupService *services.UserGroupService, siteConfigService *services.SiteConfigService) *AdminHandler {
	return &AdminHandler{
		userService:       userService,
		nodeService:       nodeService,
		ruleService:       ruleService,
		paymentService:    paymentService,
		userGroupService:  userGroupService,
		siteConfigService: siteConfigService,
	}
}

// --- 站点配置 ---

func (h *AdminHandler) GetSiteConfig(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	if h.siteConfigService == nil {
		logger.Error("GetSiteConfig: site config service not initialized", errors.New("site config service not initialized"), "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	site, err := h.siteConfigService.GetOrCreate()
	if err != nil {
		logger.Error("GetSiteConfig: failed to load site config", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取失败"})
		return
	}

	log.Info("GetSiteConfig success")
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": site})
}

type UpdateSiteConfigRequest struct {
	SiteName           string `json:"site_name"`
	SiteDomain         string `json:"site_domain"`
	NodeSecret         string `json:"node_secret"`
	NodeReportInterval *int   `json:"node_report_interval"`
}

func (h *AdminHandler) UpdateSiteConfig(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	if h.siteConfigService == nil {
		logger.Error("UpdateSiteConfig: site config service not initialized", errors.New("site config service not initialized"), "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	var req UpdateSiteConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("UpdateSiteConfig: invalid request", "error", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	updates := make(map[string]any)
	if req.SiteName != "" {
		updates["site_name"] = req.SiteName
	}
	if req.SiteDomain != "" {
		updates["site_domain"] = req.SiteDomain
	}
	if req.NodeSecret != "" {
		updates["node_secret"] = req.NodeSecret
	}
	if req.NodeReportInterval != nil {
		if *req.NodeReportInterval <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "node_report_interval 必须 > 0"})
			return
		}
		updates["node_report_interval"] = *req.NodeReportInterval
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "没有可更新的内容"})
		return
	}

	log.Debug("UpdateSiteConfig request", "updates", updates)

	site, err := h.siteConfigService.Update(updates)
	if err != nil {
		logger.Error("UpdateSiteConfig: failed to update site config", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	log.Info("UpdateSiteConfig success")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": site})
}

// --- 节点管理 ---

// GetAdminNodes 获取节点列表
func (h *AdminHandler) GetAdminNodeDetail(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithAdminContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	log.Debug("GetAdminNodeDetail request", "node_id", id)

	node, err := h.nodeService.GetNodeByID(uint(id))
	if err != nil {
		logger.Warn("GetAdminNodeDetail: node not found", "node_id", id, "request_id", requestID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}

	probe, _ := h.nodeService.GetProbeData(uint(id))
	diagnostics, _ := h.nodeService.GetDiagnostics(uint(id))
	allowedGroupIDs, _ := h.nodeService.GetAllowedGroups(uint(id))

	log.Info("GetAdminNodeDetail success", "node_id", id, "node_name", node.Name)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"node":              node,
			"probe":             probe,
			"diagnostics":       diagnostics,
			"allowed_group_ids": allowedGroupIDs,
		},
	})
}

// UpdateNode 更新节点
func (h *AdminHandler) UpdateNode(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithAdminContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		logger.Warn("UpdateNode: invalid request", "error", err, "node_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("UpdateNode request", "node_id", id)

	rawAllowedGroupIDs, hasAllowedGroupIDs := updates["allowed_group_ids"]
	delete(updates, "allowed_group_ids")

	if len(updates) > 0 {
		if err := h.nodeService.UpdateNode(uint(id), updates); err != nil {
			logger.Error("UpdateNode: failed to update node", err, "node_id", id, "request_id", requestID, "user_id", userID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
			return
		}
	} else if _, err := h.nodeService.GetNodeByID(uint(id)); err != nil {
		logger.Error("UpdateNode: node not found", err, "node_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}

	if hasAllowedGroupIDs {
		groupIDs := make([]uint, 0)
		switch values := rawAllowedGroupIDs.(type) {
		case []interface{}:
			for _, value := range values {
				switch typed := value.(type) {
				case float64:
					if typed > 0 {
						groupIDs = append(groupIDs, uint(typed))
					}
				case int:
					if typed > 0 {
						groupIDs = append(groupIDs, uint(typed))
					}
				case uint:
					if typed > 0 {
						groupIDs = append(groupIDs, typed)
					}
				}
			}
		case []uint:
			groupIDs = values
		}

		if err := h.nodeService.SetAllowedGroups(uint(id), groupIDs); err != nil {
			logger.Error("UpdateNode: failed to update allowed groups", err, "node_id", id, "request_id", requestID, "user_id", userID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新节点授权失败"})
			return
		}
	}

	log.Info("UpdateNode success", "node_id", id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteNode 删除节点
func (h *AdminHandler) DeleteNode(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithAdminContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	log.Debug("DeleteNode request", "node_id", id)

	if err := h.nodeService.DeleteNode(uint(id)); err != nil {
		logger.Error("DeleteNode: failed to delete node", err, "node_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	log.Info("DeleteNode success", "node_id", id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// --- 用户组管理 ---

// GetUserGroups 获取用户组列表
func (h *AdminHandler) GetUserGroups(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	log.Debug("GetUserGroups request")

	groups, err := h.userGroupService.ListUserGroups()
	if err != nil {
		logger.Error("GetUserGroups: failed to list user groups", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取失败"})
		return
	}

	log.Info("GetUserGroups success", "group_count", len(groups))

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": groups})
}

// CreateUserGroup 创建用户组
func (h *AdminHandler) CreateUserGroup(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("CreateUserGroup: invalid request", "error", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("CreateUserGroup request", "name", req.Name)

	group, err := h.userGroupService.CreateUserGroup(req.Name, req.Description)
	if err != nil {
		logger.Error("CreateUserGroup: failed to create user group", err, "name", req.Name, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	log.Info("CreateUserGroup success", "group_id", group.ID, "name", req.Name)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": gin.H{"id": group.ID}})
}

// UpdateUserGroup 更新用户组
func (h *AdminHandler) UpdateUserGroup(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		logger.Warn("UpdateUserGroup: invalid request", "error", err, "group_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("UpdateUserGroup request", "group_id", id)

	if err := h.userGroupService.UpdateUserGroup(uint(id), updates); err != nil {
		logger.Error("UpdateUserGroup: failed to update user group", err, "group_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	log.Info("UpdateUserGroup success", "group_id", id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteUserGroup 删除用户组
func (h *AdminHandler) DeleteUserGroup(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	log.Debug("DeleteUserGroup request", "group_id", id)

	if err := h.userGroupService.DeleteUserGroup(uint(id)); err != nil {
		logger.Error("DeleteUserGroup: failed to delete user group", err, "group_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	log.Info("DeleteUserGroup success", "group_id", id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// --- 用户管理 ---

// GetUserDetail 获取用户详情
func (h *AdminHandler) GetUserDetail(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	log.Debug("GetUserDetail request", "target_user_id", id)

	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		logger.Warn("GetUserDetail: user not found", "target_user_id", id, "request_id", requestID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	log.Info("GetUserDetail success", "target_user_id", id, "username", user.Username)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": user})
}

// UpdateUser 更新用户
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		logger.Warn("UpdateUser: invalid request", "error", err, "target_user_id", id, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("UpdateUser request", "target_user_id", id)

	if err := h.userService.UpdateUser(uint(id), updates); err != nil {
		logger.Error("UpdateUser: failed to update user", err, "target_user_id", id, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	log.Info("UpdateUser success", "target_user_id", id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteUser 删除用户
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	log.Debug("DeleteUser request", "target_user_id", id)

	if err := h.userService.DeleteUser(uint(id)); err != nil {
		logger.Error("DeleteUser: failed to delete user", err, "target_user_id", id, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	log.Info("DeleteUser success", "target_user_id", id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// --- 套餐管理 ---

// UpdatePackage 更新套餐
func (h *AdminHandler) UpdatePackage(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		logger.Warn("UpdatePackage: invalid request", "error", err, "package_id", id, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("UpdatePackage request", "package_id", id)

	if err := h.paymentService.UpdatePackage(uint(id), updates); err != nil {
		logger.Error("UpdatePackage: failed to update package", err, "package_id", id, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	log.Info("UpdatePackage success", "package_id", id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeletePackage 删除套餐
func (h *AdminHandler) DeletePackage(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	log.Debug("DeletePackage request", "package_id", id)

	if err := h.paymentService.DeletePackage(uint(id)); err != nil {
		logger.Error("DeletePackage: failed to delete package", err, "package_id", id, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	log.Info("DeletePackage success", "package_id", id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// --- 节点管理 ---

// GetAdminNodes 获取节点列表
func (h *AdminHandler) GetAdminNodes(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	log.Debug("GetAdminNodes request", "page", page, "page_size", pageSize, "status", status)

	nodes, total := h.nodeService.ListNodes(page, pageSize, status)
	type AdminNodeListItem struct {
		models.Node
		Diagnostics     []models.NodeDiagnostic `json:"diagnostics"`
		AllowedGroupIDs []uint                  `json:"allowed_group_ids"`
	}
	items := make([]AdminNodeListItem, 0, len(nodes))
	for _, node := range nodes {
		diagnostics, _ := h.nodeService.GetDiagnostics(node.ID)
		allowedGroupIDs, _ := h.nodeService.GetAllowedGroups(node.ID)
		items = append(items, AdminNodeListItem{
			Node:            node,
			Diagnostics:     diagnostics,
			AllowedGroupIDs: allowedGroupIDs,
		})
	}

	log.Info("GetAdminNodes success", "node_count", len(items), "total", total)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  items,
			"total": total,
			"page":  page,
		},
	})
}

// CreateNodeRequest 创建节点请求
type CreateNodeRequest struct {
	Name        string   `json:"name" binding:"required"`
	Host        string   `json:"host" binding:"required"`
	Port        int      `json:"port" binding:"required"`
	Secret      string   `json:"secret" binding:"required"`
	NodeGroupID uint     `json:"node_group_id"`
	Protocols   []string `json:"protocols"`
	Multiplier  float64  `json:"multiplier"`
	Region      string   `json:"region"`
}

// CreateNode 创建节点
func (h *AdminHandler) CreateNode(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	log.Warn("CreateNode: manual creation disabled", "request_id", requestID, "user_id", userID)
	c.JSON(http.StatusBadRequest, gin.H{
		"code":    400,
		"message": "节点仅支持安装脚本自动注册，不支持手动添加",
	})
}

// --- 用户管理 ---

// GetAdminUsers 获取用户列表
func (h *AdminHandler) GetAdminUsers(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	log.Debug("GetAdminUsers request", "page", page, "page_size", pageSize)

	users, total := h.userService.ListUsers(page, pageSize)

	log.Info("GetAdminUsers success", "user_count", len(users), "total", total)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  users,
			"total": total,
			"page":  page,
		},
	})
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	UserGroupID uint   `json:"user_group_id"`
	Balance     int64  `json:"balance"`
	IsAdmin     bool   `json:"is_admin"`
}

// CreateUser 创建用户
func (h *AdminHandler) CreateUser(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("CreateUser: invalid request", "error", err, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("CreateUser request", "username", req.Username, "user_group_id", req.UserGroupID, "is_admin", req.IsAdmin)

	user, err := h.userService.CreateUser(req.Username, req.Password, req.UserGroupID)
	if err != nil {
		logger.Error("CreateUser: failed to create user", err, "username", req.Username, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 设置初始余额
	if req.Balance > 0 {
		h.userService.UpdateBalance(user.ID, req.Balance)
	}

	// 设置管理员角色
	if req.IsAdmin {
		_ = h.userService.UpdateUser(user.ID, map[string]interface{}{"role": "admin"})
	}

	log.Info("CreateUser success", "user_id", user.ID, "username", req.Username)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"id": user.ID,
		},
	})
}

// AdjustBalanceRequest 调整余额请求
type AdjustBalanceRequest struct {
	Amount int64 `json:"amount" binding:"required"`
}

// AdjustBalance 调整用户余额
func (h *AdminHandler) AdjustBalance(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req AdjustBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("AdjustBalance: invalid request", "error", err, "target_user_id", id, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("AdjustBalance request", "target_user_id", id, "amount", req.Amount)

	if err := h.userService.UpdateBalance(uint(id), req.Amount); err != nil {
		logger.Error("AdjustBalance: failed to update balance", err, "target_user_id", id, "amount", req.Amount, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "调整失败"})
		return
	}

	log.Info("AdjustBalance success", "target_user_id", id, "amount", req.Amount)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "调整成功",
	})
}

// --- 订单管理 ---

// GetAdminOrders 获取订单列表
func (h *AdminHandler) GetAdminOrders(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	log.Debug("GetAdminOrders request", "page", page, "page_size", pageSize, "status", status)

	orders, total := h.paymentService.ListAllOrders(page, pageSize, status)

	log.Info("GetAdminOrders success", "order_count", len(orders), "total", total)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  orders,
			"total": total,
			"page":  page,
		},
	})
}

// UpdateOrderStatusRequest 更新订单状态请求
type UpdateOrderStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdateOrderStatus 更新订单状态
func (h *AdminHandler) UpdateOrderStatus(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	tradeNo := c.Param("id")
	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("UpdateOrderStatus: invalid request", "error", err, "trade_no", tradeNo, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 验证状态值
	validStatuses := map[string]bool{"pending": true, "processing": true, "success": true, "failed": true, "refunded": true}
	if !validStatuses[req.Status] {
		logger.Warn("UpdateOrderStatus: invalid status", "trade_no", tradeNo, "status", req.Status, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的订单状态"})
		return
	}

	log.Debug("UpdateOrderStatus request", "trade_no", tradeNo, "status", req.Status)

	// 检查订单是否存在
	order, err := h.paymentService.GetOrderByTradeNo(tradeNo)
	if err != nil {
		logger.Warn("UpdateOrderStatus: order not found", "trade_no", tradeNo, "request_id", requestID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "订单不存在"})
		return
	}

	// 检查状态转换是否合法
	if order.Status == "success" {
		logger.Warn("UpdateOrderStatus: cannot modify completed order", "trade_no", tradeNo, "current_status", order.Status, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无法修改已完成的订单"})
		return
	}

	// 禁止将订单改为 pending 状态
	if req.Status == "pending" {
		logger.Warn("UpdateOrderStatus: cannot set status to pending", "trade_no", tradeNo, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不能将订单状态设为待支付"})
		return
	}

	if err := h.paymentService.UpdateOrderStatus(tradeNo, req.Status); err != nil {
		logger.Error("UpdateOrderStatus: failed to update order status", err, "trade_no", tradeNo, "status", req.Status, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	log.Info("UpdateOrderStatus success", "trade_no", tradeNo, "status", req.Status)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

// --- 统计概览 ---

// GetOverviewStats 获取概览统计数据
func (h *AdminHandler) GetOverviewStats(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	log.Debug("GetOverviewStats request")

	// 获取用户总数
	userCount := h.userService.CountUsers()

	// 获取节点总数
	nodeCount := h.nodeService.CountNodes()

	// 获取订单总数和总收入
	orderCount, totalRevenue := h.paymentService.GetOrderStats()

	log.Info("GetOverviewStats success",
		"user_count", userCount,
		"node_count", nodeCount,
		"order_count", orderCount,
		"total_revenue", totalRevenue)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"user_count":    userCount,
			"node_count":    nodeCount,
			"order_count":   orderCount,
			"total_revenue": totalRevenue,
		},
	})
}

// --- 套餐管理 ---

// GetAdminPackages 获取套餐列表
func (h *AdminHandler) GetAdminPackages(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	log.Debug("GetAdminPackages request")

	packages, _ := h.paymentService.ListPackages(0)

	log.Info("GetAdminPackages success", "package_count", len(packages))

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": packages,
	})
}

// CreatePackageRequest 创建套餐请求
type CreatePackageRequest struct {
	Name        string `json:"name" binding:"required"`
	Traffic     int64  `json:"traffic" binding:"required"`
	Price       int64  `json:"price" binding:"required"`
	UserGroupID uint   `json:"user_group_id"`
	Visible     bool   `json:"visible"`
	Renewable   bool   `json:"renewable"`
}

// CreatePackage 创建套餐
func (h *AdminHandler) CreatePackage(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	var req CreatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("CreatePackage: invalid request", "error", err, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("CreatePackage request", "name", req.Name, "traffic", req.Traffic, "price", req.Price, "user_group_id", req.UserGroupID, "visible", req.Visible, "renewable", req.Renewable)

	pkg := &models.Package{
		Name:        req.Name,
		Traffic:     req.Traffic,
		Price:       req.Price,
		UserGroupID: req.UserGroupID,
		Visible:     req.Visible,
		Renewable:   req.Renewable,
	}

	if err := h.paymentService.CreatePackage(pkg); err != nil {
		logger.Error("CreatePackage: failed to create package", err, "name", req.Name, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	log.Info("CreatePackage success", "package_id", pkg.ID, "name", req.Name)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"id": pkg.ID,
		},
	})
}
