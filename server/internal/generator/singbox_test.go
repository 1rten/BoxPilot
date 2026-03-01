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
	if got := route["final"]; got != "proxy" {
		t.Fatalf("expected route.final=proxy, got %v", got)
	}

	rules, ok := route["rules"].([]any)
	if !ok || len(rules) != 2 {
		t.Fatalf("expected 2 route rules, got %v", route["rules"])
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
	if len(outbounds) < 5 {
		t.Fatalf("expected at least 5 outbounds, got %d", len(outbounds))
	}
}
