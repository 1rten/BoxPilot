package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	inbounds := []map[string]any{
		{"type": "http", "tag": "http-in", "listen": "127.0.0.1", "listen_port": 7890},
	}
	outbounds := []any{
		map[string]any{"type": "direct", "tag": "direct"},
		map[string]any{"type": "block", "tag": "block"},
	}
	route := map[string]any{
		"final": "manual",
	}
	dns := map[string]any{
		"servers": []map[string]any{
			{"tag": "dns-direct", "address": "8.8.8.8", "detour": "direct"},
			{"tag": "dns-cloudflare", "address": "1.1.1.1", "detour": "direct"},
		},
		"final": "dns-direct",
	}

	cfg := map[string]any{
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route":     route,
		"dns":       dns,
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Println(string(b))
}
