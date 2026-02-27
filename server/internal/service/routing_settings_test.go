package service

import (
	"strings"
	"testing"

	"boxpilot/server/internal/generator"
)

func TestNormalizeRoutingSettings_DedupAndTrim(t *testing.T) {
	got, err := NormalizeRoutingSettings(generator.RoutingSettings{
		BypassPrivateEnabled: true,
		BypassDomains:        []string{" localhost ", "", "local", "localhost"},
		BypassCIDRs:          []string{"10.0.0.0/8", " 10.0.0.0/8 ", "192.168.0.0/16"},
	})
	if err != nil {
		t.Fatalf("NormalizeRoutingSettings returned error: %v", err)
	}

	if len(got.BypassDomains) != 2 || got.BypassDomains[0] != "localhost" || got.BypassDomains[1] != "local" {
		t.Fatalf("unexpected domains result: %#v", got.BypassDomains)
	}
	if len(got.BypassCIDRs) != 2 || got.BypassCIDRs[0] != "10.0.0.0/8" || got.BypassCIDRs[1] != "192.168.0.0/16" {
		t.Fatalf("unexpected cidrs result: %#v", got.BypassCIDRs)
	}
}

func TestNormalizeRoutingSettings_InvalidCIDR(t *testing.T) {
	_, err := NormalizeRoutingSettings(generator.RoutingSettings{
		BypassPrivateEnabled: true,
		BypassDomains:        []string{"localhost"},
		BypassCIDRs:          []string{"not-a-cidr"},
	})
	if err == nil {
		t.Fatalf("expected error for invalid CIDR")
	}
	if !strings.Contains(err.Error(), "invalid CIDR") {
		t.Fatalf("expected invalid CIDR message, got %v", err)
	}
}
