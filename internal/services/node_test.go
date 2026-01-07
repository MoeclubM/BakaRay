package services

import (
	"encoding/json"
	"testing"
	"time"

	"bakaray/internal/models"

	"github.com/stretchr/testify/require"
)

// TestCreateNode 测试节点创建
func TestCreateNode(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, nil)

	t.Run("成功创建节点", func(t *testing.T) {
		node, err := service.CreateNode(
			"TestNode",
			"test.example.com",
			8080,
			"secret123",
			1,
			[]string{"gost", "iptables"},
			1.5,
			"US",
		)

		require.NoError(t, err)
		require.NotNil(t, node)
		require.Equal(t, "TestNode", node.Name)
		require.Equal(t, "test.example.com", node.Host)
		require.Equal(t, 8080, node.Port)
		require.Equal(t, "secret123", node.Secret)
		require.Equal(t, "offline", node.Status)
		require.Equal(t, uint(1), node.NodeGroupID)
		require.Equal(t, 1.5, node.Multiplier)
		require.Equal(t, "US", node.Region)
		require.NotZero(t, node.ID)
	})

	t.Run("节点参数验证", func(t *testing.T) {
		// 测试创建带空协议的节点
		node, err := service.CreateNode(
			"EmptyProtocolNode",
			"empty.example.com",
			9090,
			"secret456",
			2,
			[]string{},
			1.0,
			"CN",
		)

		require.NoError(t, err)
		require.NotNil(t, node)
		require.Equal(t, "EmptyProtocolNode", node.Name)
		require.Equal(t, 1.0, node.Multiplier)
	})

	t.Run("数据库错误时返回错误", func(t *testing.T) {
		// 使用已关闭的数据库触发错误
		cleanupTestDB(db)
		_, err := service.CreateNode(
			"ErrorNode",
			"error.example.com",
			8080,
			"secret",
			1,
			[]string{"gost"},
			1.0,
			"US",
		)

		require.Error(t, err)
	})
}

// TestGetNodeByID 测试根据ID获取节点
func TestGetNodeByID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, nil)
	createdNode := createTestNodeFull(t, db, "GetByIDNode", "getbyid.test.com", 8080, "online")

	t.Run("成功获取节点", func(t *testing.T) {
		node, err := service.GetNodeByID(createdNode.ID)

		require.NoError(t, err)
		require.NotNil(t, node)
		require.Equal(t, createdNode.ID, node.ID)
		require.Equal(t, "GetByIDNode", node.Name)
		require.Equal(t, "online", node.Status)
	})

	t.Run("节点不存在时返回错误", func(t *testing.T) {
		node, err := service.GetNodeByID(99999)

		require.Error(t, err)
		require.Equal(t, ErrNodeNotFound, err)
		require.Nil(t, node)
	})

	t.Run("无效ID时返回错误", func(t *testing.T) {
		node, err := service.GetNodeByID(0)

		require.Error(t, err)
		require.Nil(t, node)
	})
}

// TestUpdateNodeStatus 测试节点状态更新
func TestUpdateNodeStatus(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, nil)
	createdNode := createTestNodeFull(t, db, "StatusNode", "status.test.com", 8080, "offline")

	t.Run("更新为online", func(t *testing.T) {
		err := service.UpdateNodeStatus(createdNode.ID, "online")

		require.NoError(t, err)

		// 验证更新后的状态
		updatedNode, err := service.GetNodeByID(createdNode.ID)
		require.NoError(t, err)
		require.Equal(t, "online", updatedNode.Status)
		require.NotNil(t, updatedNode.LastSeen)
	})

	t.Run("更新为offline", func(t *testing.T) {
		err := service.UpdateNodeStatus(createdNode.ID, "offline")

		require.NoError(t, err)

		// 验证更新后的状态
		updatedNode, err := service.GetNodeByID(createdNode.ID)
		require.NoError(t, err)
		require.Equal(t, "offline", updatedNode.Status)
	})

	t.Run("更新不存在的节点", func(t *testing.T) {
		err := service.UpdateNodeStatus(99999, "online")

		require.Error(t, err)
	})
}

