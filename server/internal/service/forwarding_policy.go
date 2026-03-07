package service

import (
	"database/sql"
	"strconv"

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
	BizAutoIntervalSec  int
	UpdatedAt           string
}

const (
	defaultMaxLatencyMs        = 1200
	defaultNodeTestTimeoutMs   = 3000
	defaultNodeTestConcurrency = 8
	defaultBizAutoIntervalSec  = 1800
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
			BizAutoIntervalSec:  defaultBizAutoIntervalSec,
			UpdatedAt:           "",
		}, nil
	}
	p := ForwardingPolicy{
		HealthyOnlyEnabled:  row.HealthyOnlyEnabled == 1,
		MaxLatencyMs:        row.MaxLatencyMs,
		AllowUntested:       row.AllowUntested == 1,
		NodeTestTimeoutMs:   row.NodeTestTimeoutMs,
		NodeTestConcurrency: row.NodeTestConcurrency,
		BizAutoIntervalSec:  row.BizAutoIntervalSec,
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
	if p.BizAutoIntervalSec <= 0 {
		p.BizAutoIntervalSec = defaultBizAutoIntervalSec
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
	if p.BizAutoIntervalSec < 60 || p.BizAutoIntervalSec > 86400 {
		return ForwardingPolicy{}, errorx.New(errorx.REQInvalidField, "biz_auto_interval_sec must be between 60 and 86400")
	}
	row := repo.ForwardingPolicyRow{
		ID:                  "global",
		HealthyOnlyEnabled:  boolToInt(p.HealthyOnlyEnabled),
		MaxLatencyMs:        p.MaxLatencyMs,
		AllowUntested:       boolToInt(p.AllowUntested),
		NodeTestTimeoutMs:   p.NodeTestTimeoutMs,
		NodeTestConcurrency: p.NodeTestConcurrency,
		BizAutoIntervalSec:  p.BizAutoIntervalSec,
		UpdatedAt:           util.NowRFC3339(),
	}
	if err := repo.UpsertForwardingPolicy(db, row); err != nil {
		return ForwardingPolicy{}, err
	}
	p.UpdatedAt = row.UpdatedAt
	return p, nil
}

func BizAutoIntervalDuration(sec int) string {
	if sec <= 0 {
		sec = defaultBizAutoIntervalSec
	}
	if sec%3600 == 0 {
		return strconv.Itoa(sec/3600) + "h"
	}
	if sec%60 == 0 {
		return strconv.Itoa(sec/60) + "m"
	}
	return strconv.Itoa(sec) + "s"
}
