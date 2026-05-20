package service

import (
	"context"
	"database/sql"
	"log"
	"time"

	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util/errorx"
)

func Reload(ctx context.Context, db *sql.DB, configPath string) (version int, hash string, output string, err error) {
	startedAt := time.Now()
	ps, err := loadProxySettings(db)
	if err != nil {
		return 0, "", "", err
	}
	forwardingRunning, err := loadForwardingRunning(db)
	if err != nil {
		return 0, "", "", err
	}
	if !forwardingRunning {
		ps.HTTP.Enabled = false
		ps.Socks.Enabled = false
		ps.Redirect.Enabled = false
	}
	routing, _, err := LoadRoutingSettings(db)
	if err != nil {
		return 0, "", "", err
	}
	cfg, tags, h, err := BuildConfigFromDB(db, ps, routing, forwardingRunning)
	if err != nil {
		return 0, "", "", err
	}

	prevRow, _ := repo.GetRuntimeState(db)
	prevVersion := 0
	prevHash := ""
	if prevRow != nil {
		prevVersion = prevRow.ConfigVersion
		prevHash = prevRow.ConfigHash
	}

	out, err := applyConfigWithPreflight(ctx, configPath, cfg, ps, routing.ListenerReadyMaxMs)
	durationMs := int(time.Since(startedAt).Milliseconds())
	if durationMs < 0 {
		durationMs = 0
	}
	if err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			// Log all possible output fields from different error types
			keys := []string{"output", "restart_output", "rollback_output"}
			for _, k := range keys {
				if detailOut, ok := appErr.Details[k].(string); ok && detailOut != "" {
					log.Printf("runtime reload failure %s:\n%s", k, detailOut)
				}
			}
			// Log specific error messages
			msgKeys := []string{"rollback_restart", "rollback_error", "original_err"}
			for _, k := range msgKeys {
				if msg, ok := appErr.Details[k].(string); ok {
					log.Printf("runtime reload error detail %s: %s", k, msg)
				}
			}
		}
		_ = repo.UpdateRuntimeState(db, prevVersion, prevHash, err.Error(), len(tags), durationMs, false)
		return prevVersion, prevHash, string(out), err
	}
	v := prevVersion + 1
	_ = repo.UpdateRuntimeState(db, v, h, "", len(tags), durationMs, true)
	return v, h, string(out), nil
}

func loadForwardingRunning(db *sql.DB) (bool, error) {
	row, err := repo.GetRuntimeState(db)
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, nil
	}
	return row.ForwardingRunning == 1, nil
}

func loadProxySettings(db *sql.DB) (generator.ProxyInbounds, error) {
	rows, err := repo.GetProxySettings(db)
	if err != nil {
		return generator.ProxyInbounds{}, err
	}
	httpRow := rows["http"]
	socksRow := rows["socks"]
	redirectRow := rows["redirect"]
	ps := generator.ProxyInbounds{
		HTTP: generator.ProxyInbound{
			Type:          "http",
			ListenAddress: defaultStr(httpRow.ListenAddress, "0.0.0.0"),
			Port:          defaultPort(httpRow.Port, 7890),
			Enabled:       httpRow.Enabled == 1,
			AuthMode:      httpRow.AuthMode,
			Username:      httpRow.Username,
			Password:      httpRow.Password,
		},
		Socks: generator.ProxyInbound{
			Type:          "socks",
			ListenAddress: defaultStr(socksRow.ListenAddress, "0.0.0.0"),
			Port:          defaultPort(socksRow.Port, 7891),
			Enabled:       socksRow.Enabled == 1,
			AuthMode:      socksRow.AuthMode,
			Username:      socksRow.Username,
			Password:      socksRow.Password,
		},
		Redirect: generator.ProxyInbound{
			Type:          "redirect",
			ListenAddress: defaultStr(redirectRow.ListenAddress, "0.0.0.0"),
			Port:          defaultPort(redirectRow.Port, generator.DefaultRedirectPort),
			Enabled:       redirectRow.Enabled == 1,
		},
	}
	return ps, nil
}

func defaultStr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func defaultPort(v, fallback int) int {
	if v <= 0 {
		return fallback
	}
	return v
}
