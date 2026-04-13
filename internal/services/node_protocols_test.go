package services

import "testing"

func TestNormalizeNodeProtocols(t *testing.T) {
	t.Run("filters unsupported and deduplicates", func(t *testing.T) {
		got := NormalizeNodeProtocols([]string{"tcp", "legacy", "quic", "tcp", ""})
		if len(got) != 2 || got[0] != "tcp" || got[1] != "quic" {
			t.Fatalf("unexpected normalized protocols: %#v", got)
		}
	})

	t.Run("applies default linux capability set", func(t *testing.T) {
		got := NormalizeNodeProtocols(nil)
		want := SupportedNodeProtocols()
		if len(got) != len(want) {
			t.Fatalf("unexpected default protocols: %#v", got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("unexpected default protocols: %#v", got)
			}
		}
	})
}

func TestProtocolCategories(t *testing.T) {
	if !IsDirectProtocol("tcp") || !IsDirectProtocol("udp") {
		t.Fatal("expected tcp/udp to be direct protocols")
	}
	if IsDirectProtocol("ws") {
		t.Fatal("did not expect ws to be treated as direct protocol")
	}
	if !IsTunnelProtocol("grpc") || !IsTunnelProtocol("quic") {
		t.Fatal("expected grpc/quic to be tunnel protocols")
	}
	if IsTunnelProtocol("udp") {
		t.Fatal("did not expect udp to be treated as tunnel protocol")
	}
}

func TestNodeSupportsProtocol(t *testing.T) {
	if !NodeSupportsDirectProtocol([]string{"udp"}, "udp") {
		t.Fatal("expected udp direct forwarding to be supported")
	}
	if !NodeSupportsTunnelProtocol([]string{"quic"}, "quic") {
		t.Fatal("expected quic tunnel forwarding to be supported")
	}
	if NodeSupportsDirectProtocol([]string{"tcp"}, "legacy") || NodeSupportsTunnelProtocol([]string{"tcp"}, "legacy") {
		t.Fatal("did not expect legacy protocol to be supported")
	}
	if !NodeSupportsTunnelProtocol(nil, "grpc") {
		t.Fatal("expected default protocol set to include grpc")
	}
}
