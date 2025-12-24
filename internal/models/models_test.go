package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserJSON(t *testing.T) {
	user := User{
		ID:           1,
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Balance:      10000,
		UserGroupID:  1,
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var unmarshaled User
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if unmarshaled.ID != user.ID {
		t.Errorf("ID = %v, want %v", unmarshaled.ID, user.ID)
	}
	if unmarshaled.Username != user.Username {
		t.Errorf("Username = %v, want %v", unmarshaled.Username, user.Username)
	}
	// PasswordHash 应该被忽略 (json:"-")
	if unmarshaled.PasswordHash != "" {
		t.Errorf("PasswordHash should be empty, got %v", unmarshaled.PasswordHash)
	}
}

func TestNodeStatus(t *testing.T) {
	node := Node{
		ID:      1,
		Name:    "Test Node",
		Host:    "node1.example.com",
		Port:    22,
		Secret:  "secret123",
		Status:  "online",
		Region:  "US",
	}

	if node.Status != "online" {
		t.Errorf("Status = %v, want online", node.Status)
	}
	if node.Region != "US" {
		t.Errorf("Region = %v, want US", node.Region)
	}
}

func TestForwardingRule(t *testing.T) {
	rule := ForwardingRule{
		ID:           1,
		NodeID:       1,
		UserID:       1,
		Name:         "HTTP Proxy",
		Protocol:     "gost",
		Enabled:      true,
		TrafficUsed:  1024,
		TrafficLimit: 1024 * 1024 * 1024, // 1GB
		SpeedLimit:   1024,               // 1Mbps
		ListenPort:   8080,
	}

	if rule.Protocol != "gost" {
		t.Errorf("Protocol = %v, want gost", rule.Protocol)
	}
	if !rule.Enabled {
		t.Errorf("Enabled should be true")
	}
	if rule.ListenPort != 8080 {
		t.Errorf("ListenPort = %v, want 8080", rule.ListenPort)
	}
}

func TestPackageTraffic(t *testing.T) {
	pkg := Package{
		ID:          1,
		Name:        "1GB Monthly",
		Traffic:     1024 * 1024 * 1024, // 1GB
		Price:       1000,               // 10 yuan in cents
		UserGroupID: 1,
	}

	if pkg.Traffic != 1024*1024*1024 {
		t.Errorf("Traffic = %v, want %v", pkg.Traffic, 1024*1024*1024)
	}
	if pkg.Price != 1000 {
		t.Errorf("Price = %v, want 1000", pkg.Price)
	}
}

func TestOrderStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{"pending", "pending"},
		{"success", "success"},
		{"failed", "failed"},
	}

	for _, tt := range tests {
		order := Order{
			ID:      1,
			UserID:  1,
			Amount:  1000,
			Status:  tt.status,
			TradeNo: "TEST123",
		}
		if order.Status != tt.expected {
			t.Errorf("Status = %v, want %v", order.Status, tt.expected)
		}
	}
}

func TestProbeDataJSON(t *testing.T) {
	probe := ProbeData{
		Timestamp: time.Now().Unix(),
		CPU: CPUInfo{
			UsagePercent: 45.5,
			Cores:        4,
		},
		Memory: MemoryInfo{
			Total:        8 * 1024 * 1024 * 1024, // 8GB
			Used:         4 * 1024 * 1024 * 1024, // 4GB
			UsagePercent: 50.0,
		},
		Network: []NetworkInfo{
			{
				Name:    "eth0",
				RxBytes: 1000,
				TxBytes: 2000,
				RxSpeed: 100,
				TxSpeed: 200,
			},
		},
	}

	data, err := json.Marshal(probe)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var unmarshaled ProbeData
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if unmarshaled.CPU.UsagePercent != 45.5 {
		t.Errorf("CPU.UsagePercent = %v, want 45.5", unmarshaled.CPU.UsagePercent)
	}
	if unmarshaled.CPU.Cores != 4 {
		t.Errorf("CPU.Cores = %v, want 4", unmarshaled.CPU.Cores)
	}
	if len(unmarshaled.Network) != 1 {
		t.Errorf("Network length = %v, want 1", len(unmarshaled.Network))
	}
	if unmarshaled.Network[0].Name != "eth0" {
		t.Errorf("Network[0].Name = %v, want eth0", unmarshaled.Network[0].Name)
	}
}

func TestUserRole(t *testing.T) {
	tests := []struct {
		role     string
		expected string
	}{
		{"admin", "admin"},
		{"user", "user"},
	}

	for _, tt := range tests {
		user := User{
			ID:   1,
			Role: tt.role,
		}
		if user.Role != tt.expected {
			t.Errorf("Role = %v, want %v", user.Role, tt.expected)
		}
	}
}

func TestNodeGroupType(t *testing.T) {
	entryGroup := NodeGroup{
		ID:   1,
		Name: "Entry Nodes",
		Type: "entry",
	}

	targetGroup := NodeGroup{
		ID:   2,
		Name: "Target Nodes",
		Type: "target",
	}

	if entryGroup.Type != "entry" {
		t.Errorf("Type = %v, want entry", entryGroup.Type)
	}
	if targetGroup.Type != "target" {
		t.Errorf("Type = %v, want target", targetGroup.Type)
	}
}
