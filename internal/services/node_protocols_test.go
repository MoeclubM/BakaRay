package services

import "testing"

func TestNormalizeNodeProtocols(t *testing.T) {
	t.Run("filters unsupported and deduplicates", func(t *testing.T) {
		got := NormalizeNodeProtocols([]string{"GOST", "legacy", "echo", "gost", ""})
		if len(got) != 1 || got[0] != "gost" {
			t.Fatalf("unexpected normalized protocols: %#v", got)
		}
	})

	t.Run("applies default linux capability set", func(t *testing.T) {
		got := NormalizeNodeProtocols(nil)
		if len(got) != 1 || got[0] != "gost" {
			t.Fatalf("unexpected default protocols: %#v", got)
		}
	})
}

func TestNodeSupportsProtocol(t *testing.T) {
	if !NodeSupportsProtocol([]string{"gost"}, "gost") {
		t.Fatal("expected gost to be supported")
	}
	if NodeSupportsProtocol([]string{"gost"}, "legacy") {
		t.Fatal("did not expect legacy protocol to be supported")
	}
	if NodeSupportsProtocol(nil, "legacy") {
		t.Fatal("did not expect default protocol set to include legacy protocol")
	}
}
