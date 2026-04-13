package services

import (
	"testing"
	"time"

	"bakaray/internal/models"

	"github.com/stretchr/testify/require"
)

func TestRuleCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewRuleService(db, nil)
	user := createTestUser(t, db, "rule-crud-user")
	node := createTestNode(t, db, "rule-crud-node")

	rule := &models.ForwardingRule{
		NodeID:       node.ID,
		UserID:       user.ID,
		Name:         "Test Rule",
		Protocol:     "tcp",
		Enabled:      true,
		TrafficUsed:  0,
		TrafficLimit: 2048,
		ListenPort:   8080,
	}

	require.NoError(t, service.CreateRule(rule))
	require.NotZero(t, rule.ID)

	stored, err := service.GetRuleByID(rule.ID)
	require.NoError(t, err)
	require.Equal(t, "Test Rule", stored.Name)
	require.Equal(t, node.ID, stored.NodeID)
	require.Equal(t, user.ID, stored.UserID)

	require.NoError(t, service.UpdateRule(rule.ID, map[string]interface{}{
		"name":    "Updated Rule",
		"enabled": false,
	}))

	stored, err = service.GetRuleByID(rule.ID)
	require.NoError(t, err)
	require.Equal(t, "Updated Rule", stored.Name)
	require.False(t, stored.Enabled)

	require.NoError(t, service.DeleteRule(rule.ID))

	_, err = service.GetRuleByID(rule.ID)
	require.Error(t, err)
	require.Equal(t, ErrRuleNotFound, err)

	_, err = service.GetRuleByID(99999)
	require.Error(t, err)
	require.Equal(t, ErrRuleNotFound, err)
}

func TestListRules(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewRuleService(db, nil)
	user1 := createTestUser(t, db, "rule-list-user-1")
	user2 := createTestUser(t, db, "rule-list-user-2")
	node1 := createTestNode(t, db, "rule-list-node-1")
	node2 := createTestNode(t, db, "rule-list-node-2")

	rule1 := &models.ForwardingRule{NodeID: node1.ID, UserID: user1.ID, Name: "Rule 1", Protocol: "tcp", Enabled: true, ListenPort: 8101}
	rule2 := &models.ForwardingRule{NodeID: node1.ID, UserID: user1.ID, Name: "Rule 2", Protocol: "tcp", Enabled: true, ListenPort: 8102}
	rule3 := &models.ForwardingRule{NodeID: node2.ID, UserID: user2.ID, Name: "Rule 3", Protocol: "tcp", Enabled: true, ListenPort: 8103}
	require.NoError(t, db.Create(rule1).Error)
	require.NoError(t, db.Create(rule2).Error)
	require.NoError(t, db.Create(rule3).Error)
	require.NoError(t, db.Model(&models.ForwardingRule{}).Where("id = ?", rule2.ID).Update("enabled", false).Error)

	rulesByUser, totalByUser := service.ListRulesByUser(user1.ID, 1, 10)
	require.Len(t, rulesByUser, 2)
	require.Equal(t, int64(2), totalByUser)

	secondPage, secondTotal := service.ListRulesByUser(user1.ID, 2, 1)
	require.Len(t, secondPage, 1)
	require.Equal(t, int64(2), secondTotal)

	rulesByNode, err := service.ListRulesByNode(node1.ID, false)
	require.NoError(t, err)
	require.Len(t, rulesByNode, 2)

	enabledRules, err := service.ListRulesByNode(node1.ID, true)
	require.NoError(t, err)
	require.Len(t, enabledRules, 1)
	require.Equal(t, rule1.ID, enabledRules[0].ID)

	allRules, allTotal := service.ListAllRules(1, 10, 0, 0)
	require.Len(t, allRules, 3)
	require.Equal(t, int64(3), allTotal)

	nodeRules, nodeTotal := service.ListAllRules(1, 10, node1.ID, 0)
	require.Len(t, nodeRules, 2)
	require.Equal(t, int64(2), nodeTotal)

	userRules, userTotal := service.ListAllRules(1, 10, 0, user2.ID)
	require.Len(t, userRules, 1)
	require.Equal(t, int64(1), userTotal)

	require.Equal(t, int64(3), service.CountAllRules())
}

func TestTrafficAccounting(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewRuleService(db, nil)
	user := createTestUser(t, db, "traffic-user")
	node := createTestNode(t, db, "traffic-node")

	rule := &models.ForwardingRule{
		NodeID:       node.ID,
		UserID:       user.ID,
		Name:         "Traffic Rule",
		Protocol:     "udp",
		Enabled:      true,
		TrafficLimit: 2048,
		ListenPort:   8201,
	}
	require.NoError(t, db.Create(rule).Error)

	require.NoError(t, service.UpdateTrafficUsed(rule.ID, 1024))

	stored, err := service.GetRuleByID(rule.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1024), stored.TrafficUsed)

	require.NoError(t, service.UpdateTrafficUsed(rule.ID, 4096))

	stored, err = service.GetRuleByID(rule.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2048), stored.TrafficUsed)

	disabled, err := service.UpdateTrafficUsedWithDisable(rule.ID, 1)
	require.NoError(t, err)
	require.True(t, disabled)

	stored, err = service.GetRuleByID(rule.ID)
	require.NoError(t, err)
	require.False(t, stored.Enabled)
	require.Equal(t, int64(2048), stored.TrafficUsed)

	require.NoError(t, service.CreateTrafficLog(&models.TrafficLog{
		RuleID:    rule.ID,
		NodeID:    node.ID,
		BytesIn:   300,
		BytesOut:  700,
		Timestamp: time.Now(),
	}))

	totalUsed, err := service.GetUserTrafficUsed(user.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2048), totalUsed)

	bytesIn, bytesOut, err := service.GetUserTrafficStats(user.ID, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(300), bytesIn)
	require.Equal(t, int64(700), bytesOut)
}

func TestTargets(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewRuleService(db, nil)
	user := createTestUser(t, db, "target-user")
	node := createTestNode(t, db, "target-node")
	rule := &models.ForwardingRule{
		NodeID:     node.ID,
		UserID:     user.ID,
		Name:       "Target Rule",
		Protocol:   "tcp",
		Enabled:    true,
		ListenPort: 8301,
	}
	require.NoError(t, db.Create(rule).Error)

	target1 := &models.Target{RuleID: rule.ID, Host: "target1.example.com", Port: 80, Weight: 1, Enabled: true}
	target2 := &models.Target{RuleID: rule.ID, Host: "target2.example.com", Port: 443, Weight: 2, Enabled: true}
	require.NoError(t, service.AddTarget(target1))
	require.NoError(t, service.AddTarget(target2))
	require.NoError(t, db.Model(&models.Target{}).Where("id = ?", target2.ID).Update("enabled", false).Error)

	targets, err := service.GetTargets(rule.ID)
	require.NoError(t, err)
	require.Len(t, targets, 1)
	require.Equal(t, "target1.example.com", targets[0].Host)

	allTargets, err := service.ListTargets(rule.ID, false)
	require.NoError(t, err)
	require.Len(t, allTargets, 2)

	require.NoError(t, service.DeleteTarget(target1.ID))
	allTargets, err = service.ListTargets(rule.ID, false)
	require.NoError(t, err)
	require.Len(t, allTargets, 1)

	require.NoError(t, service.DeleteTargetsByRuleID(rule.ID))
	allTargets, err = service.ListTargets(rule.ID, false)
	require.NoError(t, err)
	require.Empty(t, allTargets)

}
