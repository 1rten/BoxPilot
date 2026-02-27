package generator

import (
	"encoding/json"

	"boxpilot/server/internal/util/errorx"
)

type ProxyInbound struct {
	Type          string
	ListenAddress string
	Port          int
	Enabled       bool
	AuthMode      string
	Username      string
	Password      string
}

type RoutingSettings struct {
	BypassPrivateEnabled bool
	BypassDomains        []string
	BypassCIDRs          []string
}

func DefaultRoutingSettings() RoutingSettings {
	return RoutingSettings{
		BypassPrivateEnabled: true,
		BypassDomains:        []string{"localhost", "local"},
		BypassCIDRs: []string{
			"127.0.0.0/8",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"169.254.0.0/16",
			"::1/128",
			"fc00::/7",
			"fe80::/10",
		},
	}
}

func BuildConfig(httpProxy ProxyInbound, socksProxy ProxyInbound, routing RoutingSettings, nodeOutboundJSONs []string) ([]byte, error) {
	inbounds := []map[string]any{}
	if httpProxy.Enabled {
		inbounds = append(inbounds, buildInbound("http", "http-in", httpProxy))
	}
	if socksProxy.Enabled {
		inbounds = append(inbounds, buildInbound("socks", "socks-in", socksProxy))
	}
	outbounds := []map[string]any{
		{"type": "direct", "tag": "direct"},
		{"type": "block", "tag": "block"},
	}
	var tags []string
	for _, raw := range nodeOutboundJSONs {
		var m map[string]any
		if json.Unmarshal([]byte(raw), &m) != nil {
			continue
		}
		outbounds = append(outbounds, m)
		if tag, ok := m["tag"].(string); ok {
			tags = append(tags, tag)
		}
	}
	switch len(tags) {
	case 0:
		outbounds = append(outbounds, map[string]any{
			"type":      "selector",
			"tag":       "proxy",
			"outbounds": []string{"direct"},
			"default":   "direct",
		})
	case 1:
		outbounds = append(outbounds, map[string]any{
			"type":      "selector",
			"tag":       "proxy",
			"outbounds": tags,
			"default":   tags[0],
		})
	default:
		outbounds = append(outbounds, map[string]any{
			"type":      "urltest",
			"tag":       "proxy-auto",
			"outbounds": tags,
			"url":       "https://www.gstatic.com/generate_204",
			"interval":  "3m",
			"tolerance": 120,
		})
		choices := make([]string, 0, len(tags)+1)
		choices = append(choices, "proxy-auto")
		choices = append(choices, tags...)
		outbounds = append(outbounds, map[string]any{
			"type":      "selector",
			"tag":       "proxy",
			"outbounds": choices,
			"default":   "proxy-auto",
		})
	}
	route := map[string]any{
		"final": "proxy",
	}
	if routing.BypassPrivateEnabled {
		rules := make([]map[string]any, 0, 2)
		if len(routing.BypassDomains) > 0 {
			rules = append(rules, map[string]any{
				"domain_suffix": routing.BypassDomains,
				"outbound":      "direct",
			})
		}
		if len(routing.BypassCIDRs) > 0 {
			rules = append(rules, map[string]any{
				"ip_cidr":  routing.BypassCIDRs,
				"outbound": "direct",
			})
		}
		if len(rules) > 0 {
			route["rules"] = rules
		}
	}

	cfg := map[string]any{
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route":     route,
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, errorx.New(errorx.CFGJSONInvalid, "marshal config")
	}
	return b, nil
}

func buildInbound(inType, tag string, p ProxyInbound) map[string]any {
	inb := map[string]any{
		"type":        inType,
		"tag":         tag,
		"listen":      p.ListenAddress,
		"listen_port": p.Port,
		"sniff":       true,
	}
	if p.AuthMode == "basic" && p.Username != "" && p.Password != "" {
		inb["users"] = []map[string]any{
			{"username": p.Username, "password": p.Password},
		}
	}
	return inb
}
