package services

import (
	"strings"

	"bakaray/internal/models"
)

var defaultNodeProtocols = []string{
	"tcp",
	"udp",
	"tls",
	"mtls",
	"ws",
	"mws",
	"wss",
	"mwss",
	"grpc",
	"h2",
	"h2c",
	"kcp",
	"quic",
}

var supportedNodeProtocols = map[string]struct{}{
	"tcp":  {},
	"udp":  {},
	"tls":  {},
	"mtls": {},
	"ws":   {},
	"mws":  {},
	"wss":  {},
	"mwss": {},
	"grpc": {},
	"h2":   {},
	"h2c":  {},
	"kcp":  {},
	"quic": {},
}

func SupportedNodeProtocols() []string {
	out := make([]string, len(defaultNodeProtocols))
	copy(out, defaultNodeProtocols)
	return out
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
		return models.StringSlice(SupportedNodeProtocols())
	}

	return models.StringSlice(out)
}

func NormalizeDirectProtocol(protocol string) string {
	return strings.ToLower(strings.TrimSpace(protocol))
}

func NormalizeTunnelProtocol(protocol string) string {
	return strings.ToLower(strings.TrimSpace(protocol))
}

func IsDirectProtocol(protocol string) bool {
	switch NormalizeDirectProtocol(protocol) {
	case "tcp", "udp":
		return true
	default:
		return false
	}
}

func IsTunnelProtocol(protocol string) bool {
	_, ok := supportedNodeProtocols[NormalizeTunnelProtocol(protocol)]
	return ok && !IsDirectProtocol(protocol)
}

func DirectProtocolNetwork(protocol string) string {
	if NormalizeDirectProtocol(protocol) == "udp" {
		return "udp"
	}
	return "tcp"
}

func TunnelProtocolNetwork(protocol string) string {
	switch NormalizeTunnelProtocol(protocol) {
	case "kcp", "quic":
		return "udp"
	default:
		return "tcp"
	}
}

func NodeSupportsDirectProtocol(protocols []string, protocol string) bool {
	protocol = NormalizeDirectProtocol(protocol)
	if !IsDirectProtocol(protocol) {
		return false
	}
	for _, item := range NormalizeNodeProtocols(protocols) {
		if item == protocol {
			return true
		}
	}
	return false
}

func NodeSupportsTunnelProtocol(protocols []string, protocol string) bool {
	protocol = NormalizeTunnelProtocol(protocol)
	if !IsTunnelProtocol(protocol) {
		return false
	}
	for _, item := range NormalizeNodeProtocols(protocols) {
		if item == protocol {
			return true
		}
	}
	return false
}