// TestListNodes 测试节点列表分页
func TestListNodes(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, nil)
	_ = createTestNodes(t, db, 10)

	t.Run("获取所有节点", func(t *testing.T) {
		nodes, total := service.ListNodes(1, 10, "")

		require.Len(t, nodes, 10)
		require.Equal(t, int64(10), total)
	})

	t.Run("分页获取节点", func(t *testing.T) {
		page1, total1 := service.ListNodes(1, 5, "")
		page2, total2 := service.ListNodes(2, 5, "")

		require.Len(t, page1, 5)
		require.Len(t, page2, 5)
		require.Equal(t, int64(10), total1)
		require.Equal(t, int64(10), total2)
		require.NotEqual(t, page1[0].ID, page2[0].ID)
	})

	t.Run("按状态筛选online", func(t *testing.T) {
		// 创建指定状态的节点
		createTestNodeFull(t, db, "OnlineNode1", "online1.test.com", 9001, "online")
		createTestNodeFull(t, db, "OnlineNode2", "online2.test.com", 9002, "online")

		nodes, total := service.ListNodes(1, 10, "online")

		// 注意：createTestNodes创建了5个online节点和5个offline节点
		require.GreaterOrEqual(t, len(nodes), 2)
		require.Equal(t, int64(7), total) // 5 + 2
		for _, node := range nodes {
			require.Equal(t, "online", node.Status)
		}
	})

	t.Run("按状态筛选offline", func(t *testing.T) {
		nodes, total := service.ListNodes(1, 10, "offline")

		require.Equal(t, int64(5), total) // 从createTestNodes来的5个offline节点
		for _, node := range nodes {
			require.Equal(t, "offline", node.Status)
		}
	})

	t.Run("无效页码返回空结果", func(t *testing.T) {
		nodes, total := service.ListNodes(100, 10, "")

		require.Len(t, nodes, 0)
		require.Equal(t, int64(10), total)
	})
}

// TestCountNodes 测试节点计数
func TestCountNodes(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, nil)

	t.Run("空数据库返回0", func(t *testing.T) {
		count := service.CountNodes()
		require.Equal(t, int64(0), count)
	})

	t.Run("创建节点后计数增加", func(t *testing.T) {
		initialCount := service.CountNodes()
		_ = createTestNodeFull(t, db, "CountNode1", "count1.test.com", 8080, "offline")
		_ = createTestNodeFull(t, db, "CountNode2", "count2.test.com", 8081, "offline")

		newCount := service.CountNodes()
		require.Equal(t, initialCount+2, newCount)
	})
}

// TestSaveProbeData 测试探针数据保存
func TestSaveProbeData(t *testing.T) {
	redisClient := setupTestRedis(t)
	if redisClient == nil {
		t.Skip("Redis not available")
	}
	defer cleanupTestRedis(redisClient)

	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, redisClient)
	testNode := createTestNodeFull(t, db, "ProbeNode", "probe.test.com", 8080, "online")

	probe := &models.ProbeData{
		Timestamp: time.Now().Unix(),
		CPU: models.CPUInfo{
			UsagePercent: 45.5,
			Cores:        4,
		},
		Memory: models.MemoryInfo{
			Total:       8000000000,
			Used:        4000000000,
			UsagePercent: 50.0,
		},
		Network: []models.NetworkInfo{
			{
				Name:    "eth0",
				RxBytes: 1000000,
				TxBytes: 2000000,
				RxSpeed: 100000,
				TxSpeed: 200000,
			},
		},
	}

	t.Run("成功保存探针数据", func(t *testing.T) {
		err := service.SaveProbeData(testNode.ID, probe)

		require.NoError(t, err)

		// 验证数据已保存
		savedProbe, err := service.GetProbeData(testNode.ID)
		require.NoError(t, err)
		require.NotNil(t, savedProbe)
		require.Equal(t, probe.Timestamp, savedProbe.Timestamp)
		require.Equal(t, probe.CPU.UsagePercent, savedProbe.CPU.UsagePercent)
		require.Equal(t, probe.CPU.Cores, savedProbe.CPU.Cores)
		require.Equal(t, probe.Memory.Total, savedProbe.Memory.Total)
		require.Len(t, savedProbe.Network, 1)
	})

	t.Run("Redis为空时跳过", func(t *testing.T) {
		serviceNoRedis := NewNodeService(db, nil)
		err := serviceNoRedis.SaveProbeData(testNode.ID, probe)

		// 应该不返回错误，直接跳过
		require.NoError(t, err)
	})
}

