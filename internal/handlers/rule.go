package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"bakaray/internal/logger"
	"bakaray/internal/middleware"
	"bakaray/internal/models"
	"bakaray/internal/services"

	"github.com/gin-gonic/gin"
)

// RuleHandler 转发规则处理器
type RuleHandler struct {
	ruleService *services.RuleService
	nodeService *services.NodeService
	userService *services.UserService
}

// NewRuleHandler 创建规则处理器
func NewRuleHandler(ruleService *services.RuleService, nodeService *services.NodeService, userService *services.UserService) *RuleHandler {
	return &RuleHandler{
		ruleService: ruleService,
		nodeService: nodeService,
		userService: userService,
	}
}

// GetRules 获取我的规则列表
func (h *RuleHandler) GetRules(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "rule")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	log.Debug("GetRules request", "page", page, "page_size", pageSize)

	rules, total := h.ruleService.ListRulesByUser(userID, page, pageSize)
	items := make([]models.ForwardingRule, 0, len(rules))
	for _, rule := range rules {
		ruleView := rule
		ruleView.Protocol = services.NormalizeDirectProtocol(rule.Protocol)
		items = append(items, ruleView)
	}

	log.Info("GetRules success", "count", len(items), "total", total)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  items,
			"total": total,
			"page":  page,
		},
	})
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	Name           string          `json:"name" binding:"required"`
	NodeID         uint            `json:"node_id" binding:"required"`
	Protocol       string          `json:"protocol" binding:"required"`
	ListenPort     int             `json:"listen_port" binding:"required"`
	Enabled        *bool           `json:"enabled"`
	TrafficLimit   *int64          `json:"traffic_limit"`
	SpeedLimit     *int64          `json:"speed_limit"`
	Mode           string          `json:"mode"`
	Targets        []TargetRequest `json:"targets" binding:"required,min=1"`
	TunnelEnabled  bool            `json:"tunnel_enabled"`
	ExitNodeID     uint            `json:"exit_node_id"`
	TunnelProtocol string          `json:"tunnel_protocol"`
	TunnelPort     int             `json:"tunnel_port"`
}

// TargetRequest 目标请求
type TargetRequest struct {
	Host    string `json:"host" binding:"required"`
	Port    int    `json:"port" binding:"required"`
	Weight  int    `json:"weight"`
	Enabled bool   `json:"enabled"`
}

type normalizedRuleSpec struct {
	Protocol       string
	ListenPort     int
	Enabled        bool
	TrafficLimit   int64
	SpeedLimit     int64
	Mode           string
	Targets        []TargetRequest
	TunnelEnabled  bool
	ExitNodeID     uint
	TunnelProtocol string
	TunnelPort     int
}

type existingRuleConflict struct {
	ID      uint
	Port    int
	Enabled bool
	Layer4  string
}

