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
	LastReloadAt      sql.NullString
	LastReloadError   sql.NullString
}

func GetRuntimeState(db *sql.DB) (*RuntimeStateRow, error) {
	var r RuntimeStateRow
	err := db.QueryRow("SELECT id, config_version, config_hash, forwarding_running, last_reload_at, last_reload_error FROM runtime_state WHERE id = 'runtime'").Scan(
		&r.ID, &r.ConfigVersion, &r.ConfigHash, &r.ForwardingRunning, &r.LastReloadAt, &r.LastReloadError)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func UpdateRuntimeState(db *sql.DB, configVersion int, configHash, lastReloadError string) error {
	now := util.NowRFC3339()
	_, err := db.Exec("UPDATE runtime_state SET config_version = ?, config_hash = ?, last_reload_at = ?, last_reload_error = ? WHERE id = 'runtime'",
		configVersion, configHash, now, nullStr(lastReloadError))
	return err
}

func SetForwardingRunning(db *sql.DB, running int) error {
	_, err := db.Exec("UPDATE runtime_state SET forwarding_running = ? WHERE id = 'runtime'", running)
	return err
}
