package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bakaray/internal/models"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var ErrNodeNotFound = errors.New("节点不存在")

// NodeService 节点服务
type NodeService struct {
	db    *gorm.DB
	redis *redis.Client
}

type TrafficDelta struct {
	BytesIn  int64
	BytesOut int64
}

// NewNodeService 创建节点服务
func NewNodeService(db *gorm.DB, redis *redis.Client) *NodeService {
	return &NodeService{
		db:    db,
		redis: redis,
	}
}

// CreateNode 创建节点
func (s *NodeService) CreateNode(name, host string, port int, secret string, groupID uint, protocols []string, multiplier float64, region string) (*models.Node, error) {
	node := &models.Node{
		Name:        name,
		Host:        host,
		Port:        port,
		Secret:      secret,
		Status:      "offline",
		NodeGroupID: groupID,
		Protocols:   models.StringSlice(protocols),
		Multiplier:  multiplier,
		Region:      region,
	}

	if err := s.db.Create(node).Error; err != nil {
		return nil, err
	}
	return node, nil
}

// GetNodeByID 根据ID获取节点
func (s *NodeService) GetNodeByID(id uint) (*models.Node, error) {
	var node models.Node
	if err := s.db.First(&node, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNodeNotFound
		}
		return nil, err
	}
	return &node, nil
}

// UpdateNodeStatus 更新节点状态
func (s *NodeService) UpdateNodeStatus(id uint, status string) error {
	now := time.Now()
	return s.db.Model(&models.Node{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":    status,
		"last_seen": &now,
	}).Error
}

// ListNodes 获取节点列表
func (s *NodeService) ListNodes(page, pageSize int, status string) ([]models.Node, int64) {
	var nodes []models.Node
	var total int64

	query := s.db.Model(&models.Node{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Find(&nodes)

	return nodes, total
}

// SaveProbeData 保存探针数据到 Redis
func (s *NodeService) SaveProbeData(nodeID uint, probe *models.ProbeData) error {
	data, err := json.Marshal(probe)
	if err != nil {
		return err
	}

	ctx := context.Background()
	key := fmt.Sprintf("node_probe:%d", nodeID)
	return s.redis.Set(ctx, key, data, 5*time.Minute).Err()
}

// GetProbeData 获取探针数据从 Redis
func (s *NodeService) GetProbeData(nodeID uint) (*models.ProbeData, error) {
	ctx := context.Background()
	key := fmt.Sprintf("node_probe:%d", nodeID)
	data, err := s.redis.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var probe models.ProbeData
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, err
	}
	return &probe, nil
}

// ComputeTrafficDeltas computes per-rule traffic deltas based on cumulative counters reported by the node.
// Expected keys: rule_{id}_in / rule_{id}_out.
func (s *NodeService) ComputeTrafficDeltas(nodeID uint, stats map[string]int64) (map[uint]TrafficDelta, error) {
	type counts struct {
		in  int64
		out int64
	}

	current := make(map[uint]*counts)
	for k, v := range stats {
		parts := strings.Split(k, "_")
		if len(parts) != 3 || parts[0] != "rule" {
			continue
		}

		ruleID64, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			continue
		}

		ruleID := uint(ruleID64)
		c := current[ruleID]
		if c == nil {
			c = &counts{}
			current[ruleID] = c
		}

		switch parts[2] {
		case "in":
			c.in = v
		case "out":
			c.out = v
		}
	}

	if len(current) == 0 {
		return map[uint]TrafficDelta{}, nil
	}

	ctx := context.Background()
	hashKey := fmt.Sprintf("node_traffic_last:%d", nodeID)
	pipe := s.redis.Pipeline()

	type lastCmds struct {
		in  *redis.StringCmd
		out *redis.StringCmd
	}
	last := make(map[uint]lastCmds, len(current))
	for ruleID := range current {
		last[ruleID] = lastCmds{
			in:  pipe.HGet(ctx, hashKey, fmt.Sprintf("%d_in", ruleID)),
			out: pipe.HGet(ctx, hashKey, fmt.Sprintf("%d_out", ruleID)),
		}
	}

	pipe.Expire(ctx, hashKey, 7*24*time.Hour)

	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	deltas := make(map[uint]TrafficDelta, len(current))
	updatePipe := s.redis.Pipeline()

	for ruleID, c := range current {
		lastIn, _ := strconv.ParseInt(last[ruleID].in.Val(), 10, 64)
		lastOut, _ := strconv.ParseInt(last[ruleID].out.Val(), 10, 64)

		dIn := c.in - lastIn
		dOut := c.out - lastOut
		if dIn < 0 {
			dIn = c.in
		}
		if dOut < 0 {
			dOut = c.out
		}

		if dIn != 0 || dOut != 0 {
			deltas[ruleID] = TrafficDelta{BytesIn: dIn, BytesOut: dOut}
		}

		updatePipe.HSet(ctx, hashKey,
			fmt.Sprintf("%d_in", ruleID), c.in,
			fmt.Sprintf("%d_out", ruleID), c.out,
		)
	}
	updatePipe.Expire(ctx, hashKey, 7*24*time.Hour)
	_, _ = updatePipe.Exec(ctx)

	return deltas, nil
}

