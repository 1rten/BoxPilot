package service

import (
	"database/sql"

	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"
)

type ForwardingPolicy struct {
	HealthyOnlyEnabled  bool
	MaxLatencyMs        int
	AllowUntested       bool
	NodeTestTimeoutMs   int
	NodeTestConcurrency int
	UpdatedAt           string
}

const (
	defaultMaxLatencyMs        = 1200
	defaultNodeTestTimeoutMs   = 3000
	defaultNodeTestConcurrency = 8
)

func LoadForwardingPolicy(db *sql.DB) (ForwardingPolicy, error) {
	row, err := repo.GetForwardingPolicy(db)
	if err != nil {
		return ForwardingPolicy{}, err
	}
	if row == nil {
		return ForwardingPolicy{
			HealthyOnlyEnabled:  true,
			MaxLatencyMs:        defaultMaxLatencyMs,
			AllowUntested:       false,
			NodeTestTimeoutMs:   defaultNodeTestTimeoutMs,
			NodeTestConcurrency: defaultNodeTestConcurrency,
			UpdatedAt:           "",
		}, nil
	}
	p := ForwardingPolicy{
		HealthyOnlyEnabled:  row.HealthyOnlyEnabled == 1,
		MaxLatencyMs:        row.MaxLatencyMs,
		AllowUntested:       row.AllowUntested == 1,
		NodeTestTimeoutMs:   row.NodeTestTimeoutMs,
		NodeTestConcurrency: row.NodeTestConcurrency,
		UpdatedAt:           row.UpdatedAt,
	}
	if p.MaxLatencyMs <= 0 {
		p.MaxLatencyMs = defaultMaxLatencyMs
	}
	if p.NodeTestTimeoutMs <= 0 {
		p.NodeTestTimeoutMs = defaultNodeTestTimeoutMs
	}
	if p.NodeTestConcurrency <= 0 {
		p.NodeTestConcurrency = defaultNodeTestConcurrency
	}
	return p, nil
}

func SaveForwardingPolicy(db *sql.DB, p ForwardingPolicy) (ForwardingPolicy, error) {
	if p.MaxLatencyMs < 1 || p.MaxLatencyMs > 10000 {
		return ForwardingPolicy{}, errorx.New(errorx.REQInvalidField, "max_latency_ms must be between 1 and 10000")
	}
	if p.NodeTestTimeoutMs < 500 || p.NodeTestTimeoutMs > 10000 {
		return ForwardingPolicy{}, errorx.New(errorx.REQInvalidField, "node_test_timeout_ms must be between 500 and 10000")
	}
	if p.NodeTestConcurrency < 1 || p.NodeTestConcurrency > 64 {
		return ForwardingPolicy{}, errorx.New(errorx.REQInvalidField, "node_test_concurrency must be between 1 and 64")
	}
	row := repo.ForwardingPolicyRow{
		ID:                  "global",
		HealthyOnlyEnabled:  boolToInt(p.HealthyOnlyEnabled),
		MaxLatencyMs:        p.MaxLatencyMs,
		AllowUntested:       boolToInt(p.AllowUntested),
		NodeTestTimeoutMs:   p.NodeTestTimeoutMs,
		NodeTestConcurrency: p.NodeTestConcurrency,
		UpdatedAt:           util.NowRFC3339(),
	}
	if err := repo.UpsertForwardingPolicy(db, row); err != nil {
		return ForwardingPolicy{}, err
	}
	p.UpdatedAt = row.UpdatedAt
	return p, nil
}
