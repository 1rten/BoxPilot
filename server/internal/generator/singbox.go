package generator

import (
	"encoding/json"
	"boxpilot/server/internal/util/errorx"
)

func BuildConfig(httpPort, socksPort int, nodeOutboundJSONs []string) ([]byte, error) {
	inbounds := []map[string]any{
		{"type": "http", "tag": "http-in", "listen": "0.0.0.0", "listen_port": httpPort, "sniff": true},
		{"type": "socks", "tag": "socks-in", "listen": "0.0.0.0", "listen_port": socksPort, "sniff": true},
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
