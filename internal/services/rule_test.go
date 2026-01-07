package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"bakaray/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      db,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	return gormDB, mock
}

func setupTestRuleService(t *testing.T) (*RuleService, sqlmock.Sqlmock) {
	db, mock := setupMockDB(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	svc := NewRuleService(db, rdb)
	return svc, mock
}

func TestCreateRule(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("成功创建规则", func(t *testing.T) {
		rule := &models.ForwardingRule{
			NodeID:       1,
			UserID:       1,
			Name:         "Test Rule",
			Protocol:     "gost",
			ListenPort:   8080,
			Enabled:      true,
			TrafficUsed:  0,
			TrafficLimit: 1024 * 1024 * 1024,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `forwarding_rules`").
			WithArgs(
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				rule.NodeID, rule.UserID, rule.Name, rule.Protocol,
				rule.Enabled, rule.TrafficUsed, rule.TrafficLimit,
				rule.SpeedLimit, rule.Mode, rule.ListenPort,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := svc.CreateRule(rule)
		require.NoError(t, err)
		require.Equal(t, uint(1), rule.ID)
	})

	t.Run("默认值设置", func(t *testing.T) {
		rule := &models.ForwardingRule{
			NodeID:   1,
			UserID:   1,
			Name:     "Default Test Rule",
			Protocol: "gost",
			ListenPort: 9090,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `forwarding_rules`").
			WithArgs(
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				rule.NodeID, rule.UserID, rule.Name, rule.Protocol,
				true,  // 默认 enabled = true
				int64(0),  // 默认 traffic_used = 0
				int64(0),  // 默认 traffic_limit = 0
				int64(0),  // 默认 speed_limit = 0
				"direct", // 默认 mode = "direct"
				rule.ListenPort,
			).
			WillReturnResult(sqlmock.NewResult(2, 1))
		mock.ExpectCommit()

		err := svc.CreateRule(rule)
		require.NoError(t, err)
		require.Equal(t, uint(2), rule.ID)
	})
}

func TestGetRuleByID(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("成功获取规则", func(t *testing.T) {
		expectedRule := models.ForwardingRule{
			ID:           1,
			NodeID:       1,
			UserID:       1,
			Name:         "Test Rule",
			Protocol:     "gost",
			Enabled:      true,
			TrafficUsed:  1024,
			TrafficLimit: 1024 * 1024 * 1024,
			ListenPort:   8080,
		}

		rows := sqlmock.NewRows([]string{"id", "node_id", "user_id", "name", "protocol", "enabled", "traffic_used", "traffic_limit", "speed_limit", "mode", "listen_port", "created_at", "updated_at"}).
			AddRow(
				expectedRule.ID, expectedRule.NodeID, expectedRule.UserID,
				expectedRule.Name, expectedRule.Protocol, expectedRule.Enabled,
				expectedRule.TrafficUsed, expectedRule.TrafficLimit,
				expectedRule.SpeedLimit, expectedRule.Mode, expectedRule.ListenPort,
				time.Now(), time.Now(),
			)

		mock.ExpectQuery("SELECT \\* FROM `forwarding_rules` WHERE id = \\? ORDER BY `forwarding_rules`.`id` LIMIT 1").
			WithArgs(1).
			WillReturnRows(rows)

		rule, err := svc.GetRuleByID(1)
		require.NoError(t, err)
		require.NotNil(t, rule)
		require.Equal(t, "Test Rule", rule.Name)
		require.Equal(t, "gost", rule.Protocol)
	})

	t.Run("规则不存在时返回错误", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM `forwarding_rules` WHERE id = \\? ORDER BY `forwarding_rules`.`id` LIMIT 1").
			WithArgs(999).
			WillReturnError(gorm.ErrRecordNotFound)

		rule, err := svc.GetRuleByID(999)
		require.Error(t, err)
		require.Equal(t, ErrRuleNotFound, err)
		require.Nil(t, rule)
	})
}

func TestUpdateRule(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("更新规则名称", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `forwarding_rules` SET `name`=\\?,`updated_at`=\\? WHERE id = \\?").
			WithArgs("Updated Name", sqlmock.AnyArg, 1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := svc.UpdateRule(1, map[string]interface{}{
			"name": "Updated Name",
		})
		require.NoError(t, err)
	})

	t.Run("更新启用状态", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `forwarding_rules` SET `enabled`=\\?,`updated_at`=\\? WHERE id = \\?").
			WithArgs(false, sqlmock.AnyArg, 1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := svc.UpdateRule(1, map[string]interface{}{
			"enabled": false,
		})
		require.NoError(t, err)
	})

	t.Run("批量更新字段", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `forwarding_rules` SET `enabled`=\\?,`name`=\\?,`updated_at`=\\? WHERE id = \\?").
			WithArgs(true, "Batch Update", sqlmock.AnyArg, 1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := svc.UpdateRule(1, map[string]interface{}{
			"name":    "Batch Update",
			"enabled": true,
		})
		require.NoError(t, err)
	})
}

func TestDeleteRule(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("成功删除规则", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM `forwarding_rules` WHERE `forwarding_rules`.`id` = \\?").
			WithArgs(1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := svc.DeleteRule(1)
		require.NoError(t, err)
	})

	t.Run("删除不存在的规则", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM `forwarding_rules` WHERE `forwarding_rules`.`id` = \\?").
			WithArgs(999).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		err := svc.DeleteRule(999)
		require.NoError(t, err)
	})
}

func TestListRulesByUser(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("获取用户所有规则", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "node_id", "user_id", "name", "protocol", "enabled", "traffic_used", "traffic_limit", "speed_limit", "mode", "listen_port"}).
			AddRow(1, 1, 1, "Rule 1", "gost", true, 0, 1024*1024*1024, 0, "direct", 8080).
			AddRow(2, 1, 1, "Rule 2", "iptables", true, 512, 2048*1024*1024, 0, "rr", 9090)

		mock.ExpectQuery("SELECT \\* FROM `forwarding_rules` WHERE user_id = \\?").
			WithArgs(1).
			WillReturnRows(rows)

		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `forwarding_rules` WHERE user_id = \\?").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		rules, total := svc.ListRulesByUser(1, 1, 10)
		require.Len(t, rules, 2)
		require.Equal(t, int64(2), total)
	})

	t.Run("按启用状态筛选", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "node_id", "user_id", "name", "protocol", "enabled", "traffic_used", "traffic_limit", "speed_limit", "mode", "listen_port"}).
			AddRow(1, 1, 1, "Enabled Rule", "gost", true, 0, 1024*1024*1024, 0, "direct", 8080)

		mock.ExpectQuery("SELECT \\* FROM `forwarding_rules` WHERE user_id = \\? AND enabled = \\?").
			WithArgs(1, true).
			WillReturnRows(rows)

		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `forwarding_rules` WHERE user_id = \\? AND enabled = \\?").
			WithArgs(1, true).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rules, total := svc.ListRulesByUser(1, 1, 10)
		require.Len(t, rules, 1)
		require.Equal(t, int64(1), total)
	})

	t.Run("分页测试", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "node_id", "user_id", "name", "protocol", "enabled", "traffic_used", "traffic_limit", "speed_limit", "mode", "listen_port"}).
			AddRow(2, 1, 1, "Rule 2", "gost", true, 0, 1024*1024*1024, 0, "direct", 8081)

		mock.ExpectQuery("SELECT \\* FROM `forwarding_rules` WHERE user_id = \\? LIMIT 10 OFFSET 10").
			WithArgs(1).
			WillReturnRows(rows)

		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `forwarding_rules` WHERE user_id = \\?").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

		rules, total := svc.ListRulesByUser(1, 2, 10)
		require.Len(t, rules, 1)
		require.Equal(t, int64(15), total)
	})
}

