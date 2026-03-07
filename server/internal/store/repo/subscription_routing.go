package repo

import (
	"database/sql"
)

type SubscriptionRuleSetRow struct {
	ID         string
	SubID      string
	Tag        string
	SourceType string
	Format     string
	URL        string
	Path       string
	CreatedAt  string
}

type SubscriptionRuleRow struct {
	ID             string
	SubID          string
	SourceKind     string
	Priority       int
	RuleOrder      int
	MatcherType    string
	MatcherValue   string
	TargetOutbound string
	CreatedAt      string
}

func ReplaceSubscriptionRouting(db *sql.DB, subID string, ruleSets []SubscriptionRuleSetRow, rules []SubscriptionRuleRow) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM subscription_rule_sets WHERE sub_id = ?", subID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM subscription_rules WHERE sub_id = ?", subID); err != nil {
		return err
	}

	for _, rs := range ruleSets {
		if _, err := tx.Exec(
			"INSERT INTO subscription_rule_sets (id, sub_id, tag, source_type, format, url, path, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			rs.ID, subID, rs.Tag, rs.SourceType, rs.Format, nullStr(rs.URL), nullStr(rs.Path), rs.CreatedAt,
		); err != nil {
			return err
		}
	}
	for _, r := range rules {
		if _, err := tx.Exec(
			"INSERT INTO subscription_rules (id, sub_id, source_kind, priority, rule_order, matcher_type, matcher_value, target_outbound, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			r.ID, subID, r.SourceKind, r.Priority, r.RuleOrder, r.MatcherType, r.MatcherValue, r.TargetOutbound, r.CreatedAt,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func ListEnabledSubscriptionRuleSets(db *sql.DB) ([]SubscriptionRuleSetRow, error) {
	rows, err := db.Query(`
		SELECT rs.id, rs.sub_id, rs.tag, rs.source_type, rs.format, rs.url, rs.path, rs.created_at
		FROM subscription_rule_sets rs
		INNER JOIN subscriptions s ON s.id = rs.sub_id
		WHERE s.enabled = 1
		ORDER BY rs.created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubscriptionRuleSetRow
	for rows.Next() {
		var r SubscriptionRuleSetRow
		var url sql.NullString
		var path sql.NullString
		if err := rows.Scan(&r.ID, &r.SubID, &r.Tag, &r.SourceType, &r.Format, &url, &path, &r.CreatedAt); err != nil {
			return nil, err
		}
		if url.Valid {
			r.URL = url.String
		}
		if path.Valid {
			r.Path = path.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func ListEnabledSubscriptionRules(db *sql.DB) ([]SubscriptionRuleRow, error) {
	rows, err := db.Query(`
		SELECT r.id, r.sub_id, r.source_kind, r.priority, r.rule_order, r.matcher_type, r.matcher_value, r.target_outbound, r.created_at
		FROM subscription_rules r
		INNER JOIN subscriptions s ON s.id = r.sub_id
		WHERE s.enabled = 1
		ORDER BY r.priority, r.rule_order, r.created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubscriptionRuleRow
	for rows.Next() {
		var r SubscriptionRuleRow
		if err := rows.Scan(
			&r.ID, &r.SubID, &r.SourceKind, &r.Priority, &r.RuleOrder, &r.MatcherType, &r.MatcherValue, &r.TargetOutbound, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
