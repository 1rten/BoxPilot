package service

import (
	"database/sql"

	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"
)

type ForwardingPolicy struct {
	HealthyOnlyEnabled bool
	MaxLatencyMs       int
	AllowUntested      bool
	UpdatedAt          string
}

const (
	defaultMaxLatencyMs = 1200
)

func LoadForwardingPolicy(db *sql.DB) (ForwardingPolicy, error) {
	row, err := repo.GetForwardingPolicy(db)
	if err != nil {
		return ForwardingPolicy{}, err
	}
	if row == nil {
		return ForwardingPolicy{
			HealthyOnlyEnabled: true,
			MaxLatencyMs:       defaultMaxLatencyMs,
			AllowUntested:      false,
			UpdatedAt:          "",
		}, nil
	}
	p := ForwardingPolicy{
		HealthyOnlyEnabled: row.HealthyOnlyEnabled == 1,
		MaxLatencyMs:       row.MaxLatencyMs,
		AllowUntested:      row.AllowUntested == 1,
		UpdatedAt:          row.UpdatedAt,
	}
	if p.MaxLatencyMs <= 0 {
		p.MaxLatencyMs = defaultMaxLatencyMs
	}
	return p, nil
}

func SaveForwardingPolicy(db *sql.DB, p ForwardingPolicy) (ForwardingPolicy, error) {
	if p.MaxLatencyMs < 1 || p.MaxLatencyMs > 10000 {
		return ForwardingPolicy{}, errorx.New(errorx.REQInvalidField, "max_latency_ms must be between 1 and 10000")
	}
	row := repo.ForwardingPolicyRow{
		ID:                 "global",
		HealthyOnlyEnabled: boolToInt(p.HealthyOnlyEnabled),
		MaxLatencyMs:       p.MaxLatencyMs,
		AllowUntested:      boolToInt(p.AllowUntested),
		UpdatedAt:          util.NowRFC3339(),
	}
	if err := repo.UpsertForwardingPolicy(db, row); err != nil {
		return ForwardingPolicy{}, err
	}
	p.UpdatedAt = row.UpdatedAt
	return p, nil
}
