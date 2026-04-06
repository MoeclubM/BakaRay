package handlers

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"bakaray/internal/logger"
	"bakaray/internal/models"
	"bakaray/internal/services"
)

func triggerNodeReloadAsync(nodeService *services.NodeService, requestID string, nodeIDs ...uint) {
	seen := make(map[uint]struct{}, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		if nodeID == 0 {
			continue
		}
		if _, ok := seen[nodeID]; ok {
			continue
		}
		seen[nodeID] = struct{}{}

		go func(id uint) {
			node, err := nodeService.GetNodeByID(id)
			if err != nil {
				logger.Warn("AsyncReloadNode: node not found", "node_id", id, "request_id", requestID)
				return
			}
			if err := reloadNode(node); err != nil {
				logger.Warn("AsyncReloadNode: reload failed", "node_id", id, "request_id", requestID, "error", err)
				return
			}
			logger.Info("AsyncReloadNode: reload triggered", "node_id", id, "request_id", requestID)
		}(nodeID)
	}
}

func reloadNode(node *models.Node) error {
	reloadURL := buildNodeURL(node.Host, node.Port) + "/reload"
	req, err := http.NewRequest(http.MethodPost, reloadURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Node-Secret", node.Secret)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func buildNodeURL(host string, port int) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	if strings.Contains(host, "://") {
		if u, err := url.Parse(host); err == nil {
			if u.Scheme == "" {
				u.Scheme = "http"
			}
			if u.Host == "" {
				u.Host = host
			}

			if u.Port() == "" && port > 0 {
				u.Host = net.JoinHostPort(u.Hostname(), strconv.Itoa(port))
			}

			return strings.TrimRight(u.String(), "/")
		}
	}

	if ip := net.ParseIP(host); ip != nil && port > 0 {
		return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
	}

	if _, _, err := net.SplitHostPort(host); err == nil {
		return "http://" + strings.TrimRight(host, "/")
	}

	if port > 0 {
		return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
	}

	return "http://" + strings.TrimRight(host, "/")
}