// GetAllowedGroups 获取节点允许的用户组
func (s *NodeService) GetAllowedGroups(nodeID uint) ([]uint, error) {
	var relations []models.NodeAllowedGroup
	if err := s.db.Where("node_id = ?", nodeID).Find(&relations).Error; err != nil {
		return nil, err
	}

	groups := make([]uint, len(relations))
	for i, rel := range relations {
		groups[i] = rel.UserGroupID
	}
	return groups, nil
}

// SetAllowedGroups 设置节点允许的用户组
func (s *NodeService) SetAllowedGroups(nodeID uint, groupIDs []uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 删除旧的关联
		if err := tx.Where("node_id = ?", nodeID).Delete(&models.NodeAllowedGroup{}).Error; err != nil {
			return err
		}
		// 创建新的关联
		for _, groupID := range groupIDs {
			if err := tx.Create(&models.NodeAllowedGroup{
				NodeID:      nodeID,
				UserGroupID: groupID,
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ListRulesByNode 获取节点的规则列表
func (s *NodeService) ListRulesByNode(nodeID uint, enabledOnly bool) ([]models.ForwardingRule, error) {
	var rules []models.ForwardingRule
	query := s.db.Where("node_id = ?", nodeID)
	if enabledOnly {
		query = query.Where("enabled = ?", true)
	}
	if err := query.Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// DeleteNode 删除节点
func (s *NodeService) DeleteNode(id uint) error {
	return s.db.Delete(&models.Node{}, id).Error
}

// UpdateNode 更新节点
func (s *NodeService) UpdateNode(id uint, updates map[string]interface{}) error {
	if raw, ok := updates["protocols"]; ok {
		switch v := raw.(type) {
		case []string:
			updates["protocols"] = models.StringSlice(v)
		case []interface{}:
			protocols := make([]string, 0, len(v))
			for _, item := range v {
				if s, ok := item.(string); ok && s != "" {
					protocols = append(protocols, s)
				}
			}
			updates["protocols"] = models.StringSlice(protocols)
		case string:
			// Accept JSON string or single protocol.
			var protocols []string
			if err := json.Unmarshal([]byte(v), &protocols); err == nil {
				updates["protocols"] = models.StringSlice(protocols)
			} else if v != "" {
				updates["protocols"] = models.StringSlice([]string{v})
			} else {
				updates["protocols"] = models.StringSlice(nil)
			}
		default:
			updates["protocols"] = models.StringSlice(nil)
		}
	}

	// Normalize numeric fields that may arrive as strings.
	for _, key := range []string{"port", "node_group_id"} {
		if raw, ok := updates[key]; ok {
			if sVal, ok := raw.(string); ok {
				if num, err := strconv.ParseUint(sVal, 10, 64); err == nil {
					updates[key] = num
				}
			}
		}
	}

	return s.db.Model(&models.Node{}).Where("id = ?", id).Updates(updates).Error
}
