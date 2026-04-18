package service

import (
	"encoding/json"
	"testing"
)

func TestMergeOutboundJSONTag(t *testing.T) {
	raw := []byte(`{"type":"vless","tag":"","server":"x.example","server_port":443}`)
	out, err := mergeOutboundJSONTag(raw, "manual-node-2")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if e := json.Unmarshal([]byte(out), &m); e != nil {
		t.Fatal(e)
	}
	if m["tag"] != "manual-node-2" {
		t.Fatalf("tag: want manual-node-2, got %v", m["tag"])
	}
	if m["server"] != "x.example" {
		t.Fatalf("server field lost")
	}
}

func TestMergeOutboundJSONTag_idempotent(t *testing.T) {
	raw := []byte(`{"type":"trojan","tag":"my-tag","server":"1.1.1.1","server_port":443,"password":"x"}`)
	out, err := mergeOutboundJSONTag(raw, "my-tag")
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if e := json.Unmarshal([]byte(out), &got); e != nil {
		t.Fatal(e)
	}
	if got["tag"] != "my-tag" || got["type"] != "trojan" || got["password"] != "x" {
		t.Fatalf("unexpected payload: %#v", got)
	}
}