// TestGetProbeData 测试探针数据获取
func TestGetProbeData(t *testing.T) {
	redisClient := setupTestRedis(t)
	if redisClient == nil {
		t.Skip("Redis not available")
	}
	defer cleanupTestRedis(redisClient)

	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, redisClient)
	testNode := createTestNodeFull(t, db, "GetProbeNode", "getprobe.test.com", 8080, "online")

	t.Run("成功获取探针数据", func(t *testing.T) {
		probe := &models.ProbeData{
			Timestamp: time.Now().Unix(),
			CPU: models.CPUInfo{
				UsagePercent: 30.0,
				Cores:        8,
			},
			Memory: models.MemoryInfo{
				Total:       16000000000,
				Used:        8000000000,
				UsagePercent: 50.0,
			},
		}

		err := service.SaveProbeData(testNode.ID, probe)
		require.NoError(t, err)

		savedProbe, err := service.GetProbeData(testNode.ID)
		require.NoError(t, err)
		require.NotNil(t, savedProbe)
		require.Equal(t, probe.CPU.UsagePercent, savedProbe.CPU.UsagePercent)
	})

	t.Run("数据不存在时返回错误", func(t *testing.T) {
		probe, err := service.GetProbeData(99999)

		require.Error(t, err)
		require.Nil(t, probe)
	})

	t.Run("Redis为空时返回nil", func(t *testing.T) {
		serviceNoRedis := NewNodeService(db, nil)
		probe, err := serviceNoRedis.GetProbeData(testNode.ID)

		require.NoError(t, err)
		require.Nil(t, probe)
	})
}

// TestDeleteNode 测试节点删除
func TestDeleteNode(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, nil)
	testNode := createTestNodeFull(t, db, "DeleteNode", "delete.test.com", 8080, "online")
	nodeID := testNode.ID

	t.Run("成功删除节点", func(t *testing.T) {
		err := service.DeleteNode(nodeID)

		require.NoError(t, err)

		// 验证节点已被删除
		_, err = service.GetNodeByID(nodeID)
		require.Error(t, err)
		require.Equal(t, ErrNodeNotFound, err)
	})

	t.Run("删除不存在的节点", func(t *testing.T) {
		err := service.DeleteNode(99999)

		require.Error(t, err)
	})
}

// TestUpdateNode 测试节点更新
func TestUpdateNode(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, nil)
	testNode := createTestNodeFull(t, db, "UpdateNode", "update.test.com", 8080, "offline")

	t.Run("更新节点名称", func(t *testing.T) {
		err := service.UpdateNode(testNode.ID, map[string]interface{}{
			"name": "UpdatedNodeName",
		})

		require.NoError(t, err)

		updatedNode, err := service.GetNodeByID(testNode.ID)
		require.NoError(t, err)
		require.Equal(t, "UpdatedNodeName", updatedNode.Name)
	})

	t.Run("更新节点状态", func(t *testing.T) {
		err := service.UpdateNode(testNode.ID, map[string]interface{}{
			"status": "online",
		})

		require.NoError(t, err)

		updatedNode, err := service.GetNodeByID(testNode.ID)
		require.NoError(t, err)
		require.Equal(t, "online", updatedNode.Status)
	})

	t.Run("更新端口号", func(t *testing.T) {
		err := service.UpdateNode(testNode.ID, map[string]interface{}{
			"port": 9090,
		})

		require.NoError(t, err)

		updatedNode, err := service.GetNodeByID(testNode.ID)
		require.NoError(t, err)
		require.Equal(t, 9090, updatedNode.Port)
	})

	t.Run("更新端口号为字符串", func(t *testing.T) {
		err := service.UpdateNode(testNode.ID, map[string]interface{}{
			"port": "8888",
		})

		require.NoError(t, err)

		updatedNode, err := service.GetNodeByID(testNode.ID)
		require.NoError(t, err)
		require.Equal(t, 8888, updatedNode.Port)
	})

	t.Run("更新协议列表", func(t *testing.T) {
		err := service.UpdateNode(testNode.ID, map[string]interface{}{
			"protocols": []string{"gost", "socks5", "http"},
		})

		require.NoError(t, err)

		updatedNode, err := service.GetNodeByID(testNode.ID)
		require.NoError(t, err)
		require.Equal(t, models.StringSlice{"gost", "socks5", "http"}, updatedNode.Protocols)
	})

	t.Run("更新协议列表为字符串", func(t *testing.T) {
		err := service.UpdateNode(testNode.ID, map[string]interface{}{
			"protocols": `["wireguard","ss"]`,
		})

		require.NoError(t, err)

		updatedNode, err := service.GetNodeByID(testNode.ID)
		require.NoError(t, err)
		require.Equal(t, models.StringSlice{"wireguard", "ss"}, updatedNode.Protocols)
	})

	t.Run("更新节点组ID", func(t *testing.T) {
		err := service.UpdateNode(testNode.ID, map[string]interface{}{
			"node_group_id": "5",
		})

		require.NoError(t, err)

		updatedNode, err := service.GetNodeByID(testNode.ID)
		require.NoError(t, err)
		require.Equal(t, uint(5), updatedNode.NodeGroupID)
	})

	t.Run("批量更新节点属性", func(t *testing.T) {
		err := service.UpdateNode(testNode.ID, map[string]interface{}{
			"name":        "BatchUpdateNode",
			"host":        "batch.example.com",
			"multiplier":  2.5,
			"region":      "EU",
			"node_group_id": 3,
		})

		require.NoError(t, err)

		updatedNode, err := service.GetNodeByID(testNode.ID)
		require.NoError(t, err)
		require.Equal(t, "BatchUpdateNode", updatedNode.Name)
		require.Equal(t, "batch.example.com", updatedNode.Host)
		require.Equal(t, 2.5, updatedNode.Multiplier)
		require.Equal(t, "EU", updatedNode.Region)
		require.Equal(t, uint(3), updatedNode.NodeGroupID)
	})

	t.Run("更新不存在的节点", func(t *testing.T) {
		err := service.UpdateNode(99999, map[string]interface{}{
			"name": "NonExistent",
		})

		require.Error(t, err)
	})
}

