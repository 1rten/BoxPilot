package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureCNRulesetsReady_FailsWithoutCacheWhenRefreshFails(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "sing-box.json")

	cache, err := EnsureCNRulesetsReady(context.Background(), configPath, failingCNRuleSetFetcher)
	if err == nil {
		t.Fatal("expected cache preparation error")
	}
	if cache != nil {
		t.Fatalf("expected nil cache on hard failure, got %#v", cache)
	}
	if !strings.Contains(err.Error(), "geosite-cn") {
		t.Fatalf("expected geosite-cn in error, got %v", err)
	}
}

func TestEnsureCNRulesetsReady_UsesStaleCacheWhenRefreshFails(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "sing-box.json")
	ruleDir := filepath.Join(tmp, "ruleset")
	if err := os.MkdirAll(ruleDir, 0755); err != nil {
		t.Fatalf("mkdir ruleset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ruleDir, "geosite-cn.srs"), []byte("geo-site"), 0644); err != nil {
		t.Fatalf("write geosite cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ruleDir, "geoip-cn.srs"), []byte("geo-ip"), 0644); err != nil {
		t.Fatalf("write geoip cache: %v", err)
	}

	cache, err := EnsureCNRulesetsReady(context.Background(), configPath, failingCNRuleSetFetcher)
	if err != nil {
		t.Fatalf("expected stale cache fallback, got %v", err)
	}
	if cache == nil {
		t.Fatal("expected cache metadata")
	}
	if !cache.Stale {
		t.Fatalf("expected stale cache metadata, got %#v", cache)
	}
	if cache.Warning == "" {
		t.Fatalf("expected warning on stale cache, got %#v", cache)
	}
	if len(cache.RuleSets) != 2 {
		t.Fatalf("expected 2 CN rule sets, got %#v", cache.RuleSets)
	}
}

func failingCNRuleSetFetcher(context.Context, string) ([]byte, error) {
	return nil, os.ErrNotExist
}
