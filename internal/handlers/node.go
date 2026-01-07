package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"bakaray/internal/logger"
	"bakaray/internal/middleware"
	"bakaray/internal/models"
	"bakaray/internal/services"

	"github.com/gin-gonic/gin"
)

// NodeHandler 节点处理器
type NodeHandler struct {
	nodeService  *services.NodeService
	ruleService  *services.RuleService
	userService  *services.UserService
}

// NewNodeHandler 创建节点处理器
func NewNodeHandler(nodeService *services.NodeService, ruleService *services.RuleService, userService *services.UserService) *NodeHandler {
	return &NodeHandler{
		nodeService: nodeService,
		ruleService: ruleService,
		userService: userService,
	}
}

// GetNodes 获取节点列表（所有用户看到相同列表）
func (h *NodeHandler) GetNodes(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.Log.With("request_id", requestID, "component", "node")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	log.Debug("GetNodes request", "page", page, "page_size", pageSize, "status", status, "user_id", userID)

	// 所有用户看到相同的节点列表
	nodes, total := h.nodeService.ListNodes(page, pageSize, status)

	type NodeListItem struct {
		models.Node
		Probe *models.ProbeData `json:"probe,omitempty"`
	}

	items := make([]NodeListItem, 0, len(nodes))
	for _, node := range nodes {
		probe, _ := h.nodeService.GetProbeData(node.ID)
		items = append(items, NodeListItem{
			Node:  node,
			Probe: probe,
		})
	}

	log.Info("GetNodes success", "count", len(items), "total", total)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  items,
			"total": total,
			"page":  page,
		},
	})
}

// GetNode 获取节点详情
func (h *NodeHandler) GetNode(c *gin.Context) {
	requestID := c.GetString("request_id")
	log := logger.Log.With("request_id", requestID, "component", "node")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	log.Debug("GetNode request", "node_id", id)

	node, err := h.nodeService.GetNodeByID(uint(id))
	if err != nil {
		logger.Error("GetNode: node not found", err, "node_id", id, "request_id", requestID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}

	probe, err := h.nodeService.GetProbeData(node.ID)

	log.Info("GetNode success", "node_id", id, "node_name", node.Name)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"node":  node,
			"probe": probe,
		},
	})
}

// NodeHeartbeatRequest 节点心跳请求
type NodeHeartbeatRequest struct {
	NodeID uint             `json:"node_id" binding:"required"`
	Secret string          `json:"secret" binding:"required"`
	Probe  *models.ProbeData `json:"probe"`
	TrafficStats map[string]int64 `json:"traffic_stats"`
}

// NodeHeartbeat 节点心跳
func (h *NodeHandler) NodeHeartbeat(c *gin.Context) {
	requestID := c.GetString("request_id")
	log := logger.Log.With("request_id", requestID, "component", "node")

	var req NodeHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("NodeHeartbeat: invalid request", "error", err, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("NodeHeartbeat request", "node_id", req.NodeID)

	node, err := h.nodeService.GetNodeByID(req.NodeID)
	if err != nil || node.Secret != req.Secret {
		logger.Warn("NodeHeartbeat: invalid node secret", "node_id", req.NodeID, "request_id", requestID)
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的节点密钥"})
		return
	}

	h.nodeService.UpdateNodeStatus(req.NodeID, "online")

	if req.Probe != nil {
		h.nodeService.SaveProbeData(req.NodeID, req.Probe)
	}

	if len(req.TrafficStats) > 0 {
		deltas, err := h.nodeService.ComputeTrafficDeltas(req.NodeID, req.TrafficStats)
		if err == nil {
			now := time.Now()
			disabledCount := 0
			for ruleID, d := range deltas {
				total := d.BytesIn + d.BytesOut
				if total <= 0 {
					continue
				}
				disabled, err := h.ruleService.UpdateTrafficUsedWithDisable(ruleID, total)
				if err != nil {
					logger.Warn("NodeHeartbeat: failed to update traffic", "rule_id", ruleID, "error", err, "request_id", requestID)
					continue
				}
				if disabled {
					disabledCount++
					logger.Warn("NodeHeartbeat: rule disabled due to traffic limit", "rule_id", ruleID, "node_id", req.NodeID, "request_id", requestID)
				}
				_ = h.ruleService.CreateTrafficLog(&models.TrafficLog{
					RuleID:    ruleID,
					NodeID:    req.NodeID,
					BytesIn:   d.BytesIn,
					BytesOut:  d.BytesOut,
					Timestamp: now,
				})
			}
			logger.Debug("NodeHeartbeat: traffic updated", "node_id", req.NodeID, "rules_count", len(deltas), "disabled_count", disabledCount, "request_id", requestID)
		} else {
			logger.Warn("NodeHeartbeat: compute traffic deltas failed", "error", err, "node_id", req.NodeID, "request_id", requestID)
		}
	}

	log.Info("NodeHeartbeat success", "node_id", req.NodeID, "node_name", node.Name)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "心跳成功",
	})
}

