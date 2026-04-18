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
	ruleSetRows, err := repo.ListEnabledSubscriptionRuleSets(db)
	if err != nil {
		return nil, nil, "", err
	}
	ruleRows, err := repo.ListEnabledSubscriptionRules(db)
	if err != nil {
		return nil, nil, "", err
	}
	groupMemberRows, err := repo.ListEnabledSubscriptionGroupMembers(db)
	if err != nil {
		return nil, nil, "", err
	}
	extras := generator.RoutingExtras{
		RuleSets:          make([]generator.RouteRuleSetRef, 0, len(ruleSetRows)),
		Rules:             make([]generator.RouteRule, 0, len(ruleRows)),
		GroupSelections:   map[string]string{},
		BusinessNodePools: map[string][]string{},
		AutoTestURL:       generator.DefaultAutoTestURL,
		AutoTestInterval:  BizAutoIntervalDuration(policy.BizAutoIntervalSec),
	}
	for _, rs := range ruleSetRows {
		extras.RuleSets = append(extras.RuleSets, generator.RouteRuleSetRef{
			Tag:        rs.Tag,
			SourceType: rs.SourceType,
			Format:     rs.Format,
			URL:        rs.URL,
			Path:       rs.Path,
		})
	}
	for _, r := range ruleRows {
		extras.Rules = append(extras.Rules, generator.RouteRule{
			Priority:       r.Priority,
			RuleOrder:      r.RuleOrder,
			MatcherType:    r.MatcherType,
			MatcherValue:   r.MatcherValue,
			TargetOutbound: r.TargetOutbound,
		})
	}
	for _, g := range groupMemberRows {
		target := strings.TrimSpace(g.TargetOutbound)
		tag := strings.TrimSpace(g.NodeTag)
		if target == "" || tag == "" {
			continue
		}
		extras.BusinessNodePools[target] = append(extras.BusinessNodePools[target], tag)
	}
	selectionRows, err := repo.ListRuntimeGroupSelections(db)
	if err != nil {
		return nil, nil, "", err
	}
	for _, s := range selectionRows {
		extras.GroupSelections[s.GroupTag] = s.SelectedOutbound
	}
	cfg, err := generator.BuildConfigWithRuntime(httpProxy, socksProxy, routing, outbounds, extras)
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
		// User-added manual nodes start with no probe result; default policy would
		// drop them and break forwarding until settings change or a test is run.
		if n.SubID == repo.ManualSubscriptionID && status == "" {
			out = append(out, n)
			continue
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
