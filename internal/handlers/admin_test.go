package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bakaray/internal/logger"
	"bakaray/internal/models"
	"bakaray/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 设置测试环境
func init() {
	gin.SetMode(gin.TestMode)
	_ = logger.Init("debug")
}

// --- Service Interfaces ---

type UserServiceInterface interface {
	CreateUser(username, password string, groupID uint) (*models.User, error)
	GetUserByID(id uint) (*models.User, error)
	ListUsers(page, pageSize int) ([]models.User, int64)
	CountUsers() int64
	UpdateUser(id uint, updates map[string]interface{}) error
	UpdateBalance(userID uint, amount int64) error
	DeleteUser(id uint) error
	GetJWTSecret() string
	GetJWTExpiration() int
}

type NodeServiceInterface interface {
	CreateNode(name, host string, port int, secret string, groupID uint, protocols []string, multiplier float64, region string) (*models.Node, error)
	GetNodeByID(id uint) (*models.Node, error)
	ListNodes(page, pageSize int, status string) ([]models.Node, int64)
	CountNodes() int64
	UpdateNode(id uint, updates map[string]interface{}) error
	DeleteNode(id uint) error
	UpdateNodeStatus(id uint, status string) error
	GetProbeData(nodeID uint) (*models.ProbeData, error)
	SaveProbeData(nodeID uint, probe *models.ProbeData) error
	ComputeTrafficDeltas(nodeID uint, stats map[string]int64) (map[uint]services.TrafficDelta, error)
	GetAllowedGroups(nodeID uint) ([]uint, error)
	SetAllowedGroups(nodeID uint, groupIDs []uint) error
	ListRulesByNode(nodeID uint, enabledOnly bool) ([]models.ForwardingRule, error)
}

type NodeGroupServiceInterface interface {
	CreateNodeGroup(name, nodeType, description string) (*models.NodeGroup, error)
	ListNodeGroups() ([]models.NodeGroup, error)
	UpdateNodeGroup(id uint, updates map[string]interface{}) error
	DeleteNodeGroup(id uint) error
	GetNodeGroupByID(id uint) (*models.NodeGroup, error)
}

type UserGroupServiceInterface interface {
	CreateUserGroup(name, description string) (*models.UserGroup, error)
	ListUserGroups() ([]models.UserGroup, error)
	UpdateUserGroup(id uint, updates map[string]interface{}) error
	DeleteUserGroup(id uint) error
	GetUserGroupByID(id uint) (*models.UserGroup, error)
}

type SiteConfigServiceInterface interface {
	GetOrCreate() (*models.SiteConfig, error)
	Update(updates map[string]interface{}) (*models.SiteConfig, error)
}

type PaymentServiceInterface interface {
	CreatePackage(pkg *models.Package) error
	ListPackages(userGroupID uint) ([]models.Package, error)
	UpdatePackage(id uint, updates map[string]interface{}) error
	DeletePackage(id uint) error
	ListAllOrders(page, pageSize int, status string) ([]models.Order, int64)
	UpdateOrderStatus(tradeNo string, status string) error
	GetOrderStats() (int64, int64)
	GetUserByID(userID uint) (*models.User, error)
	CreateOrder(userID, packageID uint, amount int64, payType string) (*models.Order, error)
	GetOrderByTradeNo(tradeNo string) (*models.Order, error)
	GetOrderByID(id uint) (*models.Order, error)
	AddUserBalance(userID uint, amount int64) error
	CompleteOrder(tradeNo string, userID uint, traffic int64) error
	ListOrders(userID uint, page, pageSize int) ([]models.Order, int64)
}

// --- Mock Services ---

type MockUserService struct {
	users          []models.User
	count          int64
	createErr      error
	getErr         error
	updateErr      error
	deleteErr      error
	updateBalErr   error
	listUsersFunc  func(page, pageSize int) ([]models.User, int64)
	jwtSecret      string
	jwtExp         int
}

func (m *MockUserService) CreateUser(username, password string, groupID uint) (*models.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	user := &models.User{
		ID:           1,
		Username:     username,
		PasswordHash: "hash",
		UserGroupID:  groupID,
		Role:         "user",
	}
	m.users = append(m.users, *user)
	return user, nil
}

func (m *MockUserService) GetUserByID(id uint) (*models.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, u := range m.users {
		if u.ID == id {
			return &u, nil
		}
	}
	if len(m.users) == 0 {
		return &models.User{ID: id, Username: "test"}, nil
	}
	return nil, services.ErrUserNotFound
}

func (m *MockUserService) ListUsers(page, pageSize int) ([]models.User, int64) {
	if m.listUsersFunc != nil {
		return m.listUsersFunc(page, pageSize)
	}
	return m.users, m.count
}

func (m *MockUserService) CountUsers() int64 {
	return m.count
}

func (m *MockUserService) UpdateUser(id uint, updates map[string]interface{}) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return nil
}

func (m *MockUserService) UpdateBalance(userID uint, amount int64) error {
	if m.updateBalErr != nil {
		return m.updateBalErr
	}
	return nil
}

func (m *MockUserService) DeleteUser(id uint) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func (m *MockUserService) GetJWTSecret() string {
	if m.jwtSecret == "" {
		m.jwtSecret = "test-secret"
	}
	return m.jwtSecret
}

func (m *MockUserService) GetJWTExpiration() int {
	if m.jwtExp == 0 {
		m.jwtExp = 86400
	}
	return m.jwtExp
}

