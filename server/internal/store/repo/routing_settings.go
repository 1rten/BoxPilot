package repo

import "database/sql"

type RoutingSettingsRow struct {
	ID                   string
	BypassPrivateEnabled int
	BypassDomainsJSON    string
	BypassCIDRsJSON      string
	UpdatedAt            string
}

func GetRoutingSettings(db *sql.DB) (*RoutingSettingsRow, error) {
	var r RoutingSettingsRow
	err := db.QueryRow(`SELECT id, bypass_private_enabled, bypass_domains_json, bypass_cidrs_json, updated_at FROM routing_settings WHERE id = 'global'`).
		Scan(&r.ID, &r.BypassPrivateEnabled, &r.BypassDomainsJSON, &r.BypassCIDRsJSON, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func UpsertRoutingSettings(db *sql.DB, r RoutingSettingsRow) error {
	_, err := db.Exec(`INSERT INTO routing_settings (id, bypass_private_enabled, bypass_domains_json, bypass_cidrs_json, updated_at)
		VALUES ('global', ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			bypass_private_enabled = excluded.bypass_private_enabled,
			bypass_domains_json = excluded.bypass_domains_json,
			bypass_cidrs_json = excluded.bypass_cidrs_json,
			updated_at = excluded.updated_at`,
		r.BypassPrivateEnabled, r.BypassDomainsJSON, r.BypassCIDRsJSON, r.UpdatedAt,
	)
	return err
}
