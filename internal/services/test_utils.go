package services

import (
	"context"
	"testing"
	"time"

	"bakaray/internal/models"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate all tables
	err = db.AutoMigrate(
		&models.User{},
		&models.Node{},
		&models.NodeAllowedGroup{},
		&models.ForwardingRule{},
		&models.Package{},
		&models.Order{},
		&models.UserGroup{},
		&models.NodeGroup{},
		&models.PaymentConfig{},
	)
	require.NoError(t, err)

	return db
}

// cleanupTestDB cleans up the test database
func cleanupTestDB(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
	}
}

// setupTestRedis creates a fake Redis client for testing
func setupTestRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   15,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis not available, skipping Redis-dependent tests")
		return nil
	}

	cleanRedisKeys(client)
	return client
}

// cleanRedisKeys cleans up test keys from Redis
func cleanRedisKeys(client *redis.Client) {
	ctx := context.Background()
	keys, _ := client.Keys(ctx, "test:*").Result()
	if len(keys) > 0 {
		client.Del(ctx, keys...)
	}
}

// createTestUser creates a test user for testing
func createTestUser(t *testing.T, db *gorm.DB, username string) *models.User {
	user := &models.User{
		Username:    username,
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/nMskyB.3RJHXxkhR6R6J2",
		Balance:     1000,
		Role:        "user",
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

// createTestAdmin creates a test admin user
func createTestAdmin(t *testing.T, db *gorm.DB, username string) *models.User {
	user := &models.User{
		Username:    username,
		PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/nMskyB.3RJHXxkhR6R6J2",
		Balance:     10000,
		Role:        "admin",
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

// createTestNode creates a test node for testing (basic version)
func createTestNode(t *testing.T, db *gorm.DB, name string) *models.Node {
	return createTestNodeFull(t, db, name, "localhost", 8080, "online")
}

// createTestNodeFull creates a test node with full parameters
func createTestNodeFull(t *testing.T, db *gorm.DB, name, host string, port int, status string) *models.Node {
	node := &models.Node{
		Name:        name,
		Host:        host,
		Port:        port,
		Secret:      "test-secret-" + name,
		Status:      status,
		NodeGroupID: 0,
		Protocols:   models.StringSlice([]string{"gost"}),
		Multiplier:  1.0,
		Region:      "Test",
	}
	require.NoError(t, db.Create(node).Error)
	return node
}

// createTestPackage creates a test package for testing
func createTestPackage(t *testing.T, db *gorm.DB, name string) *models.Package {
	pkg := &models.Package{
		Name:    name,
		Traffic: 1000000000,
		Price:   1000,
	}
	require.NoError(t, db.Create(pkg).Error)
	return pkg
}

// createTestRule creates a test forwarding rule
func createTestRule(t *testing.T, db *gorm.DB, nodeID uint, name string) *models.ForwardingRule {
	rule := &models.ForwardingRule{
		NodeID:     nodeID,
		Name:       name,
		Protocol:   "gost",
		Enabled:    true,
		TrafficUsed: 0,
		TrafficLimit: 1000000000,
		SpeedLimit:  1000,
		Mode:       "direct",
		ListenPort: 8000 + int(nodeID),
	}
	require.NoError(t, db.Create(rule).Error)
	return rule
}

// createTestUserGroup creates a test user group
func createTestUserGroup(t *testing.T, db *gorm.DB, name string) *models.UserGroup {
	group := &models.UserGroup{
		Name:        name,
		Description: "Test group " + name,
	}
	require.NoError(t, db.Create(group).Error)
	return group
}

// createTestNodeGroup creates a test node group
func createTestNodeGroup(t *testing.T, db *gorm.DB, name, nodeType string) *models.NodeGroup {
	group := &models.NodeGroup{
		Name:        name,
		Type:        nodeType,
		Description: "Test group " + name,
	}
	require.NoError(t, db.Create(group).Error)
	return group
}

// cleanupTestRedis closes the Redis connection and cleans up
func cleanupTestRedis(client *redis.Client) {
	if client != nil {
		cleanRedisKeys(client)
		client.Close()
	}
}

// createTestNodes creates multiple test nodes
func createTestNodes(t *testing.T, db *gorm.DB, count int) []*models.Node {
	nodes := make([]*models.Node, count)
	for i := 0; i < count; i++ {
		status := "offline"
		if i%2 == 0 {
			status = "online"
		}
		nodes[i] = createTestNodeFull(t, db, "TestNode"+string(rune('A'+i)), "host"+string(rune('A'+i))+".test.com", 8080+i, status)
	}
	return nodes
}