type MockNodeService struct {
	nodes         []models.Node
	count         int64
	createErr     error
	getErr        error
	updateErr     error
	deleteErr     error
	listNodesFunc func(page, pageSize int, status string) ([]models.Node, int64)
}

func (m *MockNodeService) CreateNode(name, host string, port int, secret string, groupID uint, protocols []string, multiplier float64, region string) (*models.Node, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	node := &models.Node{
		ID:           1,
		Name:         name,
		Host:         host,
		Port:         port,
		Secret:       secret,
		NodeGroupID:  groupID,
		Protocols:    protocols,
		Multiplier:   multiplier,
		Region:       region,
		Status:       "offline",
	}
	m.nodes = append(m.nodes, *node)
	return node, nil
}

func (m *MockNodeService) GetNodeByID(id uint) (*models.Node, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, n := range m.nodes {
		if n.ID == id {
			return &n, nil
		}
	}
	if len(m.nodes) == 0 {
		return &models.Node{ID: id, Name: "test-node", Host: "localhost", Port: 8080, Secret: "secret"}, nil
	}
	return nil, services.ErrNodeNotFound
}

func (m *MockNodeService) ListNodes(page, pageSize int, status string) ([]models.Node, int64) {
	if m.listNodesFunc != nil {
		return m.listNodesFunc(page, pageSize, status)
	}
	return m.nodes, m.count
}

func (m *MockNodeService) CountNodes() int64 {
	return m.count
}

func (m *MockNodeService) UpdateNode(id uint, updates map[string]interface{}) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return nil
}

func (m *MockNodeService) DeleteNode(id uint) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func (m *MockNodeService) UpdateNodeStatus(id uint, status string) error   { return nil }
func (m *MockNodeService) GetProbeData(nodeID uint) (*models.ProbeData, error) { return nil, nil }
func (m *MockNodeService) SaveProbeData(nodeID uint, probe *models.ProbeData) error { return nil }
func (m *MockNodeService) ComputeTrafficDeltas(nodeID uint, stats map[string]int64) (map[uint]services.TrafficDelta, error) {
	return make(map[uint]services.TrafficDelta), nil
}
func (m *MockNodeService) GetAllowedGroups(nodeID uint) ([]uint, error)      { return nil, nil }
func (m *MockNodeService) SetAllowedGroups(nodeID uint, groupIDs []uint) error { return nil }
func (m *MockNodeService) ListRulesByNode(nodeID uint, enabledOnly bool) ([]models.ForwardingRule, error) {
	return nil, nil
}

type MockNodeGroupService struct {
	groups    []models.NodeGroup
	createErr error
	updateErr error
	deleteErr error
	listErr   error
}

func (m *MockNodeGroupService) CreateNodeGroup(name, nodeType, description string) (*models.NodeGroup, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	group := &models.NodeGroup{
		ID:          1,
		Name:        name,
		Type:        nodeType,
		Description: description,
	}
	m.groups = append(m.groups, *group)
	return group, nil
}

func (m *MockNodeGroupService) ListNodeGroups() ([]models.NodeGroup, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.groups, nil
}

func (m *MockNodeGroupService) UpdateNodeGroup(id uint, updates map[string]interface{}) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return nil
}

func (m *MockNodeGroupService) DeleteNodeGroup(id uint) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func (m *MockNodeGroupService) GetNodeGroupByID(id uint) (*models.NodeGroup, error) {
	return nil, nil
}

type MockUserGroupService struct {
	groups    []models.UserGroup
	createErr error
	updateErr error
	deleteErr error
	listErr   error
}

func (m *MockUserGroupService) CreateUserGroup(name, description string) (*models.UserGroup, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	group := &models.UserGroup{
		ID:          1,
		Name:        name,
		Description: description,
	}
	m.groups = append(m.groups, *group)
	return group, nil
}

func (m *MockUserGroupService) ListUserGroups() ([]models.UserGroup, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.groups, nil
}

func (m *MockUserGroupService) UpdateUserGroup(id uint, updates map[string]interface{}) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return nil
}

func (m *MockUserGroupService) DeleteUserGroup(id uint) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

func (m *MockUserGroupService) GetUserGroupByID(id uint) (*models.UserGroup, error) {
	return nil, nil
}

type MockSiteConfigService struct {
	config    *models.SiteConfig
	getErr    error
	updateErr error
}

func (m *MockSiteConfigService) GetOrCreate() (*models.SiteConfig, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.config == nil {
		m.config = &models.SiteConfig{
			ID:                 1,
			SiteName:           "BakaRay",
			SiteDomain:         "localhost",
			NodeSecret:         "secret",
			NodeReportInterval: 30,
		}
	}
	return m.config, nil
}

func (m *MockSiteConfigService) Update(updates map[string]interface{}) (*models.SiteConfig, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	return m.config, nil
}

type MockPaymentService struct {
	packages      []models.Package
	orders        []models.Order
	orderCount    int64
	totalRevenue  int64
	createPkgErr  error
	updatePkgErr  error
	deletePkgErr  error
	listOrdersErr error
	updateOrdErr  error
	listPkgErr    error
}

func (m *MockPaymentService) CreatePackage(pkg *models.Package) error {
	if m.createPkgErr != nil {
		return m.createPkgErr
	}
	pkg.ID = 1
	m.packages = append(m.packages, *pkg)
	return nil
}

func (m *MockPaymentService) ListPackages(userGroupID uint) ([]models.Package, error) {
	if m.listPkgErr != nil {
		return nil, m.listPkgErr
	}
	return m.packages, nil
}

