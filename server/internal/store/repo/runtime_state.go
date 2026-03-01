package repo

import (
	"boxpilot/server/internal/util"
	"database/sql"
)

type RuntimeStateRow struct {
	ID                string
	ConfigVersion     int
	ConfigHash        string
	ForwardingRunning int
	LastNodesIncluded int
	LastApplyDuration int
	LastReloadAt      sql.NullString
	LastApplySuccess  sql.NullString
	LastReloadError   sql.NullString
}

func GetRuntimeState(db *sql.DB) (*RuntimeStateRow, error) {
	var r RuntimeStateRow
	err := db.QueryRow(
		"SELECT id, config_version, config_hash, forwarding_running, last_nodes_included, last_apply_duration_ms, last_reload_at, last_apply_success_at, last_reload_error FROM runtime_state WHERE id = 'runtime'",
	).Scan(
		&r.ID, &r.ConfigVersion, &r.ConfigHash, &r.ForwardingRunning, &r.LastNodesIncluded, &r.LastApplyDuration, &r.LastReloadAt, &r.LastApplySuccess, &r.LastReloadError,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func UpdateRuntimeState(db *sql.DB, configVersion int, configHash, lastReloadError string, nodesIncluded int, applyDurationMs int, success bool) error {
	now := util.NowRFC3339()
	successFlag := 0
	if success {
		successFlag = 1
	}
	_, err := db.Exec(
		`UPDATE runtime_state SET
		   config_version = ?,
		   config_hash = ?,
		   last_nodes_included = ?,
		   last_apply_duration_ms = ?,
		   last_reload_at = ?,
		   last_reload_error = ?,
		   last_apply_success_at = CASE WHEN ? = 1 THEN ? ELSE last_apply_success_at END
		 WHERE id = 'runtime'`,
		configVersion, configHash, nodesIncluded, applyDurationMs, now, nullStr(lastReloadError), successFlag, now,
	)
	return err
}

func SetForwardingRunning(db *sql.DB, running int) error {
	_, err := db.Exec("UPDATE runtime_state SET forwarding_running = ? WHERE id = 'runtime'", running)
	return err
}
