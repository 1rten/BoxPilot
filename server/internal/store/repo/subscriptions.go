package repo

import (
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"
	"database/sql"
	"strings"
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
	SubUploadBytes     sql.NullInt64
	SubDownloadBytes   sql.NullInt64
	SubTotalBytes      sql.NullInt64
	SubExpireUnix      sql.NullInt64
	SubUserinfoRaw     sql.NullString
	SubProfileWebPage  sql.NullString
	SubProfileInterval sql.NullInt64
	SubUserinfoUpdated sql.NullString
	CreatedAt          string
	UpdatedAt          string
}

type SubscriptionUsageMeta struct {
	UploadBytes          *int64
	DownloadBytes        *int64
	TotalBytes           *int64
	ExpireUnix           *int64
	UserinfoRaw          *string
	ProfileWebPage       *string
	ProfileUpdateSeconds *int
	UserinfoUpdatedAt    *string
}

func ListSubscriptions(db *sql.DB, onlyEnabled bool) ([]SubscriptionRow, error) {
	query := `SELECT id, name, url, type, enabled, auto_update_enabled, refresh_interval_sec, etag, last_modified, last_fetch_at, last_success_at, last_error,
		sub_upload_bytes, sub_download_bytes, sub_total_bytes, sub_expire_unix, sub_userinfo_raw, sub_profile_web_page, sub_profile_update_interval_sec, sub_userinfo_updated_at,
		created_at, updated_at FROM subscriptions`
	var rows *sql.Rows
	var err error
	if onlyEnabled {
		rows, err = db.Query(query + " WHERE enabled = 1 ORDER BY created_at")
	} else {
		rows, err = db.Query(query + " ORDER BY created_at")
	}
	if isMissingColumnErr(err) {
		legacy := "SELECT id, name, url, type, enabled, auto_update_enabled, refresh_interval_sec, etag, last_modified, last_fetch_at, last_success_at, last_error, created_at, updated_at FROM subscriptions"
		if onlyEnabled {
			rows, err = db.Query(legacy + " WHERE enabled = 1 ORDER BY created_at")
		} else {
			rows, err = db.Query(legacy + " ORDER BY created_at")
		}
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var list []SubscriptionRow
		for rows.Next() {
			var r SubscriptionRow
			err := rows.Scan(
				&r.ID, &r.Name, &r.URL, &r.Type, &r.Enabled, &r.AutoUpdateEnabled, &r.RefreshIntervalSec, &r.Etag, &r.LastModified,
				&r.LastFetchAt, &r.LastSuccessAt, &r.LastError,
				&r.CreatedAt, &r.UpdatedAt,
			)
			if err != nil {
				return nil, err
			}
			list = append(list, r)
		}
		return list, rows.Err()
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []SubscriptionRow
	for rows.Next() {
		var r SubscriptionRow
		err := rows.Scan(
			&r.ID, &r.Name, &r.URL, &r.Type, &r.Enabled, &r.AutoUpdateEnabled, &r.RefreshIntervalSec, &r.Etag, &r.LastModified,
			&r.LastFetchAt, &r.LastSuccessAt, &r.LastError,
			&r.SubUploadBytes, &r.SubDownloadBytes, &r.SubTotalBytes, &r.SubExpireUnix, &r.SubUserinfoRaw, &r.SubProfileWebPage, &r.SubProfileInterval, &r.SubUserinfoUpdated,
			&r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func GetSubscription(db *sql.DB, id string) (*SubscriptionRow, error) {
	var r SubscriptionRow
	err := db.QueryRow(`SELECT id, name, url, type, enabled, auto_update_enabled, refresh_interval_sec, etag, last_modified, last_fetch_at, last_success_at, last_error,
		sub_upload_bytes, sub_download_bytes, sub_total_bytes, sub_expire_unix, sub_userinfo_raw, sub_profile_web_page, sub_profile_update_interval_sec, sub_userinfo_updated_at,
		created_at, updated_at FROM subscriptions WHERE id = ?`, id).Scan(
		&r.ID, &r.Name, &r.URL, &r.Type, &r.Enabled, &r.AutoUpdateEnabled, &r.RefreshIntervalSec, &r.Etag, &r.LastModified,
		&r.LastFetchAt, &r.LastSuccessAt, &r.LastError,
		&r.SubUploadBytes, &r.SubDownloadBytes, &r.SubTotalBytes, &r.SubExpireUnix, &r.SubUserinfoRaw, &r.SubProfileWebPage, &r.SubProfileInterval, &r.SubUserinfoUpdated,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if isMissingColumnErr(err) {
		err = db.QueryRow("SELECT id, name, url, type, enabled, auto_update_enabled, refresh_interval_sec, etag, last_modified, last_fetch_at, last_success_at, last_error, created_at, updated_at FROM subscriptions WHERE id = ?", id).Scan(
			&r.ID, &r.Name, &r.URL, &r.Type, &r.Enabled, &r.AutoUpdateEnabled, &r.RefreshIntervalSec, &r.Etag, &r.LastModified,
			&r.LastFetchAt, &r.LastSuccessAt, &r.LastError, &r.CreatedAt, &r.UpdatedAt,
		)
	}
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
	successFlag := 0
	if success {
		successFlag = 1
	}
	_, err := db.Exec(`UPDATE subscriptions
		SET etag = ?,
		    last_modified = ?,
		    last_fetch_at = ?,
		    last_success_at = CASE WHEN ? = 1 THEN ? ELSE last_success_at END,
		    last_error = ?,
		    updated_at = ?
		WHERE id = ?`,
		etag, lastModified, now, successFlag, now, nullStr(lastError), now, id)
	return err
}

func UpdateSubscriptionUsageMeta(db *sql.DB, id string, meta SubscriptionUsageMeta) error {
	_, err := db.Exec(`UPDATE subscriptions
		SET sub_upload_bytes = ?,
		    sub_download_bytes = ?,
		    sub_total_bytes = ?,
		    sub_expire_unix = ?,
		    sub_userinfo_raw = ?,
		    sub_profile_web_page = ?,
		    sub_profile_update_interval_sec = ?,
		    sub_userinfo_updated_at = ?
		WHERE id = ?`,
		nullInt64(meta.UploadBytes),
		nullInt64(meta.DownloadBytes),
		nullInt64(meta.TotalBytes),
		nullInt64(meta.ExpireUnix),
		nullString(meta.UserinfoRaw),
		nullString(meta.ProfileWebPage),
		nullInt(meta.ProfileUpdateSeconds),
		nullString(meta.UserinfoUpdatedAt),
		id,
	)
	return err
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullInt64(v *int64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func nullInt(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func nullString(v *string) interface{} {
	if v == nil || *v == "" {
		return nil
	}
	return *v
}

func isMissingColumnErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "no such column")
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
