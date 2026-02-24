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

func BuildConfig(httpProxy ProxyInbound, socksProxy ProxyInbound, nodeOutboundJSONs []string) ([]byte, error) {
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
	outbounds = append(outbounds, map[string]any{"type": "selector", "tag": "proxy", "outbounds": tags})
	cfg := map[string]any{
		"inbounds": inbounds, "outbounds": outbounds,
		"route": map[string]any{"final": "proxy"},
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
