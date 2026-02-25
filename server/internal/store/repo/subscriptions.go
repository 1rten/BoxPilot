package repo

import (
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"
	"database/sql"
)

type SubscriptionRow struct {
	ID                 string
	Name               string
	URL                string
	Type               string
	Enabled            int
	AutoUpdateEnabled  int
	RefreshIntervalSec int
	Etag               string
	LastModified       string
	LastFetchAt        sql.NullString
	LastSuccessAt      sql.NullString
	LastError          sql.NullString
	CreatedAt          string
	UpdatedAt          string
}

func ListSubscriptions(db *sql.DB, onlyEnabled bool) ([]SubscriptionRow, error) {
	var rows *sql.Rows
	var err error
	if onlyEnabled {
		rows, err = db.Query("SELECT id, name, url, type, enabled, auto_update_enabled, refresh_interval_sec, etag, last_modified, last_fetch_at, last_success_at, last_error, created_at, updated_at FROM subscriptions WHERE enabled = 1 ORDER BY created_at")
	} else {
		rows, err = db.Query("SELECT id, name, url, type, enabled, auto_update_enabled, refresh_interval_sec, etag, last_modified, last_fetch_at, last_success_at, last_error, created_at, updated_at FROM subscriptions ORDER BY created_at")
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []SubscriptionRow
	for rows.Next() {
		var r SubscriptionRow
		err := rows.Scan(&r.ID, &r.Name, &r.URL, &r.Type, &r.Enabled, &r.AutoUpdateEnabled, &r.RefreshIntervalSec, &r.Etag, &r.LastModified, &r.LastFetchAt, &r.LastSuccessAt, &r.LastError, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func GetSubscription(db *sql.DB, id string) (*SubscriptionRow, error) {
	var r SubscriptionRow
	err := db.QueryRow("SELECT id, name, url, type, enabled, auto_update_enabled, refresh_interval_sec, etag, last_modified, last_fetch_at, last_success_at, last_error, created_at, updated_at FROM subscriptions WHERE id = ?", id).Scan(
		&r.ID, &r.Name, &r.URL, &r.Type, &r.Enabled, &r.AutoUpdateEnabled, &r.RefreshIntervalSec, &r.Etag, &r.LastModified, &r.LastFetchAt, &r.LastSuccessAt, &r.LastError, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func CreateSubscription(db *sql.DB, id, name, url, subType string, enabled, autoUpdateEnabled, refreshIntervalSec int) error {
	now := util.NowRFC3339()
	_, err := db.Exec("INSERT INTO subscriptions (id, name, url, type, enabled, auto_update_enabled, refresh_interval_sec, etag, last_modified, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, '', '', ?, ?)",
		id, name, url, subType, enabled, autoUpdateEnabled, refreshIntervalSec, now, now)
	return err
}

func UpdateSubscription(db *sql.DB, id string, name *string, url *string, enabled *int, autoUpdateEnabled *int, refreshIntervalSec *int) error {
	// Build update dynamically to only touch provided fields
	now := util.NowRFC3339()
	_, err := db.Exec("UPDATE subscriptions SET name = COALESCE(?, name), url = COALESCE(?, url), enabled = COALESCE(?, enabled), auto_update_enabled = COALESCE(?, auto_update_enabled), refresh_interval_sec = COALESCE(?, refresh_interval_sec), updated_at = ? WHERE id = ?",
		name, url, enabled, autoUpdateEnabled, refreshIntervalSec, now, id)
	return err
}

func DeleteSubscription(db *sql.DB, id string) (bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM nodes WHERE sub_id = ?", id); err != nil {
		return false, err
	}
	res, err := tx.Exec("DELETE FROM subscriptions WHERE id = ?", id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return n > 0, nil
}

func SetSubscriptionFetchResult(db *sql.DB, id, etag, lastModified, lastError string, success bool) error {
	now := util.NowRFC3339()
	var lastSuccessAt string
	if success {
		lastSuccessAt = now
	}
	_, err := db.Exec("UPDATE subscriptions SET etag = ?, last_modified = ?, last_fetch_at = ?, last_success_at = ?, last_error = ?, updated_at = ? WHERE id = ?",
		etag, lastModified, now, nullStr(lastSuccessAt), nullStr(lastError), now, id)
	return err
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// EnsureSubscriptionExists returns error if not found.
func EnsureSubscriptionExists(db *sql.DB, id string) error {
	var c int
	err := db.QueryRow("SELECT 1 FROM subscriptions WHERE id = ?", id).Scan(&c)
	if err == sql.ErrNoRows {
		return errorx.New(errorx.SUBNotFound, "subscription not found").WithDetails(map[string]any{"id": id})
	}
	return err
}
