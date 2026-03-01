package repo

import (
	"database/sql"

	"boxpilot/server/internal/util"
)

type ForwardingPolicyRow struct {
	ID                 string
	HealthyOnlyEnabled int
	MaxLatencyMs       int
	AllowUntested      int
	UpdatedAt          string
}

func GetForwardingPolicy(db *sql.DB) (*ForwardingPolicyRow, error) {
	var r ForwardingPolicyRow
	err := db.QueryRow(
		"SELECT id, healthy_only_enabled, max_latency_ms, allow_untested, updated_at FROM forwarding_policy WHERE id = 'global'",
	).Scan(&r.ID, &r.HealthyOnlyEnabled, &r.MaxLatencyMs, &r.AllowUntested, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func UpsertForwardingPolicy(db *sql.DB, r ForwardingPolicyRow) error {
	if r.ID == "" {
		r.ID = "global"
	}
	if r.UpdatedAt == "" {
		r.UpdatedAt = util.NowRFC3339()
	}
	_, err := db.Exec(
		`INSERT INTO forwarding_policy (id, healthy_only_enabled, max_latency_ms, allow_untested, updated_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   healthy_only_enabled = excluded.healthy_only_enabled,
		   max_latency_ms = excluded.max_latency_ms,
		   allow_untested = excluded.allow_untested,
		   updated_at = excluded.updated_at`,
		r.ID, r.HealthyOnlyEnabled, r.MaxLatencyMs, r.AllowUntested, r.UpdatedAt,
	)
	return err
}
