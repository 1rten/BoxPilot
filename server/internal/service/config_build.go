package service

import (
	"database/sql"
	"strings"

	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"
)

func BuildConfigFromDB(db *sql.DB, httpProxy, socksProxy generator.ProxyInbound, routing generator.RoutingSettings, forwardingRunning bool) ([]byte, []string, string, error) {
	if !forwardingRunning {
		httpProxy.Enabled = false
		socksProxy.Enabled = false
	}
	nodes, err := repo.ListEnabledForwardingNodes(db)
	if err != nil {
		return nil, nil, "", err
	}
	policy, err := LoadForwardingPolicy(db)
	if err != nil {
		return nil, nil, "", err
	}
	nodes = FilterForwardingNodes(nodes, policy)
	if forwardingRunning && (httpProxy.Enabled || socksProxy.Enabled) && len(nodes) == 0 {
		return nil, nil, "", errorx.New(errorx.CFGNoEnabledNodes, "no forwarding nodes enabled")
	}
	var outbounds []generator.NodeOutbound
	var tags []string
	for _, n := range nodes {
		outbounds = append(outbounds, generator.NodeOutbound{
			Tag:     n.Tag,
			RawJSON: n.OutboundJSON,
		})
		tags = append(tags, n.Tag)
	}
	cfg, err := generator.BuildConfigWithNodes(httpProxy, socksProxy, routing, outbounds)
	if err != nil {
		return nil, nil, "", err
	}
	hash := util.JSONHash(cfg)
	return cfg, tags, hash, nil
}

func FilterForwardingNodes(nodes []repo.NodeRow, policy ForwardingPolicy) []repo.NodeRow {
	if !policy.HealthyOnlyEnabled {
		return nodes
	}
	out := make([]repo.NodeRow, 0, len(nodes))
	for _, n := range nodes {
		status := ""
		if n.LastTestStatus.Valid {
			status = strings.ToLower(strings.TrimSpace(n.LastTestStatus.String))
		}
		if status == "ok" {
			if !n.LastLatencyMs.Valid {
				continue
			}
			if n.LastLatencyMs.Int64 > int64(policy.MaxLatencyMs) {
				continue
			}
			out = append(out, n)
			continue
		}
		if status == "" && policy.AllowUntested {
			out = append(out, n)
			continue
		}
	}
	return out
}
