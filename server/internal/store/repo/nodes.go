package repo

import (
	"database/sql"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"
)

type NodeRow struct {
	ID           string
	SubID        string
	Tag          string
	Name         string
	Type         string
	Enabled      int
	OutboundJSON string
	CreatedAt    string
}

func ListNodes(db *sql.DB, subID string, enabled *int) ([]NodeRow, error) {
	query := "SELECT id, sub_id, tag, name, type, enabled, outbound_json, created_at FROM nodes WHERE 1=1"
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
		if err := rows.Scan(&r.ID, &r.SubID, &r.Tag, &r.Name, &r.Type, &r.Enabled, &r.OutboundJSON, &r.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func GetNode(db *sql.DB, id string) (*NodeRow, error) {
	var r NodeRow
	err := db.QueryRow("SELECT id, sub_id, tag, name, type, enabled, outbound_json, created_at FROM nodes WHERE id = ?", id).Scan(
		&r.ID, &r.SubID, &r.Tag, &r.Name, &r.Type, &r.Enabled, &r.OutboundJSON, &r.CreatedAt)
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
	if _, err := tx.Exec("DELETE FROM nodes WHERE sub_id = ?", subID); err != nil {
		return err
	}
	for _, n := range nodes {
		if _, err := tx.Exec("INSERT INTO nodes (id, sub_id, tag, name, type, enabled, outbound_json, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			n.ID, n.SubID, n.Tag, n.Name, n.Type, n.Enabled, n.OutboundJSON, n.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func UpdateNode(db *sql.DB, id string, name *string, enabled *int) (bool, error) {
	now := util.NowRFC3339()
	res, err := db.Exec("UPDATE nodes SET name = COALESCE(?, name), enabled = COALESCE(?, enabled) WHERE id = ?", name, enabled, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
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
