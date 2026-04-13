package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bakaray/internal/logger"
	"bakaray/internal/middleware"
	"bakaray/internal/models"
	"bakaray/internal/services"

	"github.com/gin-gonic/gin"
)

// NodeHandler 节点处理器
type NodeHandler struct {
	userService       *services.UserService
	nodeService       *services.NodeService
	ruleService       *services.RuleService
	siteConfigService *services.SiteConfigService
}

// NewNodeHandler 创建节点处理器
func NewNodeHandler(userService *services.UserService, nodeService *services.NodeService, ruleService *services.RuleService, siteConfigService *services.SiteConfigService) *NodeHandler {
	return &NodeHandler{
		userService:       userService,
		nodeService:       nodeService,
		ruleService:       ruleService,
		siteConfigService: siteConfigService,
	}
}

// GetNodes 获取当前用户可见的节点列表
func (h *NodeHandler) GetNodes(c *gin.Context) {
	requestID := c.GetString("request_id")
	userID := middleware.GetUserID(c)
	log := logger.Log.With("request_id", requestID, "component", "node")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")

	log.Debug("GetNodes request", "page", page, "page_size", pageSize, "status", status, "user_id", userID)

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户不存在"})
		return
	}

	nodes, total := h.nodeService.ListNodesForUser(user.UserGroupID, page, pageSize, status)

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
	userID := middleware.GetUserID(c)
	log := logger.Log.With("request_id", requestID, "component", "node")

	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	log.Debug("GetNode request", "node_id", id)

	node, err := h.nodeService.GetNodeByID(uint(id))
	if err != nil {
		logger.Error("GetNode: node not found", err, "node_id", id, "request_id", requestID)
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "节点不存在"})
		return
	}

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户不存在"})
		return
	}

	allowedGroups, err := h.nodeService.GetAllowedGroups(node.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取节点授权失败"})
		return
	}

	allowed := false
	for _, groupID := range allowedGroups {
		if groupID == user.UserGroupID {
			allowed = true
			break
		}
	}
	if !allowed {
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
	NodeID       uint                    `json:"node_id" binding:"required"`
	Secret       string                  `json:"secret" binding:"required"`
	Probe        *models.ProbeData       `json:"probe"`
	TrafficStats map[string]int64        `json:"traffic_stats"`
	Diagnostics  []models.NodeDiagnostic `json:"diagnostics"`
}

type NodeRegisterRequest struct {
	Name   string `json:"name"`
	Secret string `json:"secret" binding:"required"`
}

