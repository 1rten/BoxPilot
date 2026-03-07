package generator

import (
	"encoding/json"
	"testing"
)

func TestBuildConfig_WithBypassRules(t *testing.T) {
	cfg, err := BuildConfig(
		ProxyInbound{Type: "http", ListenAddress: "0.0.0.0", Port: 7890, Enabled: true},
		ProxyInbound{Type: "socks", ListenAddress: "0.0.0.0", Port: 7891, Enabled: true},
		RoutingSettings{
			BypassPrivateEnabled: true,
			BypassDomains:        []string{"localhost", "local"},
			BypassCIDRs:          []string{"10.0.0.0/8", "192.168.0.0/16"},
		},
		[]string{`{"type":"trojan","tag":"node-a","server":"example.com","server_port":443,"password":"p"}`},
	)
	if err != nil {
		t.Fatalf("BuildConfig returned error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	route, ok := parsed["route"].(map[string]any)
	if !ok {
		t.Fatalf("missing route section")
	}
	if got := route["final"]; got != "manual" {
		t.Fatalf("expected route.final=manual, got %v", got)
	}

	rules, ok := route["rules"].([]any)
	if !ok || len(rules) != 3 {
		t.Fatalf("expected 3 route rules, got %v", route["rules"])
	}

	domainRule, ok := rules[0].(map[string]any)
	if !ok {
		t.Fatalf("invalid domain rule type")
	}
	if got := domainRule["outbound"]; got != "direct" {
		t.Fatalf("expected domain rule outbound direct, got %v", got)
	}
	domainSuffixes, ok := domainRule["domain_suffix"].([]any)
	if !ok || len(domainSuffixes) != 2 {
		t.Fatalf("expected 2 domain_suffix values, got %v", domainRule["domain_suffix"])
	}

	cidrRule, ok := rules[1].(map[string]any)
	if !ok {
		t.Fatalf("invalid cidr rule type")
	}
	if got := cidrRule["outbound"]; got != "direct" {
		t.Fatalf("expected cidr rule outbound direct, got %v", got)
	}
	ipCIDRs, ok := cidrRule["ip_cidr"].([]any)
	if !ok || len(ipCIDRs) != 2 {
		t.Fatalf("expected 2 ip_cidr values, got %v", cidrRule["ip_cidr"])
	}

	cnRule, ok := rules[2].(map[string]any)
	if !ok {
		t.Fatalf("invalid cn rule type")
	}
	if got := cnRule["outbound"]; got != "direct" {
		t.Fatalf("expected cn rule outbound direct, got %v", got)
	}
	if _, ok := cnRule["rule_set"].([]any); !ok {
		t.Fatalf("expected rule_set list in cn rule")
	}

	ruleSets, ok := route["rule_set"].([]any)
	if !ok || len(ruleSets) != 2 {
		t.Fatalf("expected 2 route.rule_set entries, got %v", route["rule_set"])
	}
}

func TestBuildConfig_WithoutBypassRules(t *testing.T) {
	cfg, err := BuildConfig(
		ProxyInbound{Type: "http", ListenAddress: "0.0.0.0", Port: 7890, Enabled: true},
		ProxyInbound{Type: "socks", ListenAddress: "0.0.0.0", Port: 7891, Enabled: true},
		RoutingSettings{
			BypassPrivateEnabled: false,
			BypassDomains:        []string{"localhost"},
			BypassCIDRs:          []string{"10.0.0.0/8"},
		},
		[]string{`{"type":"trojan","tag":"node-a","server":"example.com","server_port":443,"password":"p"}`},
	)
	if err != nil {
		t.Fatalf("BuildConfig returned error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	route, ok := parsed["route"].(map[string]any)
	if !ok {
		t.Fatalf("missing route section")
	}
	if _, exists := route["rules"]; exists {
		t.Fatalf("did not expect route.rules when bypass is disabled")
	}
}

func TestBuildConfigWithNodes_UsesProvidedTags(t *testing.T) {
	cfg, err := BuildConfigWithNodes(
		ProxyInbound{Type: "http", ListenAddress: "0.0.0.0", Port: 7890, Enabled: true},
		ProxyInbound{Type: "socks", ListenAddress: "0.0.0.0", Port: 7891, Enabled: true},
		RoutingSettings{BypassPrivateEnabled: false},
		[]NodeOutbound{
			{
				Tag:     "node-fast",
				RawJSON: `{"type":"trojan","tag":"node-fast","server":"example.com","server_port":443,"password":"p"}`,
			},
			{
				Tag:     "node-raw-only",
				RawJSON: `{"type":"vmess","tag":"node-raw-only","server":"example.org","server_port":443,"uuid":"11111111-1111-1111-1111-111111111111"}`,
			},
		},
	)
	if err != nil {
		t.Fatalf("BuildConfigWithNodes returned error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	outbounds, ok := parsed["outbounds"].([]any)
	if !ok {
		t.Fatalf("missing outbounds")
	}
	if len(outbounds) < 4 {
		t.Fatalf("expected at least 4 outbounds, got %d", len(outbounds))
	}
	for _, item := range outbounds {
		outbound, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if typ, _ := outbound["type"].(string); typ == "urltest" {
			t.Fatalf("did not expect global urltest outbound, got %v", outbound["tag"])
		}
	}
}

func TestBuildConfigWithRuntime_BusinessGroups(t *testing.T) {
	cfg, err := BuildConfigWithRuntime(
		ProxyInbound{Type: "http", ListenAddress: "0.0.0.0", Port: 7890, Enabled: true},
		ProxyInbound{Type: "socks", ListenAddress: "0.0.0.0", Port: 7891, Enabled: true},
		RoutingSettings{BypassPrivateEnabled: true},
		[]NodeOutbound{
			{
				Tag:     "node-a",
				RawJSON: `{"type":"trojan","tag":"node-a","server":"example.com","server_port":443,"password":"p"}`,
			},
			{
				Tag:     "node-b",
				RawJSON: `{"type":"trojan","tag":"node-b","server":"example.org","server_port":443,"password":"p"}`,
			},
		},
		RoutingExtras{
			RuleSets: []RouteRuleSetRef{
				{
					Tag:        "geosite-openai",
					SourceType: "remote",
					Format:     "binary",
					URL:        "https://example.com/openai.srs",
				},
			},
			Rules: []RouteRule{
				{
					Priority:       200,
					RuleOrder:      1,
					MatcherType:    "rule_set",
					MatcherValue:   "geosite-openai",
					TargetOutbound: "OpenAI",
				},
			},
			GroupSelections: map[string]string{
				"manual":     "node-b",
				"biz-openai": "biz-openai-auto",
			},
			BusinessNodePools: map[string][]string{
				"OpenAI": {"node-a", "node-b"},
			},
		},
	)
	if err != nil {
		t.Fatalf("BuildConfigWithRuntime returned error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	outbounds, ok := parsed["outbounds"].([]any)
	if !ok {
		t.Fatalf("missing outbounds")
	}
	hasManual := false
	hasBiz := false
	hasBizAuto := false
	for _, item := range outbounds {
		outbound, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tag, _ := outbound["tag"].(string)
		typ, _ := outbound["type"].(string)
		switch {
		case tag == "manual" && typ == "selector":
			hasManual = true
			if d, _ := outbound["default"].(string); d != "node-b" {
				t.Fatalf("expected manual selector default node-b, got %v", outbound["default"])
			}
		case tag == "biz-openai" && typ == "selector":
			hasBiz = true
			if d, _ := outbound["default"].(string); d != "biz-openai-auto" {
				t.Fatalf("expected biz selector default biz-openai-auto, got %v", outbound["default"])
			}
		case typ == "urltest" && tag == "biz-openai-auto":
			hasBizAuto = true
			if interval, _ := outbound["interval"].(string); interval != "30m" {
				t.Fatalf("expected business urltest interval 30m, got %v", outbound["interval"])
			}
			members, _ := outbound["outbounds"].([]any)
			if len(members) != 2 {
				t.Fatalf("expected business urltest members size 2, got %v", outbound["outbounds"])
			}
		}
	}
	if !hasManual || !hasBiz || !hasBizAuto {
		t.Fatalf("expected manual+biz+biz-auto outbounds, got %#v", outbounds)
	}
}

func TestBuildConfigWithRuntime_BusinessGroupsWithoutPool(t *testing.T) {
	cfg, err := BuildConfigWithRuntime(
		ProxyInbound{Type: "http", ListenAddress: "0.0.0.0", Port: 7890, Enabled: true},
		ProxyInbound{Type: "socks", ListenAddress: "0.0.0.0", Port: 7891, Enabled: true},
		RoutingSettings{BypassPrivateEnabled: true},
		[]NodeOutbound{
			{
				Tag:     "node-a",
				RawJSON: `{"type":"trojan","tag":"node-a","server":"example.com","server_port":443,"password":"p"}`,
			},
		},
		RoutingExtras{
			Rules: []RouteRule{
				{
					Priority:       200,
					RuleOrder:      1,
					MatcherType:    "domain_suffix",
					MatcherValue:   "apple.com",
					TargetOutbound: "Apple",
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("BuildConfigWithRuntime returned error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	outbounds, ok := parsed["outbounds"].([]any)
	if !ok {
		t.Fatalf("missing outbounds")
	}
	hasBizSelector := false
	hasBizAuto := false
	for _, item := range outbounds {
		outbound, ok := item.(map[string]any)
		if !ok {
			continue
		}
		tag, _ := outbound["tag"].(string)
		typ, _ := outbound["type"].(string)
		if tag == "biz-apple" && typ == "selector" {
			hasBizSelector = true
			members, _ := outbound["outbounds"].([]any)
			if len(members) != 1 || members[0] != "manual" {
				t.Fatalf("expected biz selector only manual when pool missing, got %v", outbound["outbounds"])
			}
		}
		if tag == "biz-apple-auto" {
			hasBizAuto = true
		}
	}
	if !hasBizSelector {
		t.Fatalf("expected biz selector created")
	}
	if hasBizAuto {
		t.Fatalf("did not expect biz auto when pool missing")
	}
}