// CreateRule 创建规则
func (h *RuleHandler) CreateRule(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "rule")

	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("CreateRule: invalid request", "error", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	log.Debug("CreateRule request", "name", req.Name, "protocol", req.Protocol, "node_id", req.NodeID)

	entryNode, err := h.nodeService.GetNodeByID(req.NodeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "节点不存在"})
		return
	}

	allowed, err := h.userCanUseNode(userID, req.NodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取节点授权失败"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "当前用户组无权使用该节点"})
		return
	}

	var exitNode *models.Node
	if req.TunnelEnabled {
		if req.ExitNodeID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "启用隧道时必须选择出口节点"})
			return
		}

		exitNode, err = h.nodeService.GetNodeByID(req.ExitNodeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "出口节点不存在"})
			return
		}

		allowed, err = h.userCanUseNode(userID, req.ExitNodeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取出口节点授权失败"})
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "当前用户组无权使用出口节点"})
			return
		}
	}

	entryConflicts, err := h.loadRuleConflicts(req.NodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "加载规则冲突信息失败"})
		return
	}
	exitConflicts := []existingRuleConflict(nil)
	if req.TunnelEnabled {
		exitConflicts, err = h.loadRuleConflicts(req.ExitNodeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "加载出口节点冲突信息失败"})
			return
		}
	}

	enabledValue, trafficLimitValue, speedLimitValue := resolveRuleStateValues(req.Enabled, req.TrafficLimit, req.SpeedLimit, true, 0, 0)

	spec, err := normalizeAndValidateRuleSpec(
		entryNode,
		exitNode,
		req.Protocol,
		req.ListenPort,
		enabledValue,
		trafficLimitValue,
		speedLimitValue,
		req.Mode,
		req.Targets,
		req.TunnelEnabled,
		req.ExitNodeID,
		req.TunnelProtocol,
		req.TunnelPort,
		entryConflicts,
		exitConflicts,
		0,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	rule := &models.ForwardingRule{
		NodeID:         req.NodeID,
		UserID:         userID,
		Name:           req.Name,
		Protocol:       spec.Protocol,
		ListenPort:     spec.ListenPort,
		Mode:           spec.Mode,
		Enabled:        spec.Enabled,
		TrafficUsed:    0,
		TrafficLimit:   spec.TrafficLimit,
		SpeedLimit:     spec.SpeedLimit,
		TunnelEnabled:  spec.TunnelEnabled,
		ExitNodeID:     spec.ExitNodeID,
		TunnelProtocol: spec.TunnelProtocol,
		TunnelPort:     spec.TunnelPort,
	}

	if err := h.ruleService.CreateRule(rule); err != nil {
		logger.Error("CreateRule: create rule failed", err, "user_id", userID, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建规则失败"})
		return
	}

	for _, t := range spec.Targets {
		target := &models.Target{
			RuleID:  rule.ID,
			Host:    t.Host,
			Port:    t.Port,
			Weight:  t.Weight,
			Enabled: t.Enabled,
		}
		h.ruleService.AddTarget(target)
	}

	log.Info("CreateRule success", "rule_id", rule.ID, "rule_name", rule.Name)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "创建成功",
		"data": gin.H{
			"id": rule.ID,
		},
	})
}

// GetRule 获取规则详情
func (h *RuleHandler) GetRule(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "rule")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	log.Debug("GetRule request", "rule_id", id)

	rule, err := h.ruleService.GetRuleByID(uint(id))
	if err != nil {
		logger.Error("GetRule: rule not found", err, "rule_id", id, "request_id", requestID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}
	if rule.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权访问此规则"})
		return
	}

	targets, _ := h.ruleService.ListTargets(rule.ID, false)

	ruleView := *rule
	ruleView.Protocol = services.NormalizeDirectProtocol(rule.Protocol)

	log.Info("GetRule success", "rule_id", id, "rule_name", rule.Name)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"rule":    ruleView,
			"targets": targets,
		},
	})
}

// DeleteRule 删除规则
func (h *RuleHandler) DeleteRule(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "rule")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	log.Debug("DeleteRule request", "rule_id", id)

	rule, err := h.ruleService.GetRuleByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}
	if rule.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权操作此规则"})
		return
	}

	if err := h.ruleService.DeleteRule(uint(id)); err != nil {
		logger.Error("DeleteRule: delete failed", err, "rule_id", id, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}
	_ = h.ruleService.DeleteTargetsByRuleID(uint(id))

	log.Info("DeleteRule success", "rule_id", id)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name           string          `json:"name"`
	Enabled        *bool           `json:"enabled"`
	NodeID         *uint           `json:"node_id"`
	Protocol       string          `json:"protocol"`
	ListenPort     *int            `json:"listen_port"`
	TrafficLimit   *int64          `json:"traffic_limit"`
	SpeedLimit     *int64          `json:"speed_limit"`
	Mode           string          `json:"mode"`
	Targets        []TargetRequest `json:"targets"`
	TunnelEnabled  *bool           `json:"tunnel_enabled"`
	ExitNodeID     *uint           `json:"exit_node_id"`
	TunnelProtocol string          `json:"tunnel_protocol"`
	TunnelPort     *int            `json:"tunnel_port"`
}