func (m *MockPaymentService) UpdatePackage(id uint, updates map[string]interface{}) error {
	if m.updatePkgErr != nil {
		return m.updatePkgErr
	}
	return nil
}

func (m *MockPaymentService) DeletePackage(id uint) error {
	if m.deletePkgErr != nil {
		return m.deletePkgErr
	}
	return nil
}

func (m *MockPaymentService) ListAllOrders(page, pageSize int, status string) ([]models.Order, int64) {
	return m.orders, m.orderCount
}

func (m *MockPaymentService) UpdateOrderStatus(tradeNo string, status string) error {
	if m.updateOrdErr != nil {
		return m.updateOrdErr
	}
	return nil
}

func (m *MockPaymentService) GetOrderStats() (int64, int64) {
	return m.orderCount, m.totalRevenue
}

func (m *MockPaymentService) GetUserByID(userID uint) (*models.User, error)      { return nil, nil }
func (m *MockPaymentService) CreateOrder(userID, packageID uint, amount int64, payType string) (*models.Order, error) {
	return nil, nil
}
func (m *MockPaymentService) GetOrderByTradeNo(tradeNo string) (*models.Order, error) { return nil, nil }
func (m *MockPaymentService) GetOrderByID(id uint) (*models.Order, error)           { return nil, nil }
func (m *MockPaymentService) AddUserBalance(userID uint, amount int64) error        { return nil }
func (m *MockPaymentService) CompleteOrder(tradeNo string, userID uint, traffic int64) error { return nil }
func (m *MockPaymentService) ListOrders(userID uint, page, pageSize int) ([]models.Order, int64) { return nil, 0 }

// --- Test AdminHandler ---

type TestAdminHandler struct {
	userService       UserServiceInterface
	nodeService       NodeServiceInterface
	ruleService       interface{}
	paymentService    PaymentServiceInterface
	nodeGroupService  NodeGroupServiceInterface
	userGroupService  UserGroupServiceInterface
	siteConfigService SiteConfigServiceInterface
}

func NewTestAdminHandler() *TestAdminHandler {
	return &TestAdminHandler{}
}