// NodeConfigRequest 获取配置请求
type NodeConfigRequest struct {
	NodeID uint   `json:"node_id" binding:"required"`
	Secret string `json:"secret" binding:"required"`
}

// NodeConfig 获取节点配置
func (h *NodeHandler) NodeConfig(c *gin.Context) {
	requestID := c.GetString("request_id")
	log := logger.Log.With("request_id", requestID, "component", "node")

	var req NodeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("NodeConfig: invalid request", "error", err, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("NodeConfig request", "node_id", req.NodeID)

	node, err := h.nodeService.GetNodeByID(req.NodeID)
	if err != nil || node.Secret != req.Secret {
		logger.Warn("NodeConfig: invalid node secret", "node_id", req.NodeID, "request_id", requestID)
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的节点密钥"})
		return
	}

	rules, _ := h.nodeService.ListRulesByNode(req.NodeID, true)

	type NodeTarget struct {
		Host    string `json:"host"`
		Port    int    `json:"port"`
		Weight  int    `json:"weight"`
		Enabled bool   `json:"enabled"`
	}
	type NodeGostConfig struct {
		Transport string `json:"transport"`
		TLS       bool   `json:"tls"`
		Chain     string `json:"chain"`
		Timeout   int    `json:"timeout"`
	}
	type NodeIPTablesConfig struct {
		Proto string `json:"proto"`
		SNAT  bool   `json:"snat"`
		Iface string `json:"iface"`
	}
	type NodeRule struct {
		ID             uint               `json:"id"`
		Name           string             `json:"name"`
		Protocol       string             `json:"protocol"`
		ListenPort     int                `json:"listen_port"`
		Mode           string             `json:"mode"`
		Targets        []NodeTarget       `json:"targets"`
		SpeedLimit     int64              `json:"speed_limit"`
		Enabled        bool               `json:"enabled"`
		GostConfig     *NodeGostConfig     `json:"gost_config,omitempty"`
		IPTablesConfig *NodeIPTablesConfig `json:"iptables_config,omitempty"`
	}

	nodeRules := make([]NodeRule, 0, len(rules))
	for _, r := range rules {
		targets, _ := h.ruleService.GetTargets(r.ID)
		nodeTargets := make([]NodeTarget, 0, len(targets))
		for _, t := range targets {
			nodeTargets = append(nodeTargets, NodeTarget{
				Host:    t.Host,
				Port:    t.Port,
				Weight:  t.Weight,
				Enabled: t.Enabled,
			})
		}

		nr := NodeRule{
			ID:         r.ID,
			Name:       r.Name,
			Protocol:   r.Protocol,
			ListenPort: r.ListenPort,
			Mode:       r.Mode,
			Targets:    nodeTargets,
			SpeedLimit: r.SpeedLimit,
			Enabled:    r.Enabled,
		}

		switch r.Protocol {
		case "gost":
			cfg, _ := h.ruleService.GetGostRule(r.ID)
			if cfg != nil {
				nr.GostConfig = &NodeGostConfig{
					Transport: cfg.Transport,
					TLS:       cfg.TLS,
					Chain:     cfg.Chain,
					Timeout:   cfg.Timeout,
				}
			}
		case "iptables":
			cfg, _ := h.ruleService.GetIPTablesRule(r.ID)
			if cfg != nil {
				nr.IPTablesConfig = &NodeIPTablesConfig{
					Proto: cfg.Proto,
					SNAT:  cfg.SNAT,
					Iface: cfg.Iface,
				}
			}
		}

		nodeRules = append(nodeRules, nr)
	}

	rulesJSON, err := json.Marshal(nodeRules)
	if err != nil {
		logger.Error("NodeConfig: marshal rules failed", err, "node_id", req.NodeID, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成配置失败"})
		return
	}

	log.Info("NodeConfig success", "node_id", req.NodeID, "rules_count", len(nodeRules))

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"rules":   string(rulesJSON),
			"version": 1,
		},
	})
}

// NodeReportRequest 节点上报请求
type NodeReportRequest struct {
	NodeID uint                `json:"node_id" binding:"required"`
	Secret string             `json:"secret" binding:"required"`
	Report *models.ProbeData  `json:"report"`
}

// NodeReport 节点上报数据
func (h *NodeHandler) NodeReport(c *gin.Context) {
	requestID := c.GetString("request_id")
	log := logger.Log.With("request_id", requestID, "component", "node")

	var req NodeReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("NodeReport: invalid request", "error", err, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	log.Debug("NodeReport request", "node_id", req.NodeID)

	node, err := h.nodeService.GetNodeByID(req.NodeID)
	if err != nil || node.Secret != req.Secret {
		logger.Warn("NodeReport: invalid node secret", "node_id", req.NodeID, "request_id", requestID)
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的节点密钥"})
		return
	}

	if req.Report != nil {
		h.nodeService.SaveProbeData(req.NodeID, req.Report)
	}

	log.Info("NodeReport success", "node_id", req.NodeID, "node_name", node.Name)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "上报成功",
	})
}
