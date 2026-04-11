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
}

// NewRuleHandler 创建规则处理器
func NewRuleHandler(ruleService *services.RuleService, nodeService *services.NodeService) *RuleHandler {
	return &RuleHandler{
		ruleService: ruleService,
		nodeService: nodeService,
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

	log.Info("GetRules success", "count", len(rules), "total", total)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  rules,
			"total": total,
			"page":  page,
		},
	})
}

// CreateRuleRequest 创建规则请求
type CreateRuleRequest struct {
	Name         string          `json:"name" binding:"required"`
	NodeID       uint            `json:"node_id" binding:"required"`
	Protocol     string          `json:"protocol" binding:"required"`
	ListenPort   int             `json:"listen_port" binding:"required"`
	Enabled      *bool           `json:"enabled"`
	TrafficLimit *int64          `json:"traffic_limit"`
	SpeedLimit   *int64          `json:"speed_limit"`
	Mode         string          `json:"mode"`
	Targets      []TargetRequest `json:"targets" binding:"required,min=1"`
	GostConfig   *GostConfig     `json:"gost_config"`
}

// TargetRequest 目标请求
type TargetRequest struct {
	Host    string `json:"host" binding:"required"`
	Port    int    `json:"port" binding:"required"`
	Weight  int    `json:"weight"`
	Enabled bool   `json:"enabled"`
}

// GostConfig gost 配置
type GostConfig struct {
	Transport string `json:"transport"`
	TLS       bool   `json:"tls"`
	Chain     string `json:"chain"`
	Timeout   int    `json:"timeout"`
}

type normalizedRuleSpec struct {
	Protocol     string
	ListenPort   int
	Enabled      bool
	TrafficLimit int64
	SpeedLimit   int64
	Mode         string
	Targets      []TargetRequest
	GostConfig   *GostConfig
}

