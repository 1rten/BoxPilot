package repo

import (
	"database/sql"

	"boxpilot/server/internal/util"
)

type NodeProxyOverrideRow struct {
	ID        string
	NodeID    string
	ProxyType string
	Enabled   int
	Port      int
	AuthMode  string
	Username  string
	Password  string
	CreatedAt string
	UpdatedAt string
}

func GetNodeProxyOverrides(db *sql.DB, nodeID string) (map[string]NodeProxyOverrideRow, error) {
	rows, err := db.Query(`SELECT id, node_id, proxy_type, enabled, port, auth_mode, username, password, created_at, updated_at
		FROM node_proxy_overrides WHERE node_id = ?`, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]NodeProxyOverrideRow{}
	for rows.Next() {
		var r NodeProxyOverrideRow
		if err := rows.Scan(&r.ID, &r.NodeID, &r.ProxyType, &r.Enabled, &r.Port, &r.AuthMode, &r.Username, &r.Password, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out[r.ProxyType] = r
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func UpsertNodeProxyOverride(db *sql.DB, r NodeProxyOverrideRow) error {
	if r.ID == "" {
		r.ID = util.NewID()
	}
	now := util.NowRFC3339()
	if r.CreatedAt == "" {
		r.CreatedAt = now
	}
	r.UpdatedAt = now
	_, err := db.Exec(`INSERT INTO node_proxy_overrides (id, node_id, proxy_type, enabled, port, auth_mode, username, password, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id, proxy_type) DO UPDATE SET
			enabled = excluded.enabled,
			port = excluded.port,
			auth_mode = excluded.auth_mode,
			username = excluded.username,
			password = excluded.password,
			updated_at = excluded.updated_at`,
		r.ID, r.NodeID, r.ProxyType, r.Enabled, r.Port, r.AuthMode, r.Username, r.Password, r.CreatedAt, r.UpdatedAt)
	return err
}

func DeleteNodeProxyOverride(db *sql.DB, nodeID, proxyType string) error {
	_, err := db.Exec(`DELETE FROM node_proxy_overrides WHERE node_id = ? AND proxy_type = ?`, nodeID, proxyType)
	return err
}
