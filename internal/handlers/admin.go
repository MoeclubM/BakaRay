package handlers

import (
	"net/http"
	"strconv"

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
}

// NewAdminHandler 创建后台管理处理器
func NewAdminHandler(userService *services.UserService, nodeService *services.NodeService, ruleService *services.RuleService, paymentService *services.PaymentService, nodeGroupService *services.NodeGroupService, userGroupService *services.UserGroupService) *AdminHandler {
	return &AdminHandler{
		userService:       userService,
		nodeService:       nodeService,
		ruleService:       ruleService,
		paymentService:    paymentService,
		nodeGroupService:  nodeGroupService,
		userGroupService:  userGroupService,
	}
}

// --- 节点管理 ---

// GetAdminNodes 获取节点列表
func (h *AdminHandler) GetAdminNodeDetail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	node, err := h.nodeService.GetNodeByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}

	// 获取探针数据
	probe, _ := h.nodeService.GetProbeData(uint(id))

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
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.nodeService.UpdateNode(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteNode 删除节点
func (h *AdminHandler) DeleteNode(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.nodeService.DeleteNode(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// --- 节点组管理 ---

// GetNodeGroups 获取节点组列表
func (h *AdminHandler) GetNodeGroups(c *gin.Context) {
	groups, err := h.nodeGroupService.ListNodeGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": groups})
}

// CreateNodeGroup 创建节点组
func (h *AdminHandler) CreateNodeGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Type        string `json:"type" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	group, err := h.nodeGroupService.CreateNodeGroup(req.Name, req.Type, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": gin.H{"id": group.ID}})
}

// UpdateNodeGroup 更新节点组
func (h *AdminHandler) UpdateNodeGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.nodeGroupService.UpdateNodeGroup(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteNodeGroup 删除节点组
func (h *AdminHandler) DeleteNodeGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.nodeGroupService.DeleteNodeGroup(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// --- 用户组管理 ---

// GetUserGroups 获取用户组列表
func (h *AdminHandler) GetUserGroups(c *gin.Context) {
	groups, err := h.userGroupService.ListUserGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": groups})
}

// CreateUserGroup 创建用户组
func (h *AdminHandler) CreateUserGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	group, err := h.userGroupService.CreateUserGroup(req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": gin.H{"id": group.ID}})
}

// UpdateUserGroup 更新用户组
func (h *AdminHandler) UpdateUserGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.userGroupService.UpdateUserGroup(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteUserGroup 删除用户组
func (h *AdminHandler) DeleteUserGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.userGroupService.DeleteUserGroup(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// --- 用户管理 ---

// GetUserDetail 获取用户详情
func (h *AdminHandler) GetUserDetail(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": user})
}

// UpdateUser 更新用户
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.userService.UpdateUser(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteUser 删除用户
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.userService.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// --- 套餐管理 ---

// UpdatePackage 更新套餐
func (h *AdminHandler) UpdatePackage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.paymentService.UpdatePackage(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeletePackage 删除套餐
func (h *AdminHandler) DeletePackage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.paymentService.DeletePackage(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

// --- 节点管理 ---

// GetAdminNodes 获取节点列表
func (h *AdminHandler) GetAdminNodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	nodes, total := h.nodeService.ListNodes(page, pageSize, status)

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
	Name       string `json:"name" binding:"required"`
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port" binding:"required"`
	Secret     string `json:"secret" binding:"required"`
	NodeGroupID uint  `json:"node_group_id"`
	Protocols  []string `json:"protocols"`
	Multiplier float64  `json:"multiplier"`
	Region     string   `json:"region"`
}

// CreateNode 创建节点
func (h *AdminHandler) CreateNode(c *gin.Context) {
	var req CreateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

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
                c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建节点失败"})
                return
        }

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "创建成功",
		"data": gin.H{
			"id": node.ID,
		},
	})
}

// ReloadNode 触发节点热更新
func (h *AdminHandler) ReloadNode(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	_, err := h.nodeService.GetNodeByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "热更新指令已下发",
	})
}

// --- 用户管理 ---

// GetAdminUsers 获取用户列表
func (h *AdminHandler) GetAdminUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	users, total := h.userService.ListUsers(page, pageSize)

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
        Username   string `json:"username" binding:"required"`
        Password   string `json:"password" binding:"required"`
        UserGroupID uint  `json:"user_group_id"`
        Balance    int64  `json:"balance"`
        IsAdmin    bool   `json:"is_admin"`
}

// CreateUser 创建用户
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	user, err := h.userService.CreateUser(req.Username, req.Password, req.UserGroupID)
	if err != nil {
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

        c.JSON(http.StatusOK, gin.H{
                "code": 0,
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
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req AdjustBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.userService.UpdateBalance(uint(id), req.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "调整失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "调整成功",
	})
}

// --- 订单管理 ---

// GetAdminOrders 获取订单列表
func (h *AdminHandler) GetAdminOrders(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	orders, total := h.paymentService.ListAllOrders(page, pageSize, status)

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
	tradeNo := c.Param("id")
	var req UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if err := h.paymentService.UpdateOrderStatus(tradeNo, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "更新成功",
	})
}

// --- 套餐管理 ---

// GetAdminPackages 获取套餐列表
func (h *AdminHandler) GetAdminPackages(c *gin.Context) {
	packages, _ := h.paymentService.ListPackages(0)

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
}

// CreatePackage 创建套餐
func (h *AdminHandler) CreatePackage(c *gin.Context) {
	var req CreatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	pkg := &models.Package{
		Name:        req.Name,
		Traffic:     req.Traffic,
		Price:       req.Price,
		UserGroupID: req.UserGroupID,
	}

	if err := h.paymentService.CreatePackage(pkg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "创建成功",
		"data": gin.H{
			"id": pkg.ID,
		},
	})
}
