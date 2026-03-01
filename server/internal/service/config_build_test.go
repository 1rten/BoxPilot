package service

import (
	"database/sql"
	"testing"

	"boxpilot/server/internal/store/repo"
)

func TestFilterForwardingNodes(t *testing.T) {
	nodes := []repo.NodeRow{
		{
			ID:             "ok-fast",
			LastTestStatus: sql.NullString{String: "ok", Valid: true},
			LastLatencyMs:  sql.NullInt64{Int64: 180, Valid: true},
		},
		{
			ID:             "ok-slow",
			LastTestStatus: sql.NullString{String: "ok", Valid: true},
			LastLatencyMs:  sql.NullInt64{Int64: 1800, Valid: true},
		},
		{
			ID:             "err",
			LastTestStatus: sql.NullString{String: "error", Valid: true},
			LastLatencyMs:  sql.NullInt64{Int64: 120, Valid: true},
		},
		{
			ID:             "untested",
			LastTestStatus: sql.NullString{},
			LastLatencyMs:  sql.NullInt64{},
		},
	}

	t.Run("healthy only and no untested", func(t *testing.T) {
		out := FilterForwardingNodes(nodes, ForwardingPolicy{
			HealthyOnlyEnabled: true,
			MaxLatencyMs:       500,
			AllowUntested:      false,
		})
		if len(out) != 1 || out[0].ID != "ok-fast" {
			t.Fatalf("unexpected filtered nodes: %#v", out)
		}
	})

	t.Run("healthy only and allow untested", func(t *testing.T) {
		out := FilterForwardingNodes(nodes, ForwardingPolicy{
			HealthyOnlyEnabled: true,
			MaxLatencyMs:       500,
			AllowUntested:      true,
		})
		if len(out) != 2 {
			t.Fatalf("expected 2 nodes, got %d", len(out))
		}
	})

	t.Run("policy disabled", func(t *testing.T) {
		out := FilterForwardingNodes(nodes, ForwardingPolicy{
			HealthyOnlyEnabled: false,
			MaxLatencyMs:       500,
			AllowUntested:      false,
		})
		if len(out) != len(nodes) {
			t.Fatalf("expected all nodes, got %d", len(out))
		}
	})
}