func TestGetUserTrafficUsage(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("获取用户流量使用统计", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"total"}).
			AddRow(2048)

		mock.ExpectQuery("SELECT COALESCE\\(SUM\\(traffic_used\\),0\\) FROM `forwarding_rules` WHERE user_id = \\?").
			WithArgs(1).
			WillReturnRows(rows)

		total, err := svc.GetUserTrafficUsed(1)
		require.NoError(t, err)
		require.Equal(t, int64(2048), total)
	})

	t.Run("用户无规则时返回0", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"total"}).
			AddRow(0)

		mock.ExpectQuery("SELECT COALESCE\\(SUM\\(traffic_used\\),0\\) FROM `forwarding_rules` WHERE user_id = \\?").
			WithArgs(999).
			WillReturnRows(rows)

		total, err := svc.GetUserTrafficUsed(999)
		require.NoError(t, err)
		require.Equal(t, int64(0), total)
	})

	t.Run("数据库错误时返回错误", func(t *testing.T) {
		mock.ExpectQuery("SELECT COALESCE\\(SUM\\(traffic_used\\),0\\) FROM `forwarding_rules` WHERE user_id = \\?").
			WithArgs(1).
			WillReturnError(errors.New("database error"))

		total, err := svc.GetUserTrafficUsed(1)
		require.Error(t, err)
		require.Equal(t, int64(0), total)
	})
}

