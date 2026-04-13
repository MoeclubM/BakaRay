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
	entryNode := &models.Node{ID: 1, Protocols: models.StringSlice{"tcp", "udp", "quic"}}
	exitNode := &models.Node{ID: 2, Protocols: models.StringSlice{"ws", "grpc", "quic"}}

	t.Run("rejects speed limit", func(t *testing.T) {
		_, err := normalizeAndValidateRuleSpec(
			entryNode,
			nil,
			"tcp",
			8081,
			true,
			0,
			64,
			"direct",
			[]TargetRequest{{Host: "127.0.0.1", Port: 80, Weight: 1, Enabled: true}},
			false,
			0,
			"",
			0,
			nil,
			nil,
			0,
		)
		if err == nil || !strings.Contains(err.Error(), "暂不支持限速") {
			t.Fatalf("expected speed limit validation error, got %v", err)
		}
	})

	t.Run("rejects rr with less than two enabled targets", func(t *testing.T) {
		_, err := normalizeAndValidateRuleSpec(
			entryNode,
			nil,
			"tcp",
			8082,
			true,
			0,
			0,
			"rr",
			[]TargetRequest{{Host: "127.0.0.1", Port: 80, Weight: 1, Enabled: true}},
			false,
			0,
			"",
			0,
			nil,
			nil,
			0,
		)
		if err == nil || !strings.Contains(err.Error(), "至少需要两个启用目标") {
			t.Fatalf("expected rr target count validation error, got %v", err)
		}
	})

	t.Run("detects same port and layer4 conflict", func(t *testing.T) {
		_, err := normalizeAndValidateRuleSpec(
			entryNode,
			nil,
			"tcp",
			8083,
			true,
			0,
			0,
			"direct",
			[]TargetRequest{{Host: "127.0.0.1", Port: 80, Weight: 1, Enabled: true}},
			false,
			0,
			"",
			0,
			[]existingRuleConflict{
				{ID: 2, Port: 8083, Enabled: true, Layer4: "tcp"},
			},
			nil,
			0,
		)
		if err == nil || !strings.Contains(err.Error(), "监听已存在") {
			t.Fatalf("expected listen port conflict error, got %v", err)
		}
	})

	t.Run("rejects unsupported direct protocol", func(t *testing.T) {
		_, err := normalizeAndValidateRuleSpec(
			entryNode,
			nil,
			"ws",
			8084,
			true,
			0,
			0,
			"direct",
			[]TargetRequest{{Host: "127.0.0.1", Port: 80, Weight: 1, Enabled: true}},
			false,
			0,
			"",
			0,
			nil,
			nil,
			0,
		)
		if err == nil || !strings.Contains(err.Error(), "仅支持 TCP 或 UDP") {
			t.Fatalf("expected unsupported protocol validation error, got %v", err)
		}
	})

	t.Run("accepts tunnel rule", func(t *testing.T) {
		spec, err := normalizeAndValidateRuleSpec(
			entryNode,
			exitNode,
			"udp",
			8085,
			true,
			0,
			0,
			"direct",
			[]TargetRequest{{Host: "127.0.0.1", Port: 53, Weight: 1, Enabled: true}},
			true,
			exitNode.ID,
			"quic",
			9443,
			nil,
			nil,
			0,
		)
		if err != nil {
			t.Fatalf("expected tunnel rule to pass validation, got %v", err)
		}
		if !spec.TunnelEnabled || spec.ExitNodeID != exitNode.ID || spec.TunnelProtocol != "quic" || spec.TunnelPort != 9443 {
			t.Fatalf("unexpected tunnel spec: %#v", spec)
		}
	})

	t.Run("rejects tunnel when entry node does not support protocol", func(t *testing.T) {
		_, err := normalizeAndValidateRuleSpec(
			entryNode,
			exitNode,
			"tcp",
			8086,
			true,
			0,
			0,
			"direct",
			[]TargetRequest{{Host: "127.0.0.1", Port: 80, Weight: 1, Enabled: true}},
			true,
			exitNode.ID,
			"ws",
			9443,
			nil,
			nil,
			0,
		)
		if err == nil || !strings.Contains(err.Error(), "入口节点未声明支持 ws 隧道") {
			t.Fatalf("expected entry tunnel capability validation error, got %v", err)
		}
	})

	t.Run("rejects exit conflict", func(t *testing.T) {
		_, err := normalizeAndValidateRuleSpec(
			entryNode,
			exitNode,
			"tcp",
			8086,
			true,
			0,
			0,
			"direct",
			[]TargetRequest{{Host: "127.0.0.1", Port: 80, Weight: 1, Enabled: true}},
			true,
			exitNode.ID,
			"ws",
			9444,
			nil,
			[]existingRuleConflict{
				{ID: 3, Port: 9444, Enabled: true, Layer4: "tcp"},
			},
			0,
		)
		if err == nil || !strings.Contains(err.Error(), "出口节点端口 9444") {
			t.Fatalf("expected exit conflict error, got %v", err)
		}
	})
}
