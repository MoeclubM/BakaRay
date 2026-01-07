package handlers

import (
	"net/http"
	"strconv"

	"bakaray/internal/logger"
	"bakaray/internal/middleware"
	"bakaray/internal/models"
	"bakaray/internal/services"

	"github.com/gin-gonic/gin"
)

// RuleHandler 转发规则处理器
type RuleHandler struct {
	ruleService *services.RuleService
}

// NewRuleHandler 创建规则处理器
func NewRuleHandler(ruleService *services.RuleService) *RuleHandler {
	return &RuleHandler{ruleService: ruleService}
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
	Name           string          `json:"name" binding:"required"`
	NodeID         uint            `json:"node_id" binding:"required"`
	Protocol       string          `json:"protocol" binding:"required"`
	ListenPort     int             `json:"listen_port" binding:"required"`
	Mode           string          `json:"mode"`
	Targets        []TargetRequest `json:"targets" binding:"required,min=1"`
	GostConfig     *GostConfig     `json:"gost_config"`
	IPTablesConfig *IPTablesConfig `json:"iptables_config"`
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

// IPTablesConfig iptables 配置
type IPTablesConfig struct {
	Proto string `json:"proto"`
	SNAT  bool   `json:"snat"`
	Iface string `json:"iface"`
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

	mode := req.Mode
	if mode == "" {
		mode = "direct"
	}

	rule := &models.ForwardingRule{
		NodeID:       req.NodeID,
		UserID:       userID,
		Name:         req.Name,
		Protocol:     req.Protocol,
		ListenPort:   req.ListenPort,
		Mode:         mode,
		Enabled:      true,
		TrafficUsed:  0,
		TrafficLimit: 0,
		SpeedLimit:   0,
	}

	if err := h.ruleService.CreateRule(rule); err != nil {
		logger.Error("CreateRule: create rule failed", err, "user_id", userID, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建规则失败"})
		return
	}

	for _, t := range req.Targets {
		target := &models.Target{
			RuleID:  rule.ID,
			Host:    t.Host,
			Port:    t.Port,
			Weight:  t.Weight,
			Enabled: t.Enabled,
		}
		h.ruleService.AddTarget(target)
	}

	switch req.Protocol {
	case "gost":
		if req.GostConfig != nil {
			gostRule := &models.GostRule{
				RuleID:    rule.ID,
				Transport: req.GostConfig.Transport,
				TLS:       req.GostConfig.TLS,
				Chain:     req.GostConfig.Chain,
				Timeout:   req.GostConfig.Timeout,
			}
			h.ruleService.CreateGostRule(gostRule)
		}
	case "iptables":
		if req.IPTablesConfig != nil {
			iptRule := &models.IPTablesRule{
				RuleID: rule.ID,
				Proto:  req.IPTablesConfig.Proto,
				SNAT:   req.IPTablesConfig.SNAT,
				Iface:  req.IPTablesConfig.Iface,
			}
			h.ruleService.CreateIPTablesRule(iptRule)
		}
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

	targets, _ := h.ruleService.GetTargets(rule.ID)

	var gostRule *models.GostRule
	var iptRule *models.IPTablesRule

	switch rule.Protocol {
	case "gost":
		gostRule, _ = h.ruleService.GetGostRule(rule.ID)
	case "iptables":
		iptRule, _ = h.ruleService.GetIPTablesRule(rule.ID)
	}

	log.Info("GetRule success", "rule_id", id, "rule_name", rule.Name)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"rule":            rule,
			"targets":         targets,
			"gost_config":     gostRule,
			"iptables_config": iptRule,
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

	if err := h.ruleService.DeleteRule(uint(id)); err != nil {
		logger.Error("DeleteRule: delete failed", err, "rule_id", id, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

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
	TrafficLimit   *int64          `json:"traffic_limit"`
	SpeedLimit     *int64          `json:"speed_limit"`
	Mode           string          `json:"mode"`
	Targets        []TargetRequest `json:"targets"`
	GostConfig     *GostConfig     `json:"gost_config"`
	IPTablesConfig *IPTablesConfig `json:"iptables_config"`
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

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.TrafficLimit != nil {
		updates["traffic_limit"] = *req.TrafficLimit
	}
	if req.SpeedLimit != nil {
		updates["speed_limit"] = *req.SpeedLimit
	}
	if req.Mode != "" {
		updates["mode"] = req.Mode
	}
	if req.NodeID != nil && *req.NodeID > 0 {
		updates["node_id"] = *req.NodeID
	}

	if err := h.ruleService.UpdateRule(uint(id), updates); err != nil {
		logger.Error("UpdateRule: update failed", err, "rule_id", id, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	if req.Targets != nil {
		if len(req.Targets) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "至少需要一个转发目标"})
			return
		}
		_ = h.ruleService.DeleteTargetsByRuleID(uint(id))
		for _, t := range req.Targets {
			target := &models.Target{
				RuleID:  uint(id),
				Host:    t.Host,
				Port:    t.Port,
				Weight:  t.Weight,
				Enabled: t.Enabled,
			}
			_ = h.ruleService.AddTarget(target)
		}
	}

	if req.GostConfig != nil {
		cfgUpdates := map[string]interface{}{
			"transport": req.GostConfig.Transport,
			"tls":       req.GostConfig.TLS,
			"chain":     req.GostConfig.Chain,
			"timeout":   req.GostConfig.Timeout,
		}
		if err := h.ruleService.UpsertGostRule(uint(id), cfgUpdates); err != nil {
			logger.Error("UpdateRule: upsert gost config failed", err, "rule_id", id, "request_id", requestID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新协议配置失败"})
			return
		}
	}

	if req.IPTablesConfig != nil {
		cfgUpdates := map[string]interface{}{
			"proto": req.IPTablesConfig.Proto,
			"snat":  req.IPTablesConfig.SNAT,
			"iface": req.IPTablesConfig.Iface,
		}
		if err := h.ruleService.UpsertIPTablesRule(uint(id), cfgUpdates); err != nil {
			logger.Error("UpdateRule: upsert iptables config failed", err, "rule_id", id, "request_id", requestID)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新协议配置失败"})
			return
		}
	}

	log.Info("UpdateRule success", "rule_id", id, "rule_name", rule.Name)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "更新成功",
	})
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
