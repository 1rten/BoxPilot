package service

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"

	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/runtime"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
)

func Reload(ctx context.Context, db *sql.DB, configPath string) (version int, hash string, output string, err error) {
	httpProxy, socksProxy, err := loadProxySettings(db)
	if err != nil {
		return 0, "", "", err
	}
	cfg, _, h, err := BuildConfigFromDB(db, httpProxy, socksProxy)
	if err != nil {
		return 0, "", "", err
	}
	if err := util.AtomicWrite(filepath.Dir(configPath), filepath.Base(configPath), cfg); err != nil {
		return 0, "", "", err
	}
	container := os.Getenv("SINGBOX_CONTAINER")
	if container == "" {
		container = "singbox"
	}
	out, err := runtime.DockerRestart(ctx, container)
	if err != nil {
		repo.UpdateRuntimeState(db, 0, h, err.Error())
		return 0, h, string(out), err
	}
	row, _ := repo.GetRuntimeState(db)
	v := 0
	if row != nil {
		v = row.ConfigVersion + 1
	}
	repo.UpdateRuntimeState(db, v, h, "")
	return v, h, string(out), nil
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
