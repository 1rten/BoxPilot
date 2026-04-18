package parser

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"boxpilot/server/internal/util/errorx"
)

func TestParseSubscription_TraditionalURIList(t *testing.T) {
	vmessJSON := `{"v":"2","ps":"vmess-node","add":"example.com","port":"443","id":"11111111-1111-1111-1111-111111111111","aid":"0","net":"ws","host":"ws.example.com","path":"/ws","tls":"tls"}`
	vmessURI := "vmess://" + base64.StdEncoding.EncodeToString([]byte(vmessJSON))
	payload := vmessURI + "\n" + "trojan://password@example.org:443#trojan-node"

	out, err := ParseSubscription([]byte(payload))
	if err != nil {
		t.Fatalf("ParseSubscription returned error: %v", err)
	}
	if len(out) < 2 {
		t.Fatalf("expected at least 2 outbounds, got %d", len(out))
	}
	assertHasType(t, out, "vmess")
	assertHasType(t, out, "trojan")
}

func TestParseSubscription_VLESSRealityURI(t *testing.T) {
	payload := "vless://23b2a4d9-f79b-4dab-95ba-a830bbf3319e@209.141.45.135:21883?type=tcp&encryption=none&security=reality&pbk=xcyhfdC14W9c5hlqMx0rAkhpQBBEBTtuVi1dLLOcMjs&fp=chrome&sni=www.microsoft.com&sid=5a9f0379&spx=%2F#BuyVM-LV-Reality"
	out, err := ParseSubscription([]byte(payload))
	if err != nil {
		t.Fatalf("ParseSubscription returned error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected one outbound, got %d", len(out))
	}
	var m map[string]any
	if err := json.Unmarshal(out[0].Raw, &m); err != nil {
		t.Fatalf("unmarshal outbound failed: %v", err)
	}
	tls, ok := m["tls"].(map[string]any)
	if !ok {
		t.Fatalf("expected tls config in outbound: %+v", m)
	}
	if enabled, _ := tls["enabled"].(bool); !enabled {
		t.Fatalf("expected tls.enabled=true, got %+v", tls["enabled"])
	}
	reality, ok := tls["reality"].(map[string]any)
	if !ok {
		t.Fatalf("expected tls.reality in outbound: %+v", tls)
	}
	if got := reality["public_key"]; got != "xcyhfdC14W9c5hlqMx0rAkhpQBBEBTtuVi1dLLOcMjs" {
		t.Fatalf("unexpected reality public key: %v", got)
	}
	if got := reality["short_id"]; got != "5a9f0379" {
		t.Fatalf("unexpected reality short_id: %v", got)
	}
	if got := reality["spider_x"]; got != "/" {
		t.Fatalf("expected reality.spider_x to be '/', got %v", got)
	}
}

func TestParseSubscription_SingboxPlainAndBase64(t *testing.T) {
	singbox := `{
	  "outbounds": [
	    {"type":"direct","tag":"direct"},
	    {"type":"vmess","tag":"vmess-a","server":"example.com","server_port":443,"uuid":"11111111-1111-1111-1111-111111111111"}
	  ]
	}`

	plain, err := ParseSubscription([]byte(singbox))
	if err != nil {
		t.Fatalf("parse singbox plain error: %v", err)
	}
	if len(plain) != 1 || plain[0].Type != "vmess" {
		t.Fatalf("expected only vmess outbound, got %+v", plain)
	}

	base64Payload := base64.StdEncoding.EncodeToString([]byte(singbox))
	decoded, err := ParseSubscription([]byte(base64Payload))
	if err != nil {
		t.Fatalf("parse singbox base64 error: %v", err)
	}
	if len(decoded) != 1 || decoded[0].Type != "vmess" {
		t.Fatalf("expected only vmess outbound from base64 payload, got %+v", decoded)
	}
}

func TestParseSubscription_ClashPlainAndBase64(t *testing.T) {
	clashYAML := `
proxies:
  - name: vmess-hk
    type: vmess
    server: hk.example.com
    port: 443
    uuid: 11111111-1111-1111-1111-111111111111
    alterId: 0
    cipher: auto
  - name: http-proxy
    type: http
    server: proxy.example.com
    port: 8080
`
	plain, err := ParseSubscription([]byte(clashYAML))
	if err != nil {
		t.Fatalf("parse clash plain error: %v", err)
	}
	if len(plain) < 2 {
		t.Fatalf("expected at least 2 outbounds for clash plain, got %d", len(plain))
	}
	assertHasType(t, plain, "vmess")
	assertHasType(t, plain, "http")

	base64Payload := base64.StdEncoding.EncodeToString([]byte(clashYAML))
	decoded, err := ParseSubscription([]byte(base64Payload))
	if err != nil {
		t.Fatalf("parse clash base64 error: %v", err)
	}
	if len(decoded) < 2 {
		t.Fatalf("expected at least 2 outbounds for clash base64, got %d", len(decoded))
	}
	assertHasType(t, decoded, "vmess")
	assertHasType(t, decoded, "http")
}

