package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"bakaray/internal/models"
	"bakaray/internal/services"

	"github.com/gin-gonic/gin"
)

// NodeHandler 节点处理器
type NodeHandler struct {
	nodeService *services.NodeService
	ruleService *services.RuleService
}

// NewNodeHandler 创建节点处理器
func NewNodeHandler(nodeService *services.NodeService, ruleService *services.RuleService) *NodeHandler {
	return &NodeHandler{
		nodeService: nodeService,
		ruleService: ruleService,
	}
}

// GetNodes 获取节点列表
func (h *NodeHandler) GetNodes(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

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
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	node, err := h.nodeService.GetNodeByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}

	// 获取探针数据
	probe, err := h.nodeService.GetProbeData(node.ID)

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
	var req NodeHeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 验证节点密钥
	node, err := h.nodeService.GetNodeByID(req.NodeID)
	if err != nil || node.Secret != req.Secret {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的节点密钥"})
		return
	}

	// 更新节点状态
	h.nodeService.UpdateNodeStatus(req.NodeID, "online")

	// 保存探针数据
	if req.Probe != nil {
		h.nodeService.SaveProbeData(req.NodeID, req.Probe)
	}

	// 处理流量统计（可选）
	if len(req.TrafficStats) > 0 {
		deltas, err := h.nodeService.ComputeTrafficDeltas(req.NodeID, req.TrafficStats)
		if err == nil {
			now := time.Now()
			for ruleID, d := range deltas {
				total := d.BytesIn + d.BytesOut
				if total <= 0 {
					continue
				}
				_ = h.ruleService.UpdateTrafficUsed(ruleID, total)
				_ = h.ruleService.CreateTrafficLog(&models.TrafficLog{
					RuleID:    ruleID,
					NodeID:    req.NodeID,
					BytesIn:   d.BytesIn,
					BytesOut:  d.BytesOut,
					Timestamp: now,
				})
			}
		}
	}

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
	var req NodeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 验证节点
	node, err := h.nodeService.GetNodeByID(req.NodeID)
	if err != nil || node.Secret != req.Secret {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的节点密钥"})
		return
	}

	// 获取规则列表
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
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "生成配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"rules":   string(rulesJSON),
			"version": 1, // 配置版本号
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
	var req NodeReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	// 验证节点
	node, err := h.nodeService.GetNodeByID(req.NodeID)
	if err != nil || node.Secret != req.Secret {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的节点密钥"})
		return
	}

	// 保存探针数据
	if req.Report != nil {
		h.nodeService.SaveProbeData(req.NodeID, req.Report)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "上报成功",
	})
}
