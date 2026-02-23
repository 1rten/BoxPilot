package parser

import (
	"encoding/json"
	"boxpilot/server/internal/util/errorx"
)

type OutboundItem struct {
	Tag  string
	Type string
	Raw  json.RawMessage
}

var filterTypes = map[string]bool{
	"direct": true, "block": true, "dns": true, "selector": true, "urltest": true,
}

func ParseSubscription(body []byte) ([]OutboundItem, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err == nil && len(raw) > 0 {
		return parseArray(raw)
	}
	var obj struct {
		Outbounds []json.RawMessage `json:"outbounds"`
	}
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, errorx.New(errorx.SUBParseFailed, "invalid json")
	}
	return parseArray(obj.Outbounds)
}

func parseArray(arr []json.RawMessage) ([]OutboundItem, error) {
	var out []OutboundItem
	for _, b := range arr {
		var m map[string]interface{}
		if json.Unmarshal(b, &m) != nil {
			continue
		}
		t, _ := m["type"].(string)
		if filterTypes[t] {
			continue
		}
		tag, _ := m["tag"].(string)
		out = append(out, OutboundItem{Tag: tag, Type: t, Raw: b})
	}
	return out, nil
}