func TestParseSubscription_EmptyAndUnsupported(t *testing.T) {
	_, err := ParseSubscription([]byte("   \n"))
	if err == nil {
		t.Fatalf("expected error for empty payload")
	}
	assertAppErrorCode(t, err, errorx.SUBParseFailed)

	_, err = ParseSubscription([]byte("this is definitely not a subscription payload"))
	if err == nil {
		t.Fatalf("expected unsupported format error")
	}
	assertAppErrorCode(t, err, errorx.SUBFormatUnsupported)
}

func TestParseSubscription_TraditionalBase64WholePayload(t *testing.T) {
	vmessJSON := map[string]any{
		"v":    "2",
		"ps":   "vmess-b64",
		"add":  "example.net",
		"port": "443",
		"id":   "11111111-1111-1111-1111-111111111111",
		"aid":  "0",
	}
	vmessRaw, _ := json.Marshal(vmessJSON)
	vmessURI := "vmess://" + base64.StdEncoding.EncodeToString(vmessRaw)
	whole := base64.StdEncoding.EncodeToString([]byte(vmessURI + "\n"))

	out, err := ParseSubscription([]byte(whole))
	if err != nil {
		t.Fatalf("expected traditional base64 payload to parse, got error: %v", err)
	}
	if len(out) != 1 || out[0].Type != "vmess" {
		t.Fatalf("expected single vmess outbound, got %+v", out)
	}
}

func TestParseSubscriptionBundle_SingboxBusinessRules(t *testing.T) {
	payload := `{
	  "outbounds": [
	    {"type":"vmess","tag":"node-a","server":"example.com","server_port":443,"uuid":"11111111-1111-1111-1111-111111111111"}
	  ],
	  "route": {
	    "rule_set": [
	      {"tag":"geosite-openai","type":"remote","format":"binary","url":"https://example.com/openai.srs"},
	      {"tag":"geosite-cn","type":"remote","format":"binary","url":"https://example.com/cn.srs"}
	    ],
	    "rules": [
	      {"rule_set":"geosite-openai","outbound":"OpenAI"},
	      {"rule_set":"geosite-cn","outbound":"direct"}
	    ]
	  }
	}`

	parsed, err := ParseSubscriptionBundle([]byte(payload))
	if err != nil {
		t.Fatalf("ParseSubscriptionBundle returned error: %v", err)
	}
	if len(parsed.Outbounds) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(parsed.Outbounds))
	}
	if len(parsed.Rules) != 1 {
		t.Fatalf("expected 1 business rule, got %d", len(parsed.Rules))
	}
	if parsed.Rules[0].MatcherType != "rule_set" || parsed.Rules[0].MatcherValue != "geosite-openai" {
		t.Fatalf("unexpected parsed rule: %+v", parsed.Rules[0])
	}
	if len(parsed.RuleSets) != 1 || parsed.RuleSets[0].Tag != "geosite-openai" {
		t.Fatalf("expected only openai rule_set retained, got %+v", parsed.RuleSets)
	}
}

func TestParseSubscriptionBundle_ClashBusinessRules(t *testing.T) {
	payload := `
proxies:
  - name: vmess-hk
    type: vmess
    server: hk.example.com
    port: 443
    uuid: 11111111-1111-1111-1111-111111111111
  - name: vmess-us
    type: vmess
    server: us.example.com
    port: 443
    uuid: 22222222-2222-2222-2222-222222222222
proxy-groups:
  - name: openAI
    type: select
    proxies:
      - vmess-hk
      - vmess-us
rules:
  - DOMAIN-SUFFIX,openai.com,openAI
  - GEOIP,CN,DIRECT
  - RULE-SET,openai_rule,openAI
rule-providers:
  openai_rule:
    type: http
    behavior: domain
    url: https://example.com/openai.yaml
`
	parsed, err := ParseSubscriptionBundle([]byte(payload))
	if err != nil {
		t.Fatalf("ParseSubscriptionBundle returned error: %v", err)
	}
	if len(parsed.Rules) != 2 {
		t.Fatalf("expected 2 business rules, got %d", len(parsed.Rules))
	}
	if len(parsed.RuleSets) != 1 || parsed.RuleSets[0].Tag != "openai_rule" {
		t.Fatalf("expected one rule_set provider, got %+v", parsed.RuleSets)
	}
	if len(parsed.BusinessGroups) != 1 {
		t.Fatalf("expected one business group mapping, got %+v", parsed.BusinessGroups)
	}
	if parsed.BusinessGroups[0].TargetOutbound != "openAI" {
		t.Fatalf("unexpected business target: %+v", parsed.BusinessGroups[0])
	}
	if len(parsed.BusinessGroups[0].NodeTags) != 2 {
		t.Fatalf("expected two business members, got %+v", parsed.BusinessGroups[0].NodeTags)
	}
}

