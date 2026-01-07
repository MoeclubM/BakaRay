package services

import (
	"errors"
	"time"

	"bakaray/internal/models"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var ErrRuleNotFound = errors.New("规则不存在")

// RuleService 转发规则服务
type RuleService struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewRuleService 创建规则服务
func NewRuleService(db *gorm.DB, redis *redis.Client) *RuleService {
	return &RuleService{db: db, redis: redis}
}

// CreateRule 创建转发规则
func (s *RuleService) CreateRule(rule *models.ForwardingRule) error {
	return s.db.Create(rule).Error
}

// GetRuleByID 根据ID获取规则
func (s *RuleService) GetRuleByID(id uint) (*models.ForwardingRule, error) {
	var rule models.ForwardingRule
	if err := s.db.First(&rule, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRuleNotFound
		}
		return nil, err
	}
	return &rule, nil
}

// UpdateRule 更新规则
func (s *RuleService) UpdateRule(id uint, updates map[string]interface{}) error {
	return s.db.Model(&models.ForwardingRule{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteRule 删除规则
func (s *RuleService) DeleteRule(id uint) error {
	return s.db.Delete(&models.ForwardingRule{}, id).Error
}

// ListRulesByUser 获取用户的规则列表
func (s *RuleService) ListRulesByUser(userID uint, page, pageSize int) ([]models.ForwardingRule, int64) {
	var rules []models.ForwardingRule
	var total int64

	s.db.Model(&models.ForwardingRule{}).Where("user_id = ?", userID).Count(&total)
	offset := (page - 1) * pageSize
	s.db.Where("user_id = ?", userID).Offset(offset).Limit(pageSize).Find(&rules)

	return rules, total
}

// ListRulesByNode 获取节点的规则列表
func (s *RuleService) ListRulesByNode(nodeID uint, enabledOnly bool) ([]models.ForwardingRule, error) {
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

// GetTargets 获取规则的目标列表
func (s *RuleService) GetTargets(ruleID uint) ([]models.Target, error) {
	var targets []models.Target
	if err := s.db.Where("rule_id = ? AND enabled = ?", ruleID, true).Find(&targets).Error; err != nil {
		return nil, err
	}
	return targets, nil
}

// AddTarget 添加转发目标
func (s *RuleService) AddTarget(target *models.Target) error {
	return s.db.Create(target).Error
}

// DeleteTarget 删除转发目标
func (s *RuleService) DeleteTarget(id uint) error {
	return s.db.Delete(&models.Target{}, id).Error
}

// DeleteTargetsByRuleID 删除规则的所有目标
func (s *RuleService) DeleteTargetsByRuleID(ruleID uint) error {
	return s.db.Where("rule_id = ?", ruleID).Delete(&models.Target{}).Error
}

// GetGostRule 获取 gost 协议配置
func (s *RuleService) GetGostRule(ruleID uint) (*models.GostRule, error) {
	var rule models.GostRule
	if err := s.db.Where("rule_id = ?", ruleID).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// CreateGostRule 创建 gost 协议配置
func (s *RuleService) CreateGostRule(rule *models.GostRule) error {
	return s.db.Create(rule).Error
}

// UpsertGostRule creates or updates gost config for a rule.
func (s *RuleService) UpsertGostRule(ruleID uint, updates map[string]interface{}) error {
	var existing models.GostRule
	err := s.db.Where("rule_id = ?", ruleID).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cfg := models.GostRule{RuleID: ruleID}
			if v, ok := updates["transport"]; ok {
				cfg.Transport, _ = v.(string)
			}
			if v, ok := updates["tls"]; ok {
				cfg.TLS, _ = v.(bool)
			}
			if v, ok := updates["chain"]; ok {
				cfg.Chain, _ = v.(string)
			}
			if v, ok := updates["timeout"]; ok {
				cfg.Timeout, _ = v.(int)
			}
			return s.db.Create(&cfg).Error
		}
		return err
	}

	return s.db.Model(&models.GostRule{}).Where("rule_id = ?", ruleID).Updates(updates).Error
}

// GetIPTablesRule 获取 iptables 协议配置
func (s *RuleService) GetIPTablesRule(ruleID uint) (*models.IPTablesRule, error) {
	var rule models.IPTablesRule
	if err := s.db.Where("rule_id = ?", ruleID).First(&rule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// CreateIPTablesRule 创建 iptables 协议配置
func (s *RuleService) CreateIPTablesRule(rule *models.IPTablesRule) error {
	return s.db.Create(rule).Error
}

// UpsertIPTablesRule creates or updates iptables config for a rule.
func (s *RuleService) UpsertIPTablesRule(ruleID uint, updates map[string]interface{}) error {
	var existing models.IPTablesRule
	err := s.db.Where("rule_id = ?", ruleID).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cfg := models.IPTablesRule{RuleID: ruleID}
			if v, ok := updates["proto"]; ok {
				cfg.Proto, _ = v.(string)
			}
			if v, ok := updates["snat"]; ok {
				cfg.SNAT, _ = v.(bool)
			}
			if v, ok := updates["iface"]; ok {
				cfg.Iface, _ = v.(string)
			}
			return s.db.Create(&cfg).Error
		}
		return err
	}

	return s.db.Model(&models.IPTablesRule{}).Where("rule_id = ?", ruleID).Updates(updates).Error
}

// MaxTrafficLimit 单次更新流量上限（防止异常大流量），单位：字节
const MaxTrafficLimit int64 = 1024 * 1024 * 1024 * 10 // 10GB

// UpdateTrafficUsed 更新已用流量（带上限检查）
func (s *RuleService) UpdateTrafficUsed(ruleID uint, bytes int64) error {
	// 检查并限制单次更新量
	if bytes > MaxTrafficLimit {
		bytes = MaxTrafficLimit
	}
	if bytes < 0 {
		bytes = 0
	}

	// 更新流量，限制不超过 traffic_limit（如果有设置）
	return s.db.Model(&models.ForwardingRule{}).Where("id = ? AND (traffic_limit = 0 OR traffic_used < traffic_limit)", ruleID).
		Update("traffic_used", gorm.Expr("LEAST(traffic_used + ?, COALESCE(traffic_limit, 0))", bytes)).Error
}

// UpdateTrafficUsedWithDisable 更新流量并自动禁用超限规则
func (s *RuleService) UpdateTrafficUsedWithDisable(ruleID uint, bytes int64) (disabled bool, err error) {
	// 检查并限制单次更新量
	if bytes > MaxTrafficLimit {
		bytes = MaxTrafficLimit
	}
	if bytes < 0 {
		bytes = 0
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		var rule models.ForwardingRule
		if err := tx.First(&rule, ruleID).Error; err != nil {
			return err
		}

		newUsed := rule.TrafficUsed + bytes
		// 检查是否超过限制
		if rule.TrafficLimit > 0 && newUsed >= rule.TrafficLimit {
			// 禁用规则
			if err := tx.Model(&rule).Where("id = ?", ruleID).Updates(map[string]interface{}{
				"traffic_used": rule.TrafficLimit,
				"enabled":      false,
			}).Error; err != nil {
				return err
			}
			disabled = true
			return nil
		}

		// 正常更新流量
		return tx.Model(&rule).Where("id = ?", ruleID).
			Update("traffic_used", gorm.Expr("traffic_used + ?", bytes)).Error
	})
	return
}

// CreateTrafficLog records a traffic delta entry.
func (s *RuleService) CreateTrafficLog(logEntry *models.TrafficLog) error {
	return s.db.Create(logEntry).Error
}

// GetUserTrafficUsed returns the sum of traffic_used for all rules belonging to the user.
func (s *RuleService) GetUserTrafficUsed(userID uint) (int64, error) {
	var total int64
	if err := s.db.Model(&models.ForwardingRule{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(traffic_used),0)").
		Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// GetUserTrafficStats sums traffic logs for the user since the given time.
func (s *RuleService) GetUserTrafficStats(userID uint, since time.Time) (int64, int64, error) {
	type agg struct {
		In  int64 `gorm:"column:bytes_in"`
		Out int64 `gorm:"column:bytes_out"`
	}
	var res agg

	err := s.db.Table("traffic_logs").
		Select("COALESCE(SUM(traffic_logs.bytes_in),0) AS bytes_in, COALESCE(SUM(traffic_logs.bytes_out),0) AS bytes_out").
		Joins("JOIN forwarding_rules ON forwarding_rules.id = traffic_logs.rule_id").
		Where("forwarding_rules.user_id = ? AND traffic_logs.timestamp >= ?", userID, since).
		Scan(&res).Error
	if err != nil {
		return 0, 0, err
	}
	return res.In, res.Out, nil
}

// ListAllRules 获取所有规则（管理员用）
func (s *RuleService) ListAllRules(page, pageSize int, nodeID, userID uint) ([]models.ForwardingRule, int64) {
	var rules []models.ForwardingRule
	var total int64

	query := s.db.Model(&models.ForwardingRule{})
	if nodeID > 0 {
		query = query.Where("node_id = ?", nodeID)
	}
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	query.Offset(offset).Limit(pageSize).Find(&rules)

	return rules, total
}

// CountAllRules 统计所有规则数量
func (s *RuleService) CountAllRules() int64 {
	var total int64
	s.db.Model(&models.ForwardingRule{}).Count(&total)
	return total
}
