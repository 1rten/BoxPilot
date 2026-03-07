package repo

import "database/sql"

type RuntimeGroupSelectionRow struct {
	GroupTag         string
	SelectedOutbound string
	UpdatedAt        string
}

func UpsertRuntimeGroupSelection(db *sql.DB, groupTag, selectedOutbound, updatedAt string) error {
	_, err := db.Exec(
		`INSERT INTO runtime_group_selections (group_tag, selected_outbound, updated_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(group_tag) DO UPDATE SET
		   selected_outbound = excluded.selected_outbound,
		   updated_at = excluded.updated_at`,
		groupTag, selectedOutbound, updatedAt,
	)
	return err
}

func ListRuntimeGroupSelections(db *sql.DB) ([]RuntimeGroupSelectionRow, error) {
	rows, err := db.Query(
		"SELECT group_tag, selected_outbound, updated_at FROM runtime_group_selections ORDER BY group_tag",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RuntimeGroupSelectionRow{}
	for rows.Next() {
		var r RuntimeGroupSelectionRow
		if err := rows.Scan(&r.GroupTag, &r.SelectedOutbound, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
