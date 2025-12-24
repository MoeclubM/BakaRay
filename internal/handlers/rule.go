package handlers

import (
	"net/http"
	"strconv"

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
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	rules, total := h.ruleService.ListRulesByUser(userID, page, pageSize)

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
	userID := middleware.GetUserID(c)
	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

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
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建规则失败"})
		return
	}

	// 创建目标
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

	// 创建协议专用配置
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
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	rule, err := h.ruleService.GetRuleByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}

	// 获取目标
	targets, _ := h.ruleService.GetTargets(rule.ID)

	// 获取协议配置
	var gostRule *models.GostRule
	var iptRule *models.IPTablesRule

	switch rule.Protocol {
	case "gost":
		gostRule, _ = h.ruleService.GetGostRule(rule.ID)
	case "iptables":
		iptRule, _ = h.ruleService.GetIPTablesRule(rule.ID)
	}

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
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.ruleService.DeleteRule(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// UpdateRuleRequest 更新规则请求
type UpdateRuleRequest struct {
	Name         string          `json:"name"`
	Enabled      *bool           `json:"enabled"`
	NodeID       uint            `json:"node_id"`
	TrafficLimit int64           `json:"traffic_limit"`
	SpeedLimit   int64           `json:"speed_limit"`
	Mode         string          `json:"mode"`
	Targets      []TargetRequest `json:"targets"`
}

// UpdateRule 更新规则
func (h *RuleHandler) UpdateRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	userID := middleware.GetUserID(c)

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	// 检查规则是否存在且属于该用户
	rule, err := h.ruleService.GetRuleByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "规则不存在"})
		return
	}
	if rule.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "无权操作此规则"})
		return
	}

	// 更新规则
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	if req.TrafficLimit > 0 {
		updates["traffic_limit"] = req.TrafficLimit
	}
	if req.SpeedLimit > 0 {
		updates["speed_limit"] = req.SpeedLimit
	}
	if req.Mode != "" {
		updates["mode"] = req.Mode
	}
	if req.NodeID > 0 {
		updates["node_id"] = req.NodeID
	}

	if err := h.ruleService.UpdateRule(uint(id), updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	// 更新目标
	if req.Targets != nil && len(req.Targets) > 0 {
		h.ruleService.DeleteTargetsByRuleID(uint(id))
		for _, t := range req.Targets {
			target := &models.Target{
				RuleID:  uint(id),
				Host:    t.Host,
				Port:    t.Port,
				Weight:  t.Weight,
				Enabled: t.Enabled,
			}
			h.ruleService.AddTarget(target)
		}
	}

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
