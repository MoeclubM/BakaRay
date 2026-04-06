package services

import "testing"

func TestNormalizeNodeProtocols(t *testing.T) {
	t.Run("filters unsupported and deduplicates", func(t *testing.T) {
		got := NormalizeNodeProtocols([]string{"GOST", "iptables", "echo", "gost", ""})
		if len(got) != 2 || got[0] != "gost" || got[1] != "iptables" {
			t.Fatalf("unexpected normalized protocols: %#v", got)
		}
	})

	t.Run("applies default linux capability set", func(t *testing.T) {
		got := NormalizeNodeProtocols(nil)
		if len(got) != 2 || got[0] != "gost" || got[1] != "iptables" {
			t.Fatalf("unexpected default protocols: %#v", got)
		}
	})
}

func TestNodeSupportsProtocol(t *testing.T) {
	if !NodeSupportsProtocol([]string{"gost"}, "gost") {
		t.Fatal("expected gost to be supported")
	}
	if NodeSupportsProtocol([]string{"gost"}, "iptables") {
		t.Fatal("did not expect iptables to be supported")
	}
	if !NodeSupportsProtocol(nil, "iptables") {
		t.Fatal("expected default protocol set to include iptables")
	}
}