func TestParseSubscriptionBundle_SingboxBusinessGroupMembers(t *testing.T) {
	payload := `{
	  "outbounds": [
	    {"type":"vmess","tag":"node-a","server":"example.com","server_port":443,"uuid":"11111111-1111-1111-1111-111111111111"},
	    {"type":"trojan","tag":"node-b","server":"example.org","server_port":443,"password":"p"},
	    {"type":"selector","tag":"Apple","outbounds":["node-a","node-b"]}
	  ],
	  "route": {
	    "rules": [
	      {"domain_suffix":["apple.com"],"outbound":"Apple"}
	    ]
	  }
	}`

	parsed, err := ParseSubscriptionBundle([]byte(payload))
	if err != nil {
		t.Fatalf("ParseSubscriptionBundle returned error: %v", err)
	}
	if len(parsed.BusinessGroups) != 1 {
		t.Fatalf("expected one business group mapping, got %+v", parsed.BusinessGroups)
	}
	group := parsed.BusinessGroups[0]
	if group.TargetOutbound != "Apple" {
		t.Fatalf("unexpected target outbound: %+v", group)
	}
	if len(group.NodeTags) != 2 || group.NodeTags[0] != "node-a" || group.NodeTags[1] != "node-b" {
		t.Fatalf("unexpected singbox business node tags: %+v", group.NodeTags)
	}
}

func TestParseSubscriptionBundle_SingboxBusinessGroupPreferExplicitMembers(t *testing.T) {
	payload := `{
	  "outbounds": [
	    {"type":"vmess","tag":"node-1","server":"a.example.com","server_port":443,"uuid":"11111111-1111-1111-1111-111111111111"},
	    {"type":"vmess","tag":"node-2","server":"b.example.com","server_port":443,"uuid":"22222222-2222-2222-2222-222222222222"},
	    {"type":"vmess","tag":"node-3","server":"c.example.com","server_port":443,"uuid":"33333333-3333-3333-3333-333333333333"},
	    {"type":"selector","tag":"Auto_Selector_Proxy","outbounds":["node-1","node-2","node-3"]},
	    {"type":"selector","tag":"手动切换","outbounds":["Auto_Selector_Proxy","node-1","node-2","node-3"]},
	    {"type":"selector","tag":"OpenAI","outbounds":["Auto_Selector_Proxy","手动切换","node-1","node-2"]}
	  ],
	  "route": {
	    "rules": [
	      {"domain_suffix":["openai.com"],"outbound":"OpenAI"}
	    ]
	  }
	}`

	parsed, err := ParseSubscriptionBundle([]byte(payload))
	if err != nil {
		t.Fatalf("ParseSubscriptionBundle returned error: %v", err)
	}
	if len(parsed.BusinessGroups) != 1 {
		t.Fatalf("expected one business group mapping, got %+v", parsed.BusinessGroups)
	}
	group := parsed.BusinessGroups[0]
	if group.TargetOutbound != "OpenAI" {
		t.Fatalf("unexpected target outbound: %+v", group)
	}
	if len(group.NodeTags) != 2 || group.NodeTags[0] != "node-1" || group.NodeTags[1] != "node-2" {
		t.Fatalf("expected only explicit OpenAI nodes, got %+v", group.NodeTags)
	}
}

func TestParseSubscriptionBundle_ClashBusinessGroupPreferExplicitMembers(t *testing.T) {
	payload := `
proxies:
  - name: node-1
    type: vmess
    server: a.example.com
    port: 443
    uuid: 11111111-1111-1111-1111-111111111111
  - name: node-2
    type: vmess
    server: b.example.com
    port: 443
    uuid: 22222222-2222-2222-2222-222222222222
  - name: node-3
    type: vmess
    server: c.example.com
    port: 443
    uuid: 33333333-3333-3333-3333-333333333333
proxy-groups:
  - name: Proxy
    type: select
    proxies: [node-1, node-2, node-3]
  - name: 手动切换
    type: select
    proxies: [Proxy, node-1, node-2, node-3]
  - name: OpenAI
    type: select
    proxies: [Proxy, 手动切换, node-1, node-2]
rules:
  - DOMAIN-SUFFIX,openai.com,OpenAI
`
	parsed, err := ParseSubscriptionBundle([]byte(payload))
	if err != nil {
		t.Fatalf("ParseSubscriptionBundle returned error: %v", err)
	}
	if len(parsed.BusinessGroups) != 1 {
		t.Fatalf("expected one business group mapping, got %+v", parsed.BusinessGroups)
	}
	group := parsed.BusinessGroups[0]
	if group.TargetOutbound != "OpenAI" {
		t.Fatalf("unexpected clash target outbound: %+v", group)
	}
	if len(group.NodeTags) != 2 || group.NodeTags[0] != "node-1" || group.NodeTags[1] != "node-2" {
		t.Fatalf("expected only explicit OpenAI nodes, got %+v", group.NodeTags)
	}
}