// TestComputeTrafficDeltas 测试流量增量计算
func TestComputeTrafficDeltas(t *testing.T) {
	redisClient := setupTestRedis(t)
	if redisClient == nil {
		t.Skip("Redis not available")
	}
	defer cleanupTestRedis(redisClient)

	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, redisClient)
	testNode := createTestNodeFull(t, db, "TrafficNode", "traffic.test.com", 8080, "online")

	t.Run("计算流量增量", func(t *testing.T) {
		// 模拟节点上报的流量统计
		stats := map[string]int64{
			"rule_1_in":  1000,
			"rule_1_out": 2000,
			"rule_2_in":  3000,
			"rule_2_out": 4000,
		}

		deltas, err := service.ComputeTrafficDeltas(testNode.ID, stats)

		require.NoError(t, err)
		require.NotNil(t, deltas)
		require.Len(t, deltas, 2)

		// 第一次上报，增量应等于上报值
		require.Equal(t, int64(1000), deltas[1].BytesIn)
		require.Equal(t, int64(2000), deltas[1].BytesOut)
		require.Equal(t, int64(3000), deltas[2].BytesIn)
		require.Equal(t, int64(4000), deltas[2].BytesOut)
	})

	t.Run("Redis为空时返回空map", func(t *testing.T) {
		serviceNoRedis := NewNodeService(db, nil)
		deltas, err := serviceNoRedis.ComputeTrafficDeltas(testNode.ID, map[string]int64{
			"rule_1_in": 1000,
		})

		require.NoError(t, err)
		require.NotNil(t, deltas)
		require.Empty(t, deltas)
	})

	t.Run("无效的stats返回空map", func(t *testing.T) {
		stats := map[string]int64{
			"invalid_key": 1000,
			"rule_in":     2000,
			"rule_abc_in": 3000,
		}

		deltas, err := service.ComputeTrafficDeltas(testNode.ID, stats)

		require.NoError(t, err)
		require.NotNil(t, deltas)
		require.Empty(t, deltas)
	})
}

// TestGetAllowedGroups 测试获取节点允许的用户组
func TestGetAllowedGroups(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, nil)
	testNode := createTestNodeFull(t, db, "AllowedGroupsNode", "allowed.test.com", 8080, "online")

	// 添加节点-用户组关联
	err := db.Create(&models.NodeAllowedGroup{
		NodeID:      testNode.ID,
		UserGroupID: 1,
	}).Error
	require.NoError(t, err)

	err = db.Create(&models.NodeAllowedGroup{
		NodeID:      testNode.ID,
		UserGroupID: 2,
	}).Error
	require.NoError(t, err)

	t.Run("成功获取允许的用户组", func(t *testing.T) {
		groups, err := service.GetAllowedGroups(testNode.ID)

		require.NoError(t, err)
		require.Len(t, groups, 2)
		require.Contains(t, groups, uint(1))
		require.Contains(t, groups, uint(2))
	})

	t.Run("无关联时返回空切片", func(t *testing.T) {
		anotherNode := createTestNodeFull(t, db, "NoGroupsNode", "nogroups.test.com", 8081, "online")

		groups, err := service.GetAllowedGroups(anotherNode.ID)

		require.NoError(t, err)
		require.Empty(t, groups)
	})
}

