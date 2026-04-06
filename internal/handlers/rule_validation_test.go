package handlers

import (
	"strings"
	"testing"

	"bakaray/internal/models"
)

func TestResolveRuleStateValues(t *testing.T) {
	currentEnabled := false
	currentTrafficLimit := int64(2048)
	currentSpeedLimit := int64(512)

	enabled, trafficLimit, speedLimit := resolveRuleStateValues(nil, nil, nil, currentEnabled, currentTrafficLimit, currentSpeedLimit)
	if enabled != currentEnabled || trafficLimit != currentTrafficLimit || speedLimit != currentSpeedLimit {
		t.Fatalf("expected existing values to be preserved, got enabled=%v traffic=%d speed=%d", enabled, trafficLimit, speedLimit)
	}

	nextEnabled := true
	nextTrafficLimit := int64(4096)
	nextSpeedLimit := int64(1024)

	enabled, trafficLimit, speedLimit = resolveRuleStateValues(&nextEnabled, &nextTrafficLimit, &nextSpeedLimit, currentEnabled, currentTrafficLimit, currentSpeedLimit)
	if !enabled || trafficLimit != nextTrafficLimit || speedLimit != nextSpeedLimit {
		t.Fatalf("expected explicit values to override existing ones, got enabled=%v traffic=%d speed=%d", enabled, trafficLimit, speedLimit)
	}
}

func TestNormalizeAndValidateRuleSpec(t *testing.T) {
	node := &models.Node{Protocols: models.StringSlice{"gost", "iptables"}}

	t.Run("defaults iptables proto to tcp", func(t *testing.T) {
		spec, err := normalizeAndValidateRuleSpec(
			node,
			"iptables",
			8080,
			true,
			0,
			128,
			"direct",
			[]TargetRequest{{Host: "127.0.0.1", Port: 80, Weight: 1, Enabled: true}},
			nil,
			nil,
			nil,
			0,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if spec.IPTablesConfig == nil || spec.IPTablesConfig.Proto != "tcp" {
			t.Fatalf("expected tcp iptables config, got %#v", spec.IPTablesConfig)
		}
	})

	t.Run("rejects gost speed limit", func(t *testing.T) {
		_, err := normalizeAndValidateRuleSpec(
			node,
			"gost",
			8081,
			true,
			0,
			64,
			"direct",
			[]TargetRequest{{Host: "127.0.0.1", Port: 80, Weight: 1, Enabled: true}},
			&GostConfig{Transport: "tcp"},
			nil,
			nil,
			0,
		)
		if err == nil || !strings.Contains(err.Error(), "暂不支持限速") {
			t.Fatalf("expected gost speed limit validation error, got %v", err)
		}
	})

	t.Run("rejects rr with less than two enabled targets", func(t *testing.T) {
		_, err := normalizeAndValidateRuleSpec(
			node,
			"iptables",
			8082,
			true,
			0,
			0,
			"rr",
			[]TargetRequest{{Host: "127.0.0.1", Port: 80, Weight: 1, Enabled: true}},
			nil,
			&IPTablesConfig{Proto: "tcp"},
			nil,
			0,
		)
		if err == nil || !strings.Contains(err.Error(), "至少需要两个启用目标") {
			t.Fatalf("expected rr target count validation error, got %v", err)
		}
	})

	t.Run("detects same port and layer4 conflict", func(t *testing.T) {
		_, err := normalizeAndValidateRuleSpec(
			node,
			"gost",
			8083,
			true,
			0,
			0,
			"direct",
			[]TargetRequest{{Host: "127.0.0.1", Port: 80, Weight: 1, Enabled: true}},
			&GostConfig{Transport: "tcp"},
			nil,
			[]existingRuleConflict{
				{ID: 2, ListenPort: 8083, Enabled: true, Layer4: "tcp"},
			},
			0,
		)
		if err == nil || !strings.Contains(err.Error(), "转发规则已存在") {
			t.Fatalf("expected listen port conflict error, got %v", err)
		}
	})

	t.Run("allows same port on different layer4 protocols", func(t *testing.T) {
		spec, err := normalizeAndValidateRuleSpec(
			node,
			"gost",
			8084,
			true,
			0,
			0,
			"direct",
			[]TargetRequest{{Host: "127.0.0.1", Port: 80, Weight: 1, Enabled: true}},
			&GostConfig{Transport: "tcp"},
			nil,
			[]existingRuleConflict{
				{ID: 2, ListenPort: 8084, Enabled: true, Layer4: "udp"},
			},
			0,
		)
		if err != nil {
			t.Fatalf("expected no conflict across layer4 protocols, got %v", err)
		}
		if spec.GostConfig == nil || spec.GostConfig.Transport != "tcp" {
			t.Fatalf("unexpected gost config: %#v", spec.GostConfig)
		}
	})
}