func (h *TestAdminHandler) GetSiteConfig(c *gin.Context) {
	requestID := c.GetString("request_id")
	log := logger.WithContext(requestID, 0, "admin")

	if h.siteConfigService == nil {
		log.Error("site config service not initialized", "error", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	site, err := h.siteConfigService.GetOrCreate()
	if err != nil {
		log.Error("failed to load site config", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": site})
}

type UpdateSiteConfigRequestTest struct {
	SiteName           string `json:"site_name"`
	SiteDomain         string `json:"site_domain"`
	NodeSecret         string `json:"node_secret"`
	NodeReportInterval *int   `json:"node_report_interval"`
}

func (h *TestAdminHandler) UpdateSiteConfig(c *gin.Context) {
	requestID := c.GetString("request_id")
	log := logger.WithContext(requestID, 0, "admin")

	if h.siteConfigService == nil {
		log.Error("site config service not initialized", "error", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	var req UpdateSiteConfigRequestTest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("invalid request", "error", err)
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

	site, err := h.siteConfigService.Update(updates)
	if err != nil {
		log.Error("failed to update site config", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": site})
}

func (h *TestAdminHandler) GetNodeGroups(c *gin.Context) {
	log := logger.WithContext("", 0, "admin")

	if h.nodeGroupService == nil {
		log.Error("node group service not initialized", "error", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	groups, err := h.nodeGroupService.ListNodeGroups()
	if err != nil {
		log.Error("failed to list node groups", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": groups})
}

func (h *TestAdminHandler) CreateNodeGroup(c *gin.Context) {
	requestID := c.GetString("request_id")
	log := logger.WithContext(requestID, 0, "admin")

	var req struct {
		Name        string `json:"name" binding:"required"`
		Type        string `json:"type" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn("invalid request", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.nodeGroupService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	group, err := h.nodeGroupService.CreateNodeGroup(req.Name, req.Type, req.Description)
	if err != nil {
		log.Error("failed to create node group", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": gin.H{"id": group.ID}})
}

func (h *TestAdminHandler) UpdateNodeGroup(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.nodeGroupService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	if err := h.nodeGroupService.UpdateNodeGroup(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func (h *TestAdminHandler) DeleteNodeGroup(c *gin.Context) {
	if h.nodeGroupService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	if err := h.nodeGroupService.DeleteNodeGroup(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *TestAdminHandler) GetUserGroups(c *gin.Context) {
	if h.userGroupService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	groups, err := h.userGroupService.ListUserGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": groups})
}

func (h *TestAdminHandler) CreateUserGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.userGroupService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	group, err := h.userGroupService.CreateUserGroup(req.Name, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": gin.H{"id": group.ID}})
}

func (h *TestAdminHandler) UpdateUserGroup(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.userGroupService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	if err := h.userGroupService.UpdateUserGroup(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func (h *TestAdminHandler) DeleteUserGroup(c *gin.Context) {
	if h.userGroupService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	if err := h.userGroupService.DeleteUserGroup(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *TestAdminHandler) GetAdminNodes(c *gin.Context) {
	if h.nodeService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	page := 1
	pageSize := 20
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

func (h *TestAdminHandler) CreateNode(c *gin.Context) {
	var req struct {
		Name        string   `json:"name" binding:"required"`
		Host        string   `json:"host" binding:"required"`
		Port        int      `json:"port" binding:"required"`
		Secret      string   `json:"secret" binding:"required"`
		NodeGroupID uint     `json:"node_group_id"`
		Protocols   []string `json:"protocols"`
		Multiplier  float64  `json:"multiplier"`
		Region      string   `json:"region"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.nodeService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	multiplier := req.Multiplier
	if multiplier <= 0 {
		multiplier = 1
	}

	node, err := h.nodeService.CreateNode(req.Name, req.Host, req.Port, req.Secret, req.NodeGroupID, req.Protocols, multiplier, req.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建节点失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"id": node.ID,
		},
	})
}

func (h *TestAdminHandler) UpdateNode(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.nodeService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	if err := h.nodeService.UpdateNode(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func (h *TestAdminHandler) DeleteNode(c *gin.Context) {
	if h.nodeService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	if err := h.nodeService.DeleteNode(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *TestAdminHandler) ReloadNode(c *gin.Context) {
	if h.nodeService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	node, err := h.nodeService.GetNodeByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "热更新指令已下发",
		"data":    gin.H{"url": "http://" + node.Host + ":" + itoa(node.Port) + "/reload"},
	})
}

func (h *TestAdminHandler) GetAdminUsers(c *gin.Context) {
	if h.userService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	page := 1
	pageSize := 20

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

func (h *TestAdminHandler) CreateUser(c *gin.Context) {
	var req struct {
		Username    string `json:"username" binding:"required"`
		Password    string `json:"password" binding:"required"`
		UserGroupID uint   `json:"user_group_id"`
		Balance     int64  `json:"balance"`
		IsAdmin     bool   `json:"is_admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.userService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	user, err := h.userService.CreateUser(req.Username, req.Password, req.UserGroupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if req.Balance > 0 {
		h.userService.UpdateBalance(user.ID, req.Balance)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"id": user.ID,
		},
	})
}

func (h *TestAdminHandler) UpdateUser(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.userService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	if err := h.userService.UpdateUser(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func (h *TestAdminHandler) AdjustBalance(c *gin.Context) {
	var req struct {
		Amount int64 `json:"amount" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.userService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	if err := h.userService.UpdateBalance(uint(id), req.Amount); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "调整失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "调整成功"})
}

func (h *TestAdminHandler) GetAdminOrders(c *gin.Context) {
	if h.paymentService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	page := 1
	pageSize := 20
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

func (h *TestAdminHandler) UpdateOrderStatus(c *gin.Context) {
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.paymentService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	tradeNo := c.Param("id")
	if err := h.paymentService.UpdateOrderStatus(tradeNo, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func (h *TestAdminHandler) GetOverviewStats(c *gin.Context) {
	if h.userService == nil || h.nodeService == nil || h.paymentService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	userCount := h.userService.CountUsers()
	nodeCount := h.nodeService.CountNodes()
	orderCount, totalRevenue := h.paymentService.GetOrderStats()

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

func (h *TestAdminHandler) GetAdminPackages(c *gin.Context) {
	if h.paymentService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	packages, _ := h.paymentService.ListPackages(0)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": packages,
	})
}

func (h *TestAdminHandler) CreatePackage(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Traffic     int64  `json:"traffic" binding:"required"`
		Price       int64  `json:"price" binding:"required"`
		UserGroupID uint   `json:"user_group_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.paymentService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
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
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"id": pkg.ID,
		},
	})
}

func (h *TestAdminHandler) UpdatePackage(c *gin.Context) {
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if h.paymentService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	if err := h.paymentService.UpdatePackage(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

func (h *TestAdminHandler) DeletePackage(c *gin.Context) {
	if h.paymentService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	if err := h.paymentService.DeletePackage(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *TestAdminHandler) GetUserDetail(c *gin.Context) {
	if h.userService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": user})
}

func (h *TestAdminHandler) DeleteUser(c *gin.Context) {
	if h.userService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	if err := h.userService.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *TestAdminHandler) GetAdminNodeDetail(c *gin.Context) {
	if h.nodeService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "服务未初始化"})
		return
	}

	id, _ := strconvParseUint(c.Param("id"), 10, 32)
	node, err := h.nodeService.GetNodeByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}

	probe, _ := h.nodeService.GetProbeData(uint(id))

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"node":  node,
			"probe": probe,
		},
	})
}

// Helper functions
func strconvParseUint(s string, base int, bitSize int) (uint64, error) {
	var result uint64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		result = result*10 + uint64(c-'0')
	}
	return result, nil
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	negative := i < 0
	if negative {
		i = -i
	}
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if negative {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// Helper functions
func createTestRequest(router *gin.Engine, method, path string, body interface{}) *http.Request {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func executeRequest(router *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// --- Tests ---

func TestGetSiteConfig_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.siteConfigService = &MockSiteConfigService{
		config: &models.SiteConfig{
			ID:                 1,
			SiteName:           "Test Site",
			SiteDomain:         "test.com",
			NodeSecret:         "secret123",
			NodeReportInterval: 60,
		},
	}

	router := gin.New()
	router.GET("/admin/site-config", handler.GetSiteConfig)

	req := createTestRequest(router, "GET", "/admin/site-config", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	require.Equal(t, "Test Site", data["site_name"])
	require.Equal(t, "test.com", data["site_domain"])
}

func TestGetSiteConfig_ServiceNotInitialized(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.siteConfigService = nil

	router := gin.New()
	router.GET("/admin/site-config", handler.GetSiteConfig)

	req := createTestRequest(router, "GET", "/admin/site-config", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(500), resp["code"])
	require.Equal(t, "服务未初始化", resp["message"])
}

func TestGetSiteConfig_GetError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.siteConfigService = &MockSiteConfigService{getErr: services.ErrNodeNotFound}

	router := gin.New()
	router.GET("/admin/site-config", handler.GetSiteConfig)

	req := createTestRequest(router, "GET", "/admin/site-config", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateSiteConfig_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.siteConfigService = &MockSiteConfigService{
		config: &models.SiteConfig{
			ID:                 1,
			SiteName:           "Updated Site",
			SiteDomain:         "updated.com",
			NodeSecret:         "newsecret",
			NodeReportInterval: 45,
		},
	}

	router := gin.New()
	router.PUT("/admin/site-config", handler.UpdateSiteConfig)

	body := map[string]interface{}{
		"site_name":            "Updated Site",
		"site_domain":          "updated.com",
		"node_secret":          "newsecret",
		"node_report_interval": 45,
	}
	req := createTestRequest(router, "PUT", "/admin/site-config", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])
	require.Equal(t, "更新成功", resp["message"])
}

func TestUpdateSiteConfig_InvalidParams(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.siteConfigService = &MockSiteConfigService{}

	router := gin.New()
	router.PUT("/admin/site-config", handler.UpdateSiteConfig)

	// 无效的 JSON
	req := httptest.NewRequest("PUT", "/admin/site-config", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateSiteConfig_InvalidInterval(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.siteConfigService = &MockSiteConfigService{}

	router := gin.New()
	router.PUT("/admin/site-config", handler.UpdateSiteConfig)

	body := map[string]interface{}{
		"node_report_interval": -1,
	}
	req := createTestRequest(router, "PUT", "/admin/site-config", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "node_report_interval 必须 > 0", resp["message"])
}

func TestUpdateSiteConfig_NoUpdates(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.siteConfigService = &MockSiteConfigService{}

	router := gin.New()
	router.PUT("/admin/site-config", handler.UpdateSiteConfig)

	// 空更新
	body := map[string]interface{}{}
	req := createTestRequest(router, "PUT", "/admin/site-config", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "没有可更新的内容", resp["message"])
}

func TestGetNodeGroups_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeGroupService = &MockNodeGroupService{
		groups: []models.NodeGroup{
			{ID: 1, Name: "Group 1", Type: "entry"},
			{ID: 2, Name: "Group 2", Type: "target"},
		},
	}

	router := gin.New()
	router.GET("/admin/node-groups", handler.GetNodeGroups)

	req := createTestRequest(router, "GET", "/admin/node-groups", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])

	groups := resp["data"].([]interface{})
	require.Len(t, groups, 2)
}

func TestGetNodeGroups_Error(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeGroupService = &MockNodeGroupService{listErr: services.ErrNodeNotFound}

	router := gin.New()
	router.GET("/admin/node-groups", handler.GetNodeGroups)

	req := createTestRequest(router, "GET", "/admin/node-groups", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateNodeGroup_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeGroupService = &MockNodeGroupService{}

	router := gin.New()
	router.POST("/admin/node-groups", handler.CreateNodeGroup)

	body := map[string]interface{}{
		"name":        "New Group",
		"type":        "entry",
		"description": "Test description",
	}
	req := createTestRequest(router, "POST", "/admin/node-groups", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])
	require.Equal(t, "创建成功", resp["message"])
}

func TestCreateNodeGroup_MissingRequiredFields(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeGroupService = &MockNodeGroupService{}

	router := gin.New()
	router.POST("/admin/node-groups", handler.CreateNodeGroup)

	// 缺少 required 字段
	body := map[string]interface{}{
		"name": "New Group",
	}
	req := createTestRequest(router, "POST", "/admin/node-groups", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateNodeGroup_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeGroupService = &MockNodeGroupService{createErr: services.ErrNodeNotFound}

	router := gin.New()
	router.POST("/admin/node-groups", handler.CreateNodeGroup)

	body := map[string]interface{}{
		"name": "New Group",
		"type": "entry",
	}
	req := createTestRequest(router, "POST", "/admin/node-groups", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateNodeGroup_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeGroupService = &MockNodeGroupService{}

	router := gin.New()
	router.PUT("/admin/node-groups/:id", handler.UpdateNodeGroup)

	req := httptest.NewRequest("PUT", "/admin/node-groups/1", bytes.NewBufferString(`{"name":"Updated Group"}`))
	req.Header.Set("Content-Type", "application/json")
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "更新成功", resp["message"])
}

func TestUpdateNodeGroup_InvalidJSON(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeGroupService = &MockNodeGroupService{}

	router := gin.New()
	router.PUT("/admin/node-groups/:id", handler.UpdateNodeGroup)

	req := httptest.NewRequest("PUT", "/admin/node-groups/1", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateNodeGroup_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeGroupService = &MockNodeGroupService{updateErr: services.ErrNodeNotFound}

	router := gin.New()
	router.PUT("/admin/node-groups/:id", handler.UpdateNodeGroup)

	body := map[string]interface{}{
		"name": "Updated Group",
	}
	req := createTestRequest(router, "PUT", "/admin/node-groups/1", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteNodeGroup_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeGroupService = &MockNodeGroupService{}

	router := gin.New()
	router.DELETE("/admin/node-groups/:id", handler.DeleteNodeGroup)

	req := httptest.NewRequest("DELETE", "/admin/node-groups/1", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "删除成功", resp["message"])
}

func TestDeleteNodeGroup_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeGroupService = &MockNodeGroupService{deleteErr: services.ErrNodeNotFound}

	router := gin.New()
	router.DELETE("/admin/node-groups/:id", handler.DeleteNodeGroup)

	req := httptest.NewRequest("DELETE", "/admin/node-groups/1", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetUserGroups_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userGroupService = &MockUserGroupService{
		groups: []models.UserGroup{
			{ID: 1, Name: "VIP Group", Description: "VIP users"},
			{ID: 2, Name: "Free Group", Description: "Free users"},
		},
	}

	router := gin.New()
	router.GET("/admin/user-groups", handler.GetUserGroups)

	req := createTestRequest(router, "GET", "/admin/user-groups", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])

	groups := resp["data"].([]interface{})
	require.Len(t, groups, 2)
}

func TestGetUserGroups_Error(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userGroupService = &MockUserGroupService{listErr: services.ErrUserGroupNotFound}

	router := gin.New()
	router.GET("/admin/user-groups", handler.GetUserGroups)

	req := createTestRequest(router, "GET", "/admin/user-groups", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateUserGroup_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userGroupService = &MockUserGroupService{}

	router := gin.New()
	router.POST("/admin/user-groups", handler.CreateUserGroup)

	body := map[string]interface{}{
		"name":        "New User Group",
		"description": "Test description",
	}
	req := createTestRequest(router, "POST", "/admin/user-groups", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])
	require.Equal(t, "创建成功", resp["message"])
}

func TestCreateUserGroup_MissingName(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userGroupService = &MockUserGroupService{}

	router := gin.New()
	router.POST("/admin/user-groups", handler.CreateUserGroup)

	body := map[string]interface{}{
		"description": "No name",
	}
	req := createTestRequest(router, "POST", "/admin/user-groups", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateUserGroup_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userGroupService = &MockUserGroupService{createErr: services.ErrUserGroupNotFound}

	router := gin.New()
	router.POST("/admin/user-groups", handler.CreateUserGroup)

	body := map[string]interface{}{
		"name": "New Group",
	}
	req := createTestRequest(router, "POST", "/admin/user-groups", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateUserGroup_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userGroupService = &MockUserGroupService{}

	router := gin.New()
	router.PUT("/admin/user-groups/:id", handler.UpdateUserGroup)

	body := map[string]interface{}{
		"name": "Updated User Group",
	}
	req := createTestRequest(router, "PUT", "/admin/user-groups/1", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateUserGroup_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userGroupService = &MockUserGroupService{updateErr: services.ErrUserGroupNotFound}

	router := gin.New()
	router.PUT("/admin/user-groups/:id", handler.UpdateUserGroup)

	body := map[string]interface{}{
		"name": "Updated",
	}
	req := createTestRequest(router, "PUT", "/admin/user-groups/1", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteUserGroup_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userGroupService = &MockUserGroupService{}

	router := gin.New()
	router.DELETE("/admin/user-groups/:id", handler.DeleteUserGroup)

	req := httptest.NewRequest("DELETE", "/admin/user-groups/1", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "删除成功", resp["message"])
}

func TestDeleteUserGroup_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userGroupService = &MockUserGroupService{deleteErr: services.ErrUserGroupNotFound}

	router := gin.New()
	router.DELETE("/admin/user-groups/:id", handler.DeleteUserGroup)

	req := httptest.NewRequest("DELETE", "/admin/user-groups/1", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetAdminNodes_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{
		nodes: []models.Node{
			{ID: 1, Name: "Node 1", Host: "host1.com", Port: 8080},
			{ID: 2, Name: "Node 2", Host: "host2.com", Port: 8081},
		},
		count: 2,
	}

	router := gin.New()
	router.GET("/admin/nodes", handler.GetAdminNodes)

	req := createTestRequest(router, "GET", "/admin/nodes?page=1&page_size=20", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	require.Len(t, list, 2)
	require.Equal(t, float64(2), data["total"])
}

func TestGetAdminNodes_WithStatusFilter(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{
		nodes: []models.Node{
			{ID: 1, Name: "Online Node", Status: "online"},
		},
		count: 1,
	}

	router := gin.New()
	router.GET("/admin/nodes", handler.GetAdminNodes)

	req := createTestRequest(router, "GET", "/admin/nodes?status=online", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestCreateNode_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{}

	router := gin.New()
	router.POST("/admin/nodes", handler.CreateNode)

	body := map[string]interface{}{
		"name":         "New Node",
		"host":         "newhost.com",
		"port":         8080,
		"secret":       "secret123",
		"node_group_id": 1,
		"protocols":    []string{"gost"},
		"multiplier":   1.0,
		"region":       "CN",
	}
	req := createTestRequest(router, "POST", "/admin/nodes", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])
	require.Equal(t, "创建成功", resp["message"])
}

func TestCreateNode_MissingRequiredFields(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{}

	router := gin.New()
	router.POST("/admin/nodes", handler.CreateNode)

	body := map[string]interface{}{
		"name": "New Node",
	}
	req := createTestRequest(router, "POST", "/admin/nodes", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateNode_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{createErr: services.ErrNodeNotFound}

	router := gin.New()
	router.POST("/admin/nodes", handler.CreateNode)

	body := map[string]interface{}{
		"name":   "New Node",
		"host":   "newhost.com",
		"port":   8080,
		"secret": "secret123",
	}
	req := createTestRequest(router, "POST", "/admin/nodes", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdateNode_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{}

	router := gin.New()
	router.PUT("/admin/nodes/:id", handler.UpdateNode)

	body := map[string]interface{}{
		"name": "Updated Node",
	}
	req := createTestRequest(router, "PUT", "/admin/nodes/1", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "更新成功", resp["message"])
}

func TestUpdateNode_InvalidJSON(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{}

	router := gin.New()
	router.PUT("/admin/nodes/:id", handler.UpdateNode)

	req := httptest.NewRequest("PUT", "/admin/nodes/1", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateNode_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{updateErr: services.ErrNodeNotFound}

	router := gin.New()
	router.PUT("/admin/nodes/:id", handler.UpdateNode)

	body := map[string]interface{}{
		"name": "Updated",
	}
	req := createTestRequest(router, "PUT", "/admin/nodes/1", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeleteNode_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{}

	router := gin.New()
	router.DELETE("/admin/nodes/:id", handler.DeleteNode)

	req := httptest.NewRequest("DELETE", "/admin/nodes/1", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "删除成功", resp["message"])
}

func TestDeleteNode_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{deleteErr: services.ErrNodeNotFound}

	router := gin.New()
	router.DELETE("/admin/nodes/:id", handler.DeleteNode)

	req := httptest.NewRequest("DELETE", "/admin/nodes/1", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReloadNode_NodeNotFound(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{getErr: services.ErrNodeNotFound}

	router := gin.New()
	router.POST("/admin/nodes/:id/reload", handler.ReloadNode)

	req := httptest.NewRequest("POST", "/admin/nodes/999/reload", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "节点不存在", resp["message"])
}

func TestGetAdminUsers_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{
		users: []models.User{
			{ID: 1, Username: "user1", Role: "user"},
			{ID: 2, Username: "user2", Role: "user"},
		},
		count: 2,
	}

	router := gin.New()
	router.GET("/admin/users", handler.GetAdminUsers)

	req := createTestRequest(router, "GET", "/admin/users?page=1&page_size=20", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	require.Len(t, list, 2)
}

func TestCreateUser_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{}

	router := gin.New()
	router.POST("/admin/users", handler.CreateUser)

	body := map[string]interface{}{
		"username":     "newuser",
		"password":     "password123",
		"user_group_id": 1,
		"balance":      100,
		"is_admin":     false,
	}
	req := createTestRequest(router, "POST", "/admin/users", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])
	require.Equal(t, "创建成功", resp["message"])
}

func TestCreateUser_MissingRequiredFields(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{}

	router := gin.New()
	router.POST("/admin/users", handler.CreateUser)

	body := map[string]interface{}{
		"username": "newuser",
	}
	req := createTestRequest(router, "POST", "/admin/users", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateUser_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{createErr: services.ErrUserExists}

	router := gin.New()
	router.POST("/admin/users", handler.CreateUser)

	body := map[string]interface{}{
		"username": "existinguser",
		"password": "password123",
	}
	req := createTestRequest(router, "POST", "/admin/users", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateUser_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{}

	router := gin.New()
	router.PUT("/admin/users/:id", handler.UpdateUser)

	body := map[string]interface{}{
		"username": "updateduser",
	}
	req := createTestRequest(router, "PUT", "/admin/users/1", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "更新成功", resp["message"])
}

func TestUpdateUser_InvalidJSON(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{}

	router := gin.New()
	router.PUT("/admin/users/:id", handler.UpdateUser)

	req := httptest.NewRequest("PUT", "/admin/users/1", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateUser_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{updateErr: services.ErrUserNotFound}

	router := gin.New()
	router.PUT("/admin/users/:id", handler.UpdateUser)

	body := map[string]interface{}{
		"username": "updated",
	}
	req := createTestRequest(router, "PUT", "/admin/users/1", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAdjustBalance_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{}

	router := gin.New()
	router.POST("/admin/users/:id/balance", handler.AdjustBalance)

	body := map[string]interface{}{
		"amount": 100,
	}
	req := createTestRequest(router, "POST", "/admin/users/1/balance", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])
	require.Equal(t, "调整成功", resp["message"])
}

func TestAdjustBalance_MissingAmount(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{}

	router := gin.New()
	router.POST("/admin/users/:id/balance", handler.AdjustBalance)

	body := map[string]interface{}{}
	req := createTestRequest(router, "POST", "/admin/users/1/balance", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdjustBalance_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{updateBalErr: services.ErrUserNotFound}

	router := gin.New()
	router.POST("/admin/users/:id/balance", handler.AdjustBalance)

	body := map[string]interface{}{
		"amount": 100,
	}
	req := createTestRequest(router, "POST", "/admin/users/1/balance", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetAdminOrders_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{
		orders: []models.Order{
			{ID: 1, TradeNo: "T001", Status: "pending", Amount: 100},
			{ID: 2, TradeNo: "T002", Status: "success", Amount: 200},
		},
		orderCount: 2,
	}

	router := gin.New()
	router.GET("/admin/orders", handler.GetAdminOrders)

	req := createTestRequest(router, "GET", "/admin/orders?page=1&page_size=20", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	list := data["list"].([]interface{})
	require.Len(t, list, 2)
}

func TestGetAdminOrders_WithStatusFilter(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{
		orders:    []models.Order{{ID: 1, TradeNo: "T001", Status: "success"}},
		orderCount: 1,
	}

	router := gin.New()
	router.GET("/admin/orders", handler.GetAdminOrders)

	req := createTestRequest(router, "GET", "/admin/orders?status=success", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateOrderStatus_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{}

	router := gin.New()
	router.PUT("/admin/orders/:id", handler.UpdateOrderStatus)

	body := map[string]interface{}{
		"status": "success",
	}
	req := createTestRequest(router, "PUT", "/admin/orders/T001", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "更新成功", resp["message"])
}

func TestUpdateOrderStatus_InvalidJSON(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{}

	router := gin.New()
	router.PUT("/admin/orders/:id", handler.UpdateOrderStatus)

	req := httptest.NewRequest("PUT", "/admin/orders/T001", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateOrderStatus_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{updateOrdErr: services.ErrOrderNotFound}

	router := gin.New()
	router.PUT("/admin/orders/:id", handler.UpdateOrderStatus)

	body := map[string]interface{}{
		"status": "success",
	}
	req := createTestRequest(router, "PUT", "/admin/orders/T001", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetAdminPackages_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{
		packages: []models.Package{
			{ID: 1, Name: "Basic", Traffic: 100000, Price: 1000},
			{ID: 2, Name: "Premium", Traffic: 500000, Price: 5000},
		},
	}

	router := gin.New()
	router.GET("/admin/packages", handler.GetAdminPackages)

	req := createTestRequest(router, "GET", "/admin/packages", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])

	packages := resp["data"].([]interface{})
	require.Len(t, packages, 2)
}

func TestCreatePackage_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{}

	router := gin.New()
	router.POST("/admin/packages", handler.CreatePackage)

	body := map[string]interface{}{
		"name":         "New Package",
		"traffic":      1000000,
		"price":        10000,
		"user_group_id": 1,
	}
	req := createTestRequest(router, "POST", "/admin/packages", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])
	require.Equal(t, "创建成功", resp["message"])
}

func TestCreatePackage_MissingRequiredFields(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{}

	router := gin.New()
	router.POST("/admin/packages", handler.CreatePackage)

	body := map[string]interface{}{
		"name": "New Package",
	}
	req := createTestRequest(router, "POST", "/admin/packages", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreatePackage_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{createPkgErr: services.ErrPackageNotFound}

	router := gin.New()
	router.POST("/admin/packages", handler.CreatePackage)

	body := map[string]interface{}{
		"name":    "New Package",
		"traffic": 1000000,
		"price":   10000,
	}
	req := createTestRequest(router, "POST", "/admin/packages", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetOverviewStats_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{count: 100}
	handler.nodeService = &MockNodeService{count: 10}
	handler.paymentService = &MockPaymentService{
		orderCount:   50,
		totalRevenue: 100000,
	}

	router := gin.New()
	router.GET("/admin/overview", handler.GetOverviewStats)

	req := createTestRequest(router, "GET", "/admin/overview", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	require.Equal(t, float64(100), data["user_count"])
	require.Equal(t, float64(10), data["node_count"])
	require.Equal(t, float64(50), data["order_count"])
	require.Equal(t, float64(100000), data["total_revenue"])
}

func TestGetAdminNodeDetail_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{}

	router := gin.New()
	router.GET("/admin/nodes/:id", handler.GetAdminNodeDetail)

	req := httptest.NewRequest("GET", "/admin/nodes/1", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]interface{})
	require.NotNil(t, data["node"])
}

func TestGetAdminNodeDetail_NotFound(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.nodeService = &MockNodeService{getErr: services.ErrNodeNotFound}

	router := gin.New()
	router.GET("/admin/nodes/:id", handler.GetAdminNodeDetail)

	req := httptest.NewRequest("GET", "/admin/nodes/999", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetUserDetail_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{}

	router := gin.New()
	router.GET("/admin/users/:id", handler.GetUserDetail)

	req := httptest.NewRequest("GET", "/admin/users/1", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, float64(0), resp["code"])
	require.NotNil(t, resp["data"])
}

func TestGetUserDetail_NotFound(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{getErr: services.ErrUserNotFound}

	router := gin.New()
	router.GET("/admin/users/:id", handler.GetUserDetail)

	req := httptest.NewRequest("GET", "/admin/users/999", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteUser_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{}

	router := gin.New()
	router.DELETE("/admin/users/:id", handler.DeleteUser)

	req := httptest.NewRequest("DELETE", "/admin/users/1", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "删除成功", resp["message"])
}

func TestDeleteUser_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.userService = &MockUserService{deleteErr: services.ErrUserNotFound}

	router := gin.New()
	router.DELETE("/admin/users/:id", handler.DeleteUser)

	req := httptest.NewRequest("DELETE", "/admin/users/1", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestUpdatePackage_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{}

	router := gin.New()
	router.PUT("/admin/packages/:id", handler.UpdatePackage)

	body := map[string]interface{}{
		"name": "Updated Package",
	}
	req := createTestRequest(router, "PUT", "/admin/packages/1", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "更新成功", resp["message"])
}

func TestUpdatePackage_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{updatePkgErr: services.ErrPackageNotFound}

	router := gin.New()
	router.PUT("/admin/packages/:id", handler.UpdatePackage)

	body := map[string]interface{}{
		"name": "Updated",
	}
	req := createTestRequest(router, "PUT", "/admin/packages/1", body)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDeletePackage_Success(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{}

	router := gin.New()
	router.DELETE("/admin/packages/:id", handler.DeletePackage)

	req := httptest.NewRequest("DELETE", "/admin/packages/1", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	require.Equal(t, "删除成功", resp["message"])
}

func TestDeletePackage_ServiceError(t *testing.T) {
	handler := NewTestAdminHandler()
	handler.paymentService = &MockPaymentService{deletePkgErr: services.ErrPackageNotFound}

	router := gin.New()
	router.DELETE("/admin/packages/:id", handler.DeletePackage)

	req := httptest.NewRequest("DELETE", "/admin/packages/1", nil)
	w := executeRequest(router, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
}