func TestUpdateTrafficUsed(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("成功更新已用流量", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `forwarding_rules` SET `traffic_used`=traffic_used \\+ \\?,`updated_at`=\\? WHERE id = \\?").
			WithArgs(1024, sqlmock.AnyArg, 1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := svc.UpdateTrafficUsed(1, 1024)
		require.NoError(t, err)
	})
}

func TestTrafficLog(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("创建流量日志", func(t *testing.T) {
		log := &models.TrafficLog{
			RuleID:    1,
			NodeID:    1,
			BytesIn:   1024,
			BytesOut:  2048,
			Timestamp: time.Now(),
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `traffic_logs`").
			WithArgs(
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				log.RuleID, log.NodeID, log.BytesIn, log.BytesOut,
				log.Timestamp,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := svc.CreateTrafficLog(log)
		require.NoError(t, err)
	})
}

func TestGetUserTrafficStats(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("获取用户流量统计", func(t *testing.T) {
		since := time.Now().Add(-24 * time.Hour)

		rows := sqlmock.NewRows([]string{"bytes_in", "bytes_out"}).
			AddRow(1024, 2048)

		mock.ExpectQuery("SELECT COALESCE\\(SUM\\(traffic_logs\\.bytes_in\\),0\\) AS bytes_in, COALESCE\\(SUM\\(traffic_logs\\.bytes_out\\),0\\) AS bytes_out FROM `traffic_logs` JOIN forwarding_rules ON forwarding_rules\\.id = traffic_logs\\.rule_id WHERE forwarding_rules\\.user_id = \\? AND traffic_logs\\.timestamp >= \\?").
			WithArgs(1, since).
			WillReturnRows(rows)

		bytesIn, bytesOut, err := svc.GetUserTrafficStats(1, since)
		require.NoError(t, err)
		require.Equal(t, int64(1024), bytesIn)
		require.Equal(t, int64(2048), bytesOut)
	})
}

func TestTargets(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("获取规则的目标列表", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "rule_id", "host", "port", "weight", "enabled"}).
			AddRow(1, 1, "target1.example.com", 80, 1, true).
			AddRow(2, 1, "target2.example.com", 80, 2, true)

		mock.ExpectQuery("SELECT \\* FROM `targets` WHERE rule_id = \\? AND enabled = \\?").
			WithArgs(1, true).
			WillReturnRows(rows)

		targets, err := svc.GetTargets(1)
		require.NoError(t, err)
		require.Len(t, targets, 2)
	})

	t.Run("添加转发目标", func(t *testing.T) {
		target := &models.Target{
			RuleID:  1,
			Host:    "newtarget.example.com",
			Port:    443,
			Weight:  1,
			Enabled: true,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `targets`").
			WithArgs(
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				target.RuleID, target.Host, target.Port, target.Weight, target.Enabled,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := svc.AddTarget(target)
		require.NoError(t, err)
	})

	t.Run("删除转发目标", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM `targets` WHERE `targets`.`id` = \\?").
			WithArgs(1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := svc.DeleteTarget(1)
		require.NoError(t, err)
	})

	t.Run("删除规则的所有目标", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM `targets` WHERE rule_id = \\?").
			WithArgs(1).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectCommit()

		err := svc.DeleteTargetsByRuleID(1)
		require.NoError(t, err)
	})
}