// TestSetAllowedGroups 测试设置节点允许的用户组
func TestSetAllowedGroups(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, nil)
	testNode := createTestNodeFull(t, db, "SetGroupsNode", "setgroups.test.com", 8080, "online")

	t.Run("设置允许的用户组", func(t *testing.T) {
		groupIDs := []uint{3, 4, 5}
		err := service.SetAllowedGroups(testNode.ID, groupIDs)

		require.NoError(t, err)

		groups, err := service.GetAllowedGroups(testNode.ID)
		require.NoError(t, err)
		require.Len(t, groups, 3)
		require.Contains(t, groups, uint(3))
		require.Contains(t, groups, uint(4))
		require.Contains(t, groups, uint(5))
	})

	t.Run("覆盖已有的用户组", func(t *testing.T) {
		// 先添加一些关联
		_ = service.SetAllowedGroups(testNode.ID, []uint{1, 2})

		// 覆盖为新的关联
		newGroupIDs := []uint{6, 7}
		err := service.SetAllowedGroups(testNode.ID, newGroupIDs)

		require.NoError(t, err)

		groups, err := service.GetAllowedGroups(testNode.ID)
		require.NoError(t, err)
		require.Len(t, groups, 2)
		require.Contains(t, groups, uint(6))
		require.Contains(t, groups, uint(7))
		require.NotContains(t, groups, uint(1))
		require.NotContains(t, groups, uint(2))
	})

	t.Run("清空用户组", func(t *testing.T) {
		err := service.SetAllowedGroups(testNode.ID, []uint{})

		require.NoError(t, err)

		groups, err := service.GetAllowedGroups(testNode.ID)
		require.NoError(t, err)
		require.Empty(t, groups)
	})
}

// TestListRulesByNode 测试获取节点的规则列表
func TestListRulesByNode(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(db)

	service := NewNodeService(db, nil)
	testNode := createTestNodeFull(t, db, "RulesNode", "rules.test.com", 8080, "online")

	// 创建测试规则
	rules := []models.ForwardingRule{
		{
			NodeID:     testNode.ID,
			Name:       "Rule1",
			Protocol:   "gost",
			Enabled:    true,
			ListenPort: 8081,
		},
		{
			NodeID:     testNode.ID,
			Name:       "Rule2",
			Protocol:   "iptables",
			Enabled:    false,
			ListenPort: 8082,
		},
		{
			NodeID:     testNode.ID,
			Name:       "Rule3",
			Protocol:   "gost",
			Enabled:    true,
			ListenPort: 8083,
		},
	}

	for _, rule := range rules {
		err := db.Create(&rule).Error
		require.NoError(t, err)
	}

	t.Run("获取所有规则", func(t *testing.T) {
		result, err := service.ListRulesByNode(testNode.ID, false)

		require.NoError(t, err)
		require.Len(t, result, 3)
	})

	t.Run("仅获取启用的规则", func(t *testing.T) {
		result, err := service.ListRulesByNode(testNode.ID, true)

		require.NoError(t, err)
		require.Len(t, result, 2)
		for _, rule := range result {
			require.True(t, rule.Enabled)
		}
	})

	t.Run("节点无规则时返回空切片", func(t *testing.T) {
		anotherNode := createTestNodeFull(t, db, "NoRulesNode", "norules.test.com", 8084, "online")

		result, err := service.ListRulesByNode(anotherNode.ID, false)

		require.NoError(t, err)
		require.Empty(t, result)
	})
}

// TestProbeDataJSONMarshal 测试探针数据JSON序列化
func TestProbeDataJSONMarshal(t *testing.T) {
	probe := &models.ProbeData{
		Timestamp: time.Now().Unix(),
		CPU: models.CPUInfo{
			UsagePercent: 75.5,
			Cores:        16,
		},
		Memory: models.MemoryInfo{
			Total:       32000000000,
			Used:        24000000000,
			UsagePercent: 75.0,
		},
		Network: []models.NetworkInfo{
			{Name: "eth0", RxBytes: 100000, TxBytes: 200000, RxSpeed: 10000, TxSpeed: 20000},
			{Name: "eth1", RxBytes: 300000, TxBytes: 400000, RxSpeed: 30000, TxSpeed: 40000},
		},
	}

	data, err := json.Marshal(probe)
	require.NoError(t, err)

	var unmarshaled models.ProbeData
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	require.Equal(t, probe.Timestamp, unmarshaled.Timestamp)
	require.Equal(t, probe.CPU.UsagePercent, unmarshaled.CPU.UsagePercent)
	require.Equal(t, probe.CPU.Cores, unmarshaled.CPU.Cores)
	require.Equal(t, probe.Memory.Total, unmarshaled.Memory.Total)
	require.Len(t, unmarshaled.Network, 2)
}