// UpdateRule 更新规则
func (h *RuleHandler) UpdateRule(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.WithContext(requestID, userID, "rule")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("UpdateRule: invalid request", "error", err, "request_id", requestID, "user_id", userID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	log.Debug("UpdateRule request", "rule_id", id)

	rule, err := h.ruleService.GetRuleByID(uint(id))
	if err != nil {
		logger.Error("UpdateRule: rule not found", err, "rule_id", id, "request_id", requestID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}
	if rule.UserID != userID {
		logger.Warn("UpdateRule: permission denied", "rule_id", id, "user_id", userID, "rule_owner_id", rule.UserID, "request_id", requestID)
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权操作此规则"})
		return
	}

	targets, _ := h.ruleService.ListTargets(rule.ID, false)

	nodeID := rule.NodeID
	if req.NodeID != nil && *req.NodeID > 0 {
		nodeID = *req.NodeID
	}
	entryNode, err := h.nodeService.GetNodeByID(nodeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "节点不存在"})
		return
	}

	allowed, err := h.userCanUseNode(userID, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取节点授权失败"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "当前用户组无权使用该节点"})
		return
	}

	tunnelEnabled := rule.TunnelEnabled
	if req.TunnelEnabled != nil {
		tunnelEnabled = *req.TunnelEnabled
	}

	exitNodeID := rule.ExitNodeID
	if req.ExitNodeID != nil {
		exitNodeID = *req.ExitNodeID
	}

	tunnelProtocol := coalesceString(req.TunnelProtocol, rule.TunnelProtocol)
	tunnelPort := valueOrDefaultInt(req.TunnelPort, rule.TunnelPort)

	var exitNode *models.Node
	if tunnelEnabled {
		if exitNodeID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "启用隧道时必须选择出口节点"})
			return
		}

		exitNode, err = h.nodeService.GetNodeByID(exitNodeID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "出口节点不存在"})
			return
		}

		allowed, err = h.userCanUseNode(userID, exitNodeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取出口节点授权失败"})
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "当前用户组无权使用出口节点"})
			return
		}
	}

	entryConflicts, err := h.loadRuleConflicts(nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "加载规则冲突信息失败"})
		return
	}
	exitConflicts := []existingRuleConflict(nil)
	if tunnelEnabled {
		exitConflicts, err = h.loadRuleConflicts(exitNodeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "加载出口节点冲突信息失败"})
			return
		}
	}

	enabledValue, trafficLimitValue, speedLimitValue := resolveRuleStateValues(req.Enabled, req.TrafficLimit, req.SpeedLimit, rule.Enabled, rule.TrafficLimit, rule.SpeedLimit)

	spec, err := normalizeAndValidateRuleSpec(
		entryNode,
		exitNode,
		coalesceString(req.Protocol, services.NormalizeDirectProtocol(rule.Protocol)),
		valueOrDefaultInt(req.ListenPort, rule.ListenPort),
		enabledValue,
		trafficLimitValue,
		speedLimitValue,
		coalesceString(req.Mode, rule.Mode),
		coalesceTargets(req.Targets, targets),
		tunnelEnabled,
		exitNodeID,
		tunnelProtocol,
		tunnelPort,
		entryConflicts,
		exitConflicts,
		rule.ID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	updates["enabled"] = spec.Enabled
	updates["traffic_limit"] = spec.TrafficLimit
	updates["speed_limit"] = spec.SpeedLimit
	updates["mode"] = spec.Mode
	updates["node_id"] = nodeID
	updates["protocol"] = spec.Protocol
	updates["listen_port"] = spec.ListenPort
	updates["tunnel_enabled"] = spec.TunnelEnabled
	updates["exit_node_id"] = spec.ExitNodeID
	updates["tunnel_protocol"] = spec.TunnelProtocol
	updates["tunnel_port"] = spec.TunnelPort

	if err := h.ruleService.UpdateRule(uint(id), updates); err != nil {
		logger.Error("UpdateRule: update failed", err, "rule_id", id, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	_ = h.ruleService.DeleteTargetsByRuleID(uint(id))
	for _, t := range spec.Targets {
		target := &models.Target{
			RuleID:  uint(id),
			Host:    t.Host,
			Port:    t.Port,
			Weight:  t.Weight,
			Enabled: t.Enabled,
		}
		_ = h.ruleService.AddTarget(target)
	}

	log.Info("UpdateRule success", "rule_id", id, "rule_name", rule.Name)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

func normalizeAndValidateRuleSpec(entryNode *models.Node, exitNode *models.Node, protocol string, listenPort int, enabled bool, trafficLimit int64, speedLimit int64, mode string, targets []TargetRequest, tunnelEnabled bool, exitNodeID uint, tunnelProtocol string, tunnelPort int, entryRules []existingRuleConflict, exitRules []existingRuleConflict, currentRuleID uint) (*normalizedRuleSpec, error) {
	spec := &normalizedRuleSpec{
		Protocol:       services.NormalizeDirectProtocol(protocol),
		ListenPort:     listenPort,
		Enabled:        enabled,
		TrafficLimit:   maxInt64(0, trafficLimit),
		SpeedLimit:     maxInt64(0, speedLimit),
		Mode:           strings.ToLower(strings.TrimSpace(mode)),
		Targets:        sanitizeTargets(targets),
		TunnelEnabled:  tunnelEnabled,
		ExitNodeID:     exitNodeID,
		TunnelProtocol: services.NormalizeTunnelProtocol(tunnelProtocol),
		TunnelPort:     tunnelPort,
	}

	if spec.Mode == "" {
		spec.Mode = "direct"
	}
	if spec.ListenPort <= 0 || spec.ListenPort > 65535 {
		return nil, fmt.Errorf("监听端口必须在 1-65535 之间")
	}

	if !services.IsDirectProtocol(spec.Protocol) {
		return nil, fmt.Errorf("直接转发协议仅支持 TCP 或 UDP")
	}
	if !services.NodeSupportsDirectProtocol([]string(entryNode.Protocols), spec.Protocol) {
		return nil, fmt.Errorf("节点未声明支持 %s", spec.Protocol)
	}

	switch spec.Mode {
	case "direct", "rr", "lb":
	default:
		return nil, fmt.Errorf("仅支持 direct、rr 或 lb")
	}

	enabledTargets := 0
	for _, target := range spec.Targets {
		if target.Enabled {
			enabledTargets++
		}
	}
	if enabledTargets == 0 {
		return nil, fmt.Errorf("至少需要一个启用目标")
	}
	if spec.Mode == "direct" && enabledTargets != 1 {
		return nil, fmt.Errorf("direct 模式必须且只能有一个启用目标")
	}
	if spec.Mode != "direct" && enabledTargets < 2 {
		return nil, fmt.Errorf("%s 模式至少需要两个启用目标", spec.Mode)
	}

	if spec.SpeedLimit > 0 {
		return nil, fmt.Errorf("当前规则暂不支持限速")
	}
	entryLayer4 := services.DirectProtocolNetwork(spec.Protocol)
	for _, existing := range entryRules {
		if existing.ID == currentRuleID || !existing.Enabled {
			continue
		}
		if existing.Port != spec.ListenPort {
			continue
		}
		if existing.Layer4 == entryLayer4 {
			return nil, fmt.Errorf("该节点端口 %d 的 %s 监听已存在", spec.ListenPort, strings.ToUpper(entryLayer4))
		}
	}

	if !spec.TunnelEnabled {
		spec.ExitNodeID = 0
		spec.TunnelProtocol = ""
		spec.TunnelPort = 0
		return spec, nil
	}

	if exitNode == nil {
		return nil, fmt.Errorf("启用隧道时必须选择出口节点")
	}
	if entryNode.ID == exitNode.ID {
		return nil, fmt.Errorf("入口节点与出口节点不能相同")
	}
	if !services.IsTunnelProtocol(spec.TunnelProtocol) {
		return nil, fmt.Errorf("不支持的隧道协议")
	}
	if spec.TunnelPort <= 0 || spec.TunnelPort > 65535 {
		return nil, fmt.Errorf("隧道端口必须在 1-65535 之间")
	}
	if !services.NodeSupportsTunnelProtocol([]string(entryNode.Protocols), spec.TunnelProtocol) {
		return nil, fmt.Errorf("入口节点未声明支持 %s 隧道", spec.TunnelProtocol)
	}
	if !services.NodeSupportsTunnelProtocol([]string(exitNode.Protocols), spec.TunnelProtocol) {
		return nil, fmt.Errorf("出口节点未声明支持 %s 隧道", spec.TunnelProtocol)
	}

	exitLayer4 := services.TunnelProtocolNetwork(spec.TunnelProtocol)
	for _, existing := range exitRules {
		if existing.ID == currentRuleID || !existing.Enabled {
			continue
		}
		if existing.Port != spec.TunnelPort {
			continue
		}
		if existing.Layer4 == exitLayer4 {
			return nil, fmt.Errorf("出口节点端口 %d 的 %s 监听已存在", spec.TunnelPort, strings.ToUpper(exitLayer4))
		}
	}

	return spec, nil
}

func (h *RuleHandler) loadRuleConflicts(nodeID uint) ([]existingRuleConflict, error) {
	entryRules, err := h.ruleService.ListRulesByNode(nodeID, false)
	if err != nil {
		return nil, err
	}
	exitRules, err := h.ruleService.ListRulesByExitNode(nodeID, false)
	if err != nil {
		return nil, err
	}

	out := make([]existingRuleConflict, 0, len(entryRules)+len(exitRules))
	for _, rule := range entryRules {
		ruleProtocol := services.NormalizeDirectProtocol(rule.Protocol)
		if !services.IsDirectProtocol(ruleProtocol) {
			continue
		}
		out = append(out, existingRuleConflict{
			ID:      rule.ID,
			Port:    rule.ListenPort,
			Enabled: rule.Enabled,
			Layer4:  services.DirectProtocolNetwork(ruleProtocol),
		})
	}
	for _, rule := range exitRules {
		if !rule.TunnelEnabled || !services.IsTunnelProtocol(rule.TunnelProtocol) {
			continue
		}
		out = append(out, existingRuleConflict{
			ID:      rule.ID,
			Port:    rule.TunnelPort,
			Enabled: rule.Enabled,
			Layer4:  services.TunnelProtocolNetwork(rule.TunnelProtocol),
		})
	}

	return out, nil
}

func (h *RuleHandler) userCanUseNode(userID uint, nodeID uint) (bool, error) {
	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		return false, err
	}

	groupIDs, err := h.nodeService.GetAllowedGroups(nodeID)
	if err != nil {
		return false, err
	}

	for _, groupID := range groupIDs {
		if groupID == user.UserGroupID {
			return true, nil
		}
	}
	return false, nil
}

func sanitizeTargets(targets []TargetRequest) []TargetRequest {
	out := make([]TargetRequest, 0, len(targets))
	for _, target := range targets {
		host := strings.TrimSpace(target.Host)
		if host == "" || target.Port <= 0 || target.Port > 65535 {
			continue
		}
		weight := target.Weight
		if weight <= 0 {
			weight = 1
		}
		out = append(out, TargetRequest{
			Host:    host,
			Port:    target.Port,
			Weight:  weight,
			Enabled: target.Enabled,
		})
	}
	return out
}

func coalesceString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func resolveRuleStateValues(enabled *bool, trafficLimit *int64, speedLimit *int64, fallbackEnabled bool, fallbackTrafficLimit int64, fallbackSpeedLimit int64) (bool, int64, int64) {
	return valueOrDefaultBool(enabled, fallbackEnabled), valueOrDefaultInt64(trafficLimit, fallbackTrafficLimit), valueOrDefaultInt64(speedLimit, fallbackSpeedLimit)
}

func valueOrDefaultBool(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func valueOrDefaultInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func valueOrDefaultInt64(value *int64, fallback int64) int64 {
	if value == nil {
		return fallback
	}
	return *value
}

func coalesceTargets(incoming []TargetRequest, existing []models.Target) []TargetRequest {
	if incoming != nil {
		return incoming
	}
	out := make([]TargetRequest, 0, len(existing))
	for _, target := range existing {
		out = append(out, TargetRequest{
			Host:    target.Host,
			Port:    target.Port,
			Weight:  target.Weight,
			Enabled: target.Enabled,
		})
	}
	return out
}

func maxInt64(minimum, value int64) int64 {
	if value < minimum {
		return minimum
	}
	return value
}

// CountRules 统计规则数量（管理员用）
func (h *RuleHandler) CountRules(c *gin.Context) {
	total := h.ruleService.CountAllRules()
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"total": total,
		},
	})
}
