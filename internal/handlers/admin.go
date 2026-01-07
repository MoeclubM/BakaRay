package handlers

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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
	nodeGroupService  *services.NodeGroupService
	userGroupService  *services.UserGroupService
	siteConfigService *services.SiteConfigService
}

// NewAdminHandler 创建后台管理处理器
func NewAdminHandler(userService *services.UserService, nodeService *services.NodeService, ruleService *services.RuleService, paymentService *services.PaymentService, nodeGroupService *services.NodeGroupService, userGroupService *services.UserGroupService, siteConfigService *services.SiteConfigService) *AdminHandler {
	return &AdminHandler{
		userService:       userService,
		nodeService:       nodeService,
		ruleService:       ruleService,
		paymentService:    paymentService,
		nodeGroupService:  nodeGroupService,
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

	log.Info("GetAdminNodeDetail success", "node_id", id, "node_name", node.Name)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"node":  node,
			"probe": probe,
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

	if err := h.nodeService.UpdateNode(uint(id), updates); err != nil {
		logger.Error("UpdateNode: failed to update node", err, "node_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
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

// --- 节点组管理 ---

// GetNodeGroups 获取节点组列表
func (h *AdminHandler) GetNodeGroups(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	log.Debug("GetNodeGroups request")

	groups, err := h.nodeGroupService.ListNodeGroups()
	if err != nil {
		logger.Error("GetNodeGroups: failed to list node groups", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取失败"})
		return
	}

	log.Info("GetNodeGroups success", "group_count", len(groups))

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": groups})
}

// CreateNodeGroup 创建节点组
func (h *AdminHandler) CreateNodeGroup(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Type        string `json:"type" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("CreateNodeGroup: invalid request", "error", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("CreateNodeGroup request", "name", req.Name, "type", req.Type)

	group, err := h.nodeGroupService.CreateNodeGroup(req.Name, req.Type, req.Description)
	if err != nil {
		logger.Error("CreateNodeGroup: failed to create node group", err, "name", req.Name, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	log.Info("CreateNodeGroup success", "group_id", group.ID, "name", req.Name)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": gin.H{"id": group.ID}})
}

// UpdateNodeGroup 更新节点组
func (h *AdminHandler) UpdateNodeGroup(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		logger.Warn("UpdateNodeGroup: invalid request", "error", err, "group_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("UpdateNodeGroup request", "group_id", id)

	if err := h.nodeGroupService.UpdateNodeGroup(uint(id), updates); err != nil {
		logger.Error("UpdateNodeGroup: failed to update node group", err, "group_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	log.Info("UpdateNodeGroup success", "group_id", id)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteNodeGroup 删除节点组
func (h *AdminHandler) DeleteNodeGroup(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	log.Debug("DeleteNodeGroup request", "group_id", id)

	if err := h.nodeGroupService.DeleteNodeGroup(uint(id)); err != nil {
		logger.Error("DeleteNodeGroup: failed to delete node group", err, "group_id", id, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	log.Info("DeleteNodeGroup success", "group_id", id)

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

	log.Info("GetAdminNodes success", "node_count", len(nodes), "total", total)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  nodes,
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

	var req CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("CreateNode: invalid request", "error", err, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("CreateNode request", "name", req.Name, "host", req.Host, "port", req.Port)

	multiplier := req.Multiplier
	if multiplier <= 0 {
		multiplier = 1
	}

	node, err := h.nodeService.CreateNode(
		req.Name,
		req.Host,
		req.Port,
		req.Secret,
		req.NodeGroupID,
		req.Protocols,
		multiplier,
		req.Region,
	)
	if err != nil {
		logger.Error("CreateNode: failed to create node", err, "name", req.Name, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建节点失败"})
		return
	}

	log.Info("CreateNode success", "node_id", node.ID, "name", req.Name)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"id": node.ID,
		},
	})
}

// ReloadNode 触发节点热更新
func (h *AdminHandler) ReloadNode(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "admin")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	log.Debug("ReloadNode request", "node_id", id)

	node, err := h.nodeService.GetNodeByID(uint(id))
	if err != nil {
		logger.Warn("ReloadNode: node not found", "node_id", id, "request_id", requestID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}

	reloadURL := buildNodeURL(node.Host, node.Port) + "/reload"
	req, err := http.NewRequest(http.MethodPost, reloadURL, nil)
	if err != nil {
		logger.Error("ReloadNode: build request failed", err, "node_id", id, "url", reloadURL, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成请求失败"})
		return
	}
	req.Header.Set("X-Node-Secret", node.Secret)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error("ReloadNode: request failed", err, "node_id", id, "url", reloadURL, "request_id", requestID)
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "下发失败：节点不可达"})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Warn("ReloadNode: node returned non-200", "node_id", id, "url", reloadURL, "status", resp.StatusCode, "request_id", requestID)
		c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "下发失败：节点响应异常"})
		return
	}

	log.Info("ReloadNode success", "node_id", id)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "热更新指令已下发",
	})
}

func buildNodeURL(host string, port int) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	if strings.Contains(host, "://") {
		if u, err := url.Parse(host); err == nil {
			if u.Scheme == "" {
				u.Scheme = "http"
			}
			if u.Host == "" {
				u.Host = host
			}

			if u.Port() == "" && port > 0 {
				u.Host = net.JoinHostPort(u.Hostname(), strconv.Itoa(port))
			}

			return strings.TrimRight(u.String(), "/")
		}
	}

	if ip := net.ParseIP(host); ip != nil && port > 0 {
		return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
	}

	if _, _, err := net.SplitHostPort(host); err == nil {
		return "http://" + strings.TrimRight(host, "/")
	}

	if port > 0 {
		return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
	}

	return "http://" + strings.TrimRight(host, "/")
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
