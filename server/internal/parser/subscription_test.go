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