// NodeRegister 节点自动注册
func (h *NodeHandler) NodeRegister(c *gin.Context) {
	requestID := c.GetString("request_id")
	log := logger.Log.With("request_id", requestID, "component", "node")

	if h.siteConfigService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "站点配置服务未初始化"})
		return
	}

	var req NodeRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("NodeRegister: invalid request", "error", err, "request_id", requestID)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	site, err := h.siteConfigService.GetOrCreate()
	if err != nil {
		logger.Error("NodeRegister: failed to load site config", err, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取站点配置失败"})
		return
	}
	if req.Secret != site.NodeSecret {
		logger.Warn("NodeRegister: invalid node secret", "request_id", requestID)
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的节点密钥"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		replacer := strings.NewReplacer(".", "-", ":", "-")
		name = "node-" + replacer.Replace(c.ClientIP())
	}

	host := strings.TrimSpace(c.ClientIP())
	if host == "" {
		host = "127.0.0.1"
	}

	node, err := h.nodeService.RegisterNode(name, host, 0, req.Secret)
	if err != nil {
		logger.Error("NodeRegister: failed to register node", err, "name", name, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "注册节点失败"})
		return
	}

	log.Info("NodeRegister success", "node_id", node.ID, "name", node.Name, "host", node.Host)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"id":      node.ID,
			"node_id": node.ID,
			"name":    node.Name,
			"host":    node.Host,
		},
	})
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
	if req.Diagnostics != nil {
		h.nodeService.SaveDiagnostics(req.NodeID, req.Diagnostics)
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
		"code":    0,
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
	site, err := h.siteConfigService.GetOrCreate()
	if err != nil {
		logger.Error("NodeConfig: load site config failed", err, "node_id", req.NodeID, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取站点配置失败"})
		return
	}

	type NodeTarget struct {
		Host    string `json:"host"`
		Port    int    `json:"port"`
		Weight  int    `json:"weight"`
		Enabled bool   `json:"enabled"`
	}
	type NodeRule struct {
		ID             uint         `json:"id"`
		Name           string       `json:"name"`
		Protocol       string       `json:"protocol"`
		ListenPort     int          `json:"listen_port"`
		Mode           string       `json:"mode"`
		Targets        []NodeTarget `json:"targets"`
		SpeedLimit     int64        `json:"speed_limit"`
		Enabled        bool         `json:"enabled"`
		TunnelRole     string       `json:"tunnel_role,omitempty"`
		TunnelProtocol string       `json:"tunnel_protocol,omitempty"`
		TunnelRemote   string       `json:"tunnel_remote,omitempty"`
		ReportTraffic  bool         `json:"report_traffic"`
	}

	nodeRules := make([]NodeRule, 0, len(rules))
	for _, r := range rules {
		ruleProtocol := services.NormalizeDirectProtocol(r.Protocol)
		if !services.NodeSupportsDirectProtocol([]string(node.Protocols), ruleProtocol) {
			continue
		}
		targets, _ := h.ruleService.ListTargets(r.ID, true)
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
			ID:            r.ID,
			Name:          r.Name,
			Protocol:      ruleProtocol,
			ListenPort:    r.ListenPort,
			Mode:          r.Mode,
			Targets:       nodeTargets,
			SpeedLimit:    r.SpeedLimit,
			Enabled:       r.Enabled,
			ReportTraffic: true,
		}
		if r.TunnelEnabled {
			tunnelProtocol := services.NormalizeTunnelProtocol(r.TunnelProtocol)
			if !services.NodeSupportsTunnelProtocol([]string(node.Protocols), tunnelProtocol) {
				continue
			}
			exitNode, err := h.nodeService.GetNodeByID(r.ExitNodeID)
			if err != nil {
				continue
			}
			if !services.NodeSupportsTunnelProtocol([]string(exitNode.Protocols), tunnelProtocol) {
				continue
			}
			nr.TunnelRole = "entry"
			nr.TunnelProtocol = tunnelProtocol
			nr.TunnelRemote = fmt.Sprintf("%s:%d", exitNode.Host, r.TunnelPort)
		}
		nodeRules = append(nodeRules, nr)
	}

	exitRules, err := h.ruleService.ListRulesByExitNode(req.NodeID, true)
	if err != nil {
		logger.Error("NodeConfig: load exit rules failed", err, "node_id", req.NodeID, "request_id", requestID)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "读取隧道规则失败"})
		return
	}
	for _, r := range exitRules {
		tunnelProtocol := services.NormalizeTunnelProtocol(r.TunnelProtocol)
		if !r.TunnelEnabled {
			continue
		}
		if !services.NodeSupportsTunnelProtocol([]string(node.Protocols), tunnelProtocol) {
			continue
		}
		nodeRules = append(nodeRules, NodeRule{
			ID:             r.ID,
			Name:           r.Name + " (隧道出口)",
			Protocol:       services.NormalizeDirectProtocol(r.Protocol),
			ListenPort:     r.TunnelPort,
			Mode:           "direct",
			Enabled:        r.Enabled,
			TunnelRole:     "exit",
			TunnelProtocol: tunnelProtocol,
			ReportTraffic:  false,
		})
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
			"rules":           string(rulesJSON),
			"report_interval": site.NodeReportInterval,
			"version":         1,
		},
	})
}

// NodeReportRequest 节点上报请求
type NodeReportRequest struct {
	NodeID uint              `json:"node_id" binding:"required"`
	Secret string            `json:"secret" binding:"required"`
	Report *models.ProbeData `json:"report"`
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
		"code":    0,
		"message": "上报成功",
	})
}
