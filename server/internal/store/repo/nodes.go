package repo

import (
	"database/sql"
	"time"

	"boxpilot/server/internal/util/errorx"
)

type NodeRow struct {
	ID                string
	SubID             string
	Tag               string
	Name              string
	Type              string
	Enabled           int
	ForwardingEnabled int
	OutboundJSON      string
	CreatedAt         string
	LastTestAt        sql.NullString
	LastLatencyMs     sql.NullInt64
	LastTestStatus    sql.NullString
	LastTestError     sql.NullString
}

func ListNodes(db *sql.DB, subID string, enabled *int) ([]NodeRow, error) {
	query := "SELECT id, sub_id, tag, name, type, enabled, forwarding_enabled, outbound_json, created_at, last_test_at, last_latency_ms, last_test_status, last_test_error FROM nodes WHERE 1=1"
	args := []any{}
	if subID != "" {
		query += " AND sub_id = ?"
		args = append(args, subID)
	}
	if enabled != nil {
		query += " AND enabled = ?"
		args = append(args, *enabled)
	}
	query += " ORDER BY created_at"
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []NodeRow
	for rows.Next() {
		var r NodeRow
		if err := rows.Scan(
			&r.ID, &r.SubID, &r.Tag, &r.Name, &r.Type, &r.Enabled, &r.ForwardingEnabled,
			&r.OutboundJSON, &r.CreatedAt, &r.LastTestAt, &r.LastLatencyMs, &r.LastTestStatus, &r.LastTestError,
		); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func GetNode(db *sql.DB, id string) (*NodeRow, error) {
	var r NodeRow
	err := db.QueryRow("SELECT id, sub_id, tag, name, type, enabled, forwarding_enabled, outbound_json, created_at, last_test_at, last_latency_ms, last_test_status, last_test_error FROM nodes WHERE id = ?", id).Scan(
		&r.ID, &r.SubID, &r.Tag, &r.Name, &r.Type, &r.Enabled, &r.ForwardingEnabled, &r.OutboundJSON, &r.CreatedAt,
		&r.LastTestAt, &r.LastLatencyMs, &r.LastTestStatus, &r.LastTestError,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func ReplaceNodesForSubscription(db *sql.DB, subID string, nodes []NodeRow) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	oldForwarding := map[string]int{}
	oldRows, err := tx.Query("SELECT tag, forwarding_enabled FROM nodes WHERE sub_id = ?", subID)
	if err == nil {
		defer oldRows.Close()
		for oldRows.Next() {
			var tag string
			var forwardingEnabled int
			if scanErr := oldRows.Scan(&tag, &forwardingEnabled); scanErr == nil {
				oldForwarding[tag] = forwardingEnabled
			}
		}
	}
	if _, err := tx.Exec("DELETE FROM nodes WHERE sub_id = ?", subID); err != nil {
		return err
	}
	for _, n := range nodes {
		forwardingEnabled := n.ForwardingEnabled
		if old, ok := oldForwarding[n.Tag]; ok {
			forwardingEnabled = old
		}
		if _, err := tx.Exec("INSERT INTO nodes (id, sub_id, tag, name, type, enabled, forwarding_enabled, outbound_json, created_at, last_test_at, last_latency_ms, last_test_status, last_test_error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, NULL, NULL)",
			n.ID, n.SubID, n.Tag, n.Name, n.Type, n.Enabled, forwardingEnabled, n.OutboundJSON, n.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func UpdateNode(db *sql.DB, id string, name *string, enabled *int, forwardingEnabled *int) (bool, error) {
	res, err := db.Exec("UPDATE nodes SET name = COALESCE(?, name), enabled = COALESCE(?, enabled), forwarding_enabled = COALESCE(?, forwarding_enabled) WHERE id = ?", name, enabled, forwardingEnabled, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func SetNodeProbeResult(db *sql.DB, id string, latencyMs *int, status, errMsg string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		"UPDATE nodes SET last_test_at = ?, last_latency_ms = ?, last_test_status = ?, last_test_error = ? WHERE id = ?",
		now, nullableInt(latencyMs), status, nullableString(errMsg), id,
	)
	return err
}

func nullableInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func EnsureNodeExists(db *sql.DB, id string) error {
	var c int
	err := db.QueryRow("SELECT 1 FROM nodes WHERE id = ?", id).Scan(&c)
	if err == sql.ErrNoRows {
		return errorx.New(errorx.NODENotFound, "node not found").WithDetails(map[string]any{"id": id})
	}
	return err
}

func ListEnabledNodes(db *sql.DB) ([]NodeRow, error) {
	one := 1
	return ListNodes(db, "", &one)
}

func ListEnabledForwardingNodes(db *sql.DB) ([]NodeRow, error) {
	rows, err := db.Query(
		"SELECT id, sub_id, tag, name, type, enabled, forwarding_enabled, outbound_json, created_at, last_test_at, last_latency_ms, last_test_status, last_test_error FROM nodes WHERE enabled = 1 AND forwarding_enabled = 1 ORDER BY created_at",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []NodeRow
	for rows.Next() {
		var r NodeRow
		if err := rows.Scan(
			&r.ID, &r.SubID, &r.Tag, &r.Name, &r.Type, &r.Enabled, &r.ForwardingEnabled,
			&r.OutboundJSON, &r.CreatedAt, &r.LastTestAt, &r.LastLatencyMs, &r.LastTestStatus, &r.LastTestError,
		); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}
