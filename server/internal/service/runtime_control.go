package service

import (
	"context"
	"database/sql"
	"time"

	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/store/repo"
)

func Reload(ctx context.Context, db *sql.DB, configPath string) (version int, hash string, output string, err error) {
	startedAt := time.Now()
	httpProxy, socksProxy, err := loadProxySettings(db)
	if err != nil {
		return 0, "", "", err
	}
	forwardingRunning, err := loadForwardingRunning(db)
	if err != nil {
		return 0, "", "", err
	}
	routing, _, err := LoadRoutingSettings(db)
	if err != nil {
		return 0, "", "", err
	}
	cfg, tags, h, err := BuildConfigFromDB(db, httpProxy, socksProxy, routing, forwardingRunning)
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

	out, err := applyConfigWithPreflight(ctx, configPath, cfg)
	durationMs := int(time.Since(startedAt).Milliseconds())
	if durationMs < 0 {
		durationMs = 0
	}
	if err != nil {
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

func loadProxySettings(db *sql.DB) (generator.ProxyInbound, generator.ProxyInbound, error) {
	rows, err := repo.GetProxySettings(db)
	if err != nil {
		return generator.ProxyInbound{}, generator.ProxyInbound{}, err
	}
	httpRow := rows["http"]
	socksRow := rows["socks"]
	httpProxy := generator.ProxyInbound{
		Type:          "http",
		ListenAddress: httpRow.ListenAddress,
		Port:          httpRow.Port,
		Enabled:       httpRow.Enabled == 1,
		AuthMode:      httpRow.AuthMode,
		Username:      httpRow.Username,
		Password:      httpRow.Password,
	}
	socksProxy := generator.ProxyInbound{
		Type:          "socks",
		ListenAddress: socksRow.ListenAddress,
		Port:          socksRow.Port,
		Enabled:       socksRow.Enabled == 1,
		AuthMode:      socksRow.AuthMode,
		Username:      socksRow.Username,
		Password:      socksRow.Password,
	}
	if httpProxy.ListenAddress == "" {
		httpProxy.ListenAddress = "0.0.0.0"
	}
	if httpProxy.Port == 0 {
		httpProxy.Port = 7890
	}
	if socksProxy.ListenAddress == "" {
		socksProxy.ListenAddress = "0.0.0.0"
	}
	if socksProxy.Port == 0 {
		socksProxy.Port = 7891
	}
	return httpProxy, socksProxy, nil
}