func TestParseSubscriptionBundle_ClashBusinessGroupPreferExplicitMembersKeepDirectFromHelper(t *testing.T) {
	payload := `
proxies:
  - name: node-1
    type: vmess
    server: a.example.com
    port: 443
    uuid: 11111111-1111-1111-1111-111111111111
proxy-groups:
  - name: Proxy
    type: select
    proxies: [node-1]
  - name: 本地直连
    type: select
    proxies: [DIRECT]
  - name: OpenAI
    type: select
    proxies: [Proxy, 本地直连, node-1]
rules:
  - DOMAIN-SUFFIX,openai.com,OpenAI
`
	parsed, err := ParseSubscriptionBundle([]byte(payload))
	if err != nil {
		t.Fatalf("ParseSubscriptionBundle returned error: %v", err)
	}
	if len(parsed.BusinessGroups) != 1 {
		t.Fatalf("expected one business group mapping, got %+v", parsed.BusinessGroups)
	}
	group := parsed.BusinessGroups[0]
	if group.TargetOutbound != "OpenAI" {
		t.Fatalf("unexpected clash target outbound: %+v", group)
	}
	if len(group.NodeTags) != 2 || group.NodeTags[0] != "node-1" || group.NodeTags[1] != "direct" {
		t.Fatalf("expected explicit node + direct from helper group, got %+v", group.NodeTags)
	}
}

func TestParseSubscriptionBundle_SingboxIgnoreAutoSelectorAsBusinessTarget(t *testing.T) {
	payload := `{
	  "outbounds": [
	    {"type":"vmess","tag":"node-a","server":"example.com","server_port":443,"uuid":"11111111-1111-1111-1111-111111111111"},
	    {"type":"urltest","tag":"Auto_Selector_Proxy","outbounds":["node-a"]}
	  ],
	  "route": {
	    "rules": [
	      {"domain_suffix":["example.com"],"outbound":"Auto_Selector_Proxy"},
	      {"domain_suffix":["openai.com"],"outbound":"OpenAI"}
	    ]
	  }
	}`
	parsed, err := ParseSubscriptionBundle([]byte(payload))
	if err != nil {
		t.Fatalf("ParseSubscriptionBundle returned error: %v", err)
	}
	if len(parsed.BusinessGroups) != 0 {
		t.Fatalf("expected no business groups when target only points to helper/unknown groups, got %+v", parsed.BusinessGroups)
	}
	if len(parsed.Rules) != 1 || parsed.Rules[0].TargetOutbound != "OpenAI" {
		t.Fatalf("expected only OpenAI business rule retained, got %+v", parsed.Rules)
	}
}

func TestParseSubscriptionBundle_ClashIgnoreAutoSelectorAsBusinessTarget(t *testing.T) {
	payload := `
proxies:
  - name: node-1
    type: vmess
    server: a.example.com
    port: 443
    uuid: 11111111-1111-1111-1111-111111111111
proxy-groups:
  - name: 自动选择 - HK
    type: url-test
    proxies: [node-1]
  - name: OpenAI
    type: select
    proxies: [node-1]
rules:
  - DOMAIN-SUFFIX,example.com,自动选择 - HK
  - DOMAIN-SUFFIX,openai.com,OpenAI
`
	parsed, err := ParseSubscriptionBundle([]byte(payload))
	if err != nil {
		t.Fatalf("ParseSubscriptionBundle returned error: %v", err)
	}
	if len(parsed.Rules) != 1 || parsed.Rules[0].TargetOutbound != "OpenAI" {
		t.Fatalf("expected only OpenAI business rule retained, got %+v", parsed.Rules)
	}
	if len(parsed.BusinessGroups) != 1 || parsed.BusinessGroups[0].TargetOutbound != "OpenAI" {
		t.Fatalf("expected only OpenAI business group mapping, got %+v", parsed.BusinessGroups)
	}
}

func assertHasType(t *testing.T, list []OutboundItem, typ string) {
	t.Helper()
	for _, item := range list {
		if item.Type == typ {
			return
		}
	}
	t.Fatalf("expected outbound type %q in %#v", typ, list)
}

func assertAppErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	appErr, ok := err.(*errorx.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.Code != code {
		t.Fatalf("expected app error code %s, got %s", code, appErr.Code)
	}
}