type existingRuleConflict struct {
	ID         uint
	ListenPort int
	Enabled    bool
	Layer4     string
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

	node, err := h.nodeService.GetNodeByID(req.NodeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "节点不存在"})
		return
	}

	conflicts, err := h.loadRuleConflicts(req.NodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "加载规则冲突信息失败"})
		return
	}

	enabledValue, trafficLimitValue, speedLimitValue := resolveRuleStateValues(req.Enabled, req.TrafficLimit, req.SpeedLimit, true, 0, 0)

	spec, err := normalizeAndValidateRuleSpec(node, req.Protocol, req.ListenPort, enabledValue, trafficLimitValue, speedLimitValue, req.Mode, req.Targets, req.GostConfig, conflicts, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	rule := &models.ForwardingRule{
		NodeID:       req.NodeID,
		UserID:       userID,
		Name:         req.Name,
		Protocol:     spec.Protocol,
		ListenPort:   spec.ListenPort,
		Mode:         spec.Mode,
		Enabled:      spec.Enabled,
		TrafficUsed:  0,
		TrafficLimit: spec.TrafficLimit,
		SpeedLimit:   spec.SpeedLimit,
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

	gostRule := &models.GostRule{
		RuleID:    rule.ID,
		Transport: spec.GostConfig.Transport,
		TLS:       spec.GostConfig.TLS,
		Chain:     spec.GostConfig.Chain,
		Timeout:   spec.GostConfig.Timeout,
	}
	_ = h.ruleService.CreateGostRule(gostRule)

	triggerNodeReloadAsync(h.nodeService, requestID, rule.NodeID)

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

	var gostRule *models.GostRule
	if rule.Protocol == "gost" {
		gostRule, _ = h.ruleService.GetGostRule(rule.ID)
	}

	log.Info("GetRule success", "rule_id", id, "rule_name", rule.Name)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"rule":        rule,
			"targets":     targets,
			"gost_config": gostRule,
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
	_ = h.ruleService.DeleteGostRule(uint(id))
	triggerNodeReloadAsync(h.nodeService, requestID, rule.NodeID)

	log.Info("DeleteRule success", "rule_id", id)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name         string          `json:"name"`
	Enabled      *bool           `json:"enabled"`
	NodeID       *uint           `json:"node_id"`
	Protocol     string          `json:"protocol"`
	ListenPort   *int            `json:"listen_port"`
	TrafficLimit *int64          `json:"traffic_limit"`
	SpeedLimit   *int64          `json:"speed_limit"`
	Mode         string          `json:"mode"`
	Targets      []TargetRequest `json:"targets"`
	GostConfig   *GostConfig     `json:"gost_config"`
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
	existingGost, _ := h.ruleService.GetGostRule(rule.ID)

	nodeID := rule.NodeID
	if req.NodeID != nil && *req.NodeID > 0 {
		nodeID = *req.NodeID
	}
	node, err := h.nodeService.GetNodeByID(nodeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "节点不存在"})
		return
	}

	conflicts, err := h.loadRuleConflicts(nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "加载规则冲突信息失败"})
		return
	}

	enabledValue, trafficLimitValue, speedLimitValue := resolveRuleStateValues(req.Enabled, req.TrafficLimit, req.SpeedLimit, rule.Enabled, rule.TrafficLimit, rule.SpeedLimit)

	spec, err := normalizeAndValidateRuleSpec(
		node,
		coalesceString(req.Protocol, rule.Protocol),
		valueOrDefaultInt(req.ListenPort, rule.ListenPort),
		enabledValue,
		trafficLimitValue,
		speedLimitValue,
		coalesceString(req.Mode, rule.Mode),
		coalesceTargets(req.Targets, targets),
		coalesceGostConfig(req.GostConfig, existingGost),
		conflicts,
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

	cfgUpdates := map[string]interface{}{
		"transport": spec.GostConfig.Transport,
		"tls":       spec.GostConfig.TLS,
		"chain":     spec.GostConfig.Chain,
		"timeout":   spec.GostConfig.Timeout,
	}
	if err := h.ruleService.UpsertGostRule(uint(id), cfgUpdates); err != nil {
		logger.Error("UpdateRule: upsert gost config failed", err, "rule_id", id, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新协议配置失败"})
		return
	}

	triggerNodeReloadAsync(h.nodeService, requestID, rule.NodeID, nodeID)

	log.Info("UpdateRule success", "rule_id", id, "rule_name", rule.Name)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
}

func normalizeAndValidateRuleSpec(node *models.Node, protocol string, listenPort int, enabled bool, trafficLimit int64, speedLimit int64, mode string, targets []TargetRequest, gostConfig *GostConfig, existingRules []existingRuleConflict, currentRuleID uint) (*normalizedRuleSpec, error) {
	spec := &normalizedRuleSpec{
		Protocol:     strings.ToLower(strings.TrimSpace(protocol)),
		ListenPort:   listenPort,
		Enabled:      enabled,
		TrafficLimit: maxInt64(0, trafficLimit),
		SpeedLimit:   maxInt64(0, speedLimit),
		Mode:         strings.ToLower(strings.TrimSpace(mode)),
		Targets:      sanitizeTargets(targets),
		GostConfig:   normalizeGostConfig(gostConfig),
	}

	if spec.Mode == "" {
		spec.Mode = "direct"
	}
	if spec.ListenPort <= 0 || spec.ListenPort > 65535 {
		return nil, fmt.Errorf("监听端口必须在 1-65535 之间")
	}

	if spec.Protocol != "gost" {
		return nil, fmt.Errorf("仅支持 gost")
	}

	if !services.NodeSupportsProtocol([]string(node.Protocols), spec.Protocol) {
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
		return nil, fmt.Errorf("gost 规则暂不支持限速")
	}
	if spec.GostConfig.TLS || strings.TrimSpace(spec.GostConfig.Chain) != "" {
		return nil, fmt.Errorf("当前 gost 首版仅支持 TCP/UDP 端口转发")
	}

	layer4 := layer4Protocol(spec.GostConfig)
	for _, existing := range existingRules {
		if existing.ID == currentRuleID || !existing.Enabled {
			continue
		}
		if existing.ListenPort != spec.ListenPort {
			continue
		}

		if existing.Layer4 == layer4 {
			return nil, fmt.Errorf("该节点端口 %d 的 %s 转发规则已存在", spec.ListenPort, strings.ToUpper(layer4))
		}
	}

	return spec, nil
}

func (h *RuleHandler) loadRuleConflicts(nodeID uint) ([]existingRuleConflict, error) {
	rules, err := h.ruleService.ListRulesByNode(nodeID, false)
	if err != nil {
		return nil, err
	}

	out := make([]existingRuleConflict, 0, len(rules))
	for _, rule := range rules {
		if rule.Protocol != "gost" {
			continue
		}
		layer4 := "tcp"
		cfg, err := h.ruleService.GetGostRule(rule.ID)
		if err != nil {
			return nil, err
		}
		if cfg != nil && strings.EqualFold(strings.TrimSpace(cfg.Transport), "udp") {
			layer4 = "udp"
		}

		out = append(out, existingRuleConflict{
			ID:         rule.ID,
			ListenPort: rule.ListenPort,
			Enabled:    rule.Enabled,
			Layer4:     layer4,
		})
	}

	return out, nil
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

func normalizeGostConfig(cfg *GostConfig) *GostConfig {
	if cfg == nil {
		return &GostConfig{Transport: "tcp"}
	}
	out := *cfg
	out.Transport = strings.ToLower(strings.TrimSpace(out.Transport))
	if out.Transport == "" {
		out.Transport = "tcp"
	}
	if out.Transport != "tcp" && out.Transport != "udp" {
		out.Transport = "tcp"
	}
	if out.Timeout < 0 {
		out.Timeout = 0
	}
	out.Chain = strings.TrimSpace(out.Chain)
	return &out
}

func layer4Protocol(gostConfig *GostConfig) string {
	return normalizeGostConfig(gostConfig).Transport
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

func coalesceGostConfig(incoming *GostConfig, existing *models.GostRule) *GostConfig {
	if incoming != nil {
		return incoming
	}
	if existing == nil {
		return nil
	}
	return &GostConfig{
		Transport: existing.Transport,
		TLS:       existing.TLS,
		Chain:     existing.Chain,
		Timeout:   existing.Timeout,
	}
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
