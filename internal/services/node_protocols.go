package services

import (
	"strings"

	"bakaray/internal/models"
)

var supportedNodeProtocols = map[string]struct{}{
	"gost":     {},
	"iptables": {},
}

// NormalizeNodeProtocols keeps only supported node capabilities and applies the
// default Linux capability set when the input is empty.
func NormalizeNodeProtocols(protocols []string) models.StringSlice {
	seen := make(map[string]struct{}, len(protocols))
	out := make([]string, 0, len(protocols))

	for _, protocol := range protocols {
		protocol = strings.ToLower(strings.TrimSpace(protocol))
		if protocol == "" {
			continue
		}
		if _, ok := supportedNodeProtocols[protocol]; !ok {
			continue
		}
		if _, ok := seen[protocol]; ok {
			continue
		}
		seen[protocol] = struct{}{}
		out = append(out, protocol)
	}

	if len(out) == 0 {
		return models.StringSlice{"gost", "iptables"}
	}

	return models.StringSlice(out)
}

func NodeSupportsProtocol(protocols []string, protocol string) bool {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	for _, item := range NormalizeNodeProtocols(protocols) {
		if item == protocol {
			return true
		}
	}
	return false
}
