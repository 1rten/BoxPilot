package repo

import (
	"database/sql"

	"boxpilot/server/internal/util"
)

type ProxySettingsRow struct {
	ProxyType     string
	Enabled       int
	ListenAddress string
	Port          int
	AuthMode      string
	Username      string
	Password      string
	UpdatedAt     string
}

func GetProxySettings(db *sql.DB) (map[string]ProxySettingsRow, error) {
	rows, err := db.Query(`SELECT proxy_type, enabled, listen_address, port, auth_mode, username, password, updated_at FROM proxy_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]ProxySettingsRow{}
	for rows.Next() {
		var r ProxySettingsRow
		if err := rows.Scan(&r.ProxyType, &r.Enabled, &r.ListenAddress, &r.Port, &r.AuthMode, &r.Username, &r.Password, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out[r.ProxyType] = r
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func GetProxySetting(db *sql.DB, proxyType string) (*ProxySettingsRow, error) {
	var r ProxySettingsRow
	err := db.QueryRow(`SELECT proxy_type, enabled, listen_address, port, auth_mode, username, password, updated_at FROM proxy_settings WHERE proxy_type = ?`, proxyType).
		Scan(&r.ProxyType, &r.Enabled, &r.ListenAddress, &r.Port, &r.AuthMode, &r.Username, &r.Password, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func UpsertProxySetting(db *sql.DB, r ProxySettingsRow) error {
	if r.UpdatedAt == "" {
		r.UpdatedAt = util.NowRFC3339()
	}
	_, err := db.Exec(`INSERT INTO proxy_settings (proxy_type, enabled, listen_address, port, auth_mode, username, password, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(proxy_type) DO UPDATE SET
			enabled = excluded.enabled,
			listen_address = excluded.listen_address,
			port = excluded.port,
			auth_mode = excluded.auth_mode,
			username = excluded.username,
			password = excluded.password,
			updated_at = excluded.updated_at`,
		r.ProxyType, r.Enabled, r.ListenAddress, r.Port, r.AuthMode, r.Username, r.Password, r.UpdatedAt,
	)
	return err
}