func TestGostRule(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("获取gost规则配置", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "rule_id", "transport", "tls", "chain", "timeout"}).
			AddRow(1, 1, "tcp", true, "", 30)

		mock.ExpectQuery("SELECT \\* FROM `gost_rules` WHERE rule_id = \\? ORDER BY `gost_rules`.`id` LIMIT 1").
			WithArgs(1).
			WillReturnRows(rows)

		rule, err := svc.GetGostRule(1)
		require.NoError(t, err)
		require.NotNil(t, rule)
		require.Equal(t, "tcp", rule.Transport)
		require.True(t, rule.TLS)
	})

	t.Run("gost规则不存在", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM `gost_rules` WHERE rule_id = \\? ORDER BY `gost_rules`.`id` LIMIT 1").
			WithArgs(999).
			WillReturnError(gorm.ErrRecordNotFound)

		rule, err := svc.GetGostRule(999)
		require.NoError(t, err)
		require.Nil(t, rule)
	})

	t.Run("创建gost规则", func(t *testing.T) {
		rule := &models.GostRule{
			RuleID:    1,
			Transport: "quic",
			TLS:       true,
			Chain:     "chain1",
			Timeout:   60,
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `gost_rules`").
			WithArgs(
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				rule.RuleID, rule.Transport, rule.TLS, rule.Chain, rule.Timeout,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := svc.CreateGostRule(rule)
		require.NoError(t, err)
	})
}

func TestIPTablesRule(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("获取iptables规则配置", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "rule_id", "proto", "snat", "iface"}).
			AddRow(1, 1, "tcp", false, "eth0")

		mock.ExpectQuery("SELECT \\* FROM `iptables_rules` WHERE rule_id = \\? ORDER BY `iptables_rules`.`id` LIMIT 1").
			WithArgs(1).
			WillReturnRows(rows)

		rule, err := svc.GetIPTablesRule(1)
		require.NoError(t, err)
		require.NotNil(t, rule)
		require.Equal(t, "tcp", rule.Proto)
		require.False(t, rule.SNAT)
	})

	t.Run("iptables规则不存在", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM `iptables_rules` WHERE rule_id = \\? ORDER BY `iptables_rules`.`id` LIMIT 1").
			WithArgs(999).
			WillReturnError(gorm.ErrRecordNotFound)

		rule, err := svc.GetIPTablesRule(999)
		require.NoError(t, err)
		require.Nil(t, rule)
	})

	t.Run("创建iptables规则", func(t *testing.T) {
		rule := &models.IPTablesRule{
			RuleID: 1,
			Proto:  "udp",
			SNAT:   true,
			Iface:  "eth1",
		}

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `iptables_rules`").
			WithArgs(
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				rule.RuleID, rule.Proto, rule.SNAT, rule.Iface,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		err := svc.CreateIPTablesRule(rule)
		require.NoError(t, err)
	})
}

func TestListAllRules(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("获取所有规则", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "node_id", "user_id", "name", "protocol", "enabled", "traffic_used", "traffic_limit", "speed_limit", "mode", "listen_port"}).
			AddRow(1, 1, 1, "Rule 1", "gost", true, 0, 1024*1024*1024, 0, "direct", 8080).
			AddRow(2, 1, 2, "Rule 2", "iptables", true, 0, 2048*1024*1024, 0, "rr", 9090)

		mock.ExpectQuery("SELECT \\* FROM `forwarding_rules`").
			WillReturnRows(rows)

		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `forwarding_rules`").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		rules, total := svc.ListAllRules(1, 10, 0, 0)
		require.Len(t, rules, 2)
		require.Equal(t, int64(2), total)
	})

	t.Run("按节点ID筛选", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "node_id", "user_id", "name", "protocol", "enabled", "traffic_used", "traffic_limit", "speed_limit", "mode", "listen_port"}).
			AddRow(1, 1, 1, "Rule 1", "gost", true, 0, 1024*1024*1024, 0, "direct", 8080)

		mock.ExpectQuery("SELECT \\* FROM `forwarding_rules` WHERE node_id = \\?").
			WithArgs(1).
			WillReturnRows(rows)

		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `forwarding_rules` WHERE node_id = \\?").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rules, total := svc.ListAllRules(1, 10, 1, 0)
		require.Len(t, rules, 1)
		require.Equal(t, int64(1), total)
	})

	t.Run("按用户ID筛选", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "node_id", "user_id", "name", "protocol", "enabled", "traffic_used", "traffic_limit", "speed_limit", "mode", "listen_port"}).
			AddRow(1, 1, 1, "User 1 Rule", "gost", true, 0, 1024*1024*1024, 0, "direct", 8080)

		mock.ExpectQuery("SELECT \\* FROM `forwarding_rules` WHERE user_id = \\?").
			WithArgs(1).
			WillReturnRows(rows)

		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `forwarding_rules` WHERE user_id = \\?").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		rules, total := svc.ListAllRules(1, 10, 0, 1)
		require.Len(t, rules, 1)
		require.Equal(t, int64(1), total)
	})
}

func TestCountAllRules(t *testing.T) {
	svc, mock := setupTestRuleService(t)

	t.Run("统计所有规则数量", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"count"}).AddRow(100)

		mock.ExpectQuery("SELECT count\\(\\*\\) FROM `forwarding_rules`").
			WillReturnRows(rows)

		total := svc.CountAllRules()
		require.Equal(t, int64(100), total)
	})
}

// TestRuleServiceWithRedis 测试带有Redis模拟的服务
func TestRuleServiceWithRedis(t *testing.T) {
	db, _ := setupMockDB(t)

	// 创建一个不连接到真实Redis服务器的Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// 使用context来模拟Redis连接错误
	ctx := context.Background()
	_, _ = rdb.Ping(ctx).Result()
	// 这个测试不需要Redis真正连接，我们只是验证服务创建

	svc := NewRuleService(db, rdb)
	require.NotNil(t, svc)

	// 验证RuleService结构正确
	require.NotNil(t, svc.db)
	require.NotNil(t, svc.redis)
}
