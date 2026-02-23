package service

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"

	"boxpilot/server/internal/runtime"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
)

func Reload(ctx context.Context, db *sql.DB, configPath string) (version int, hash string, output string, err error) {
	httpPort := 7890
	if p := os.Getenv("HTTP_PROXY_PORT"); p != "" {
		if v, e := parseInt(p); e == nil {
			httpPort = v
		}
	}
	socksPort := 7891
	if p := os.Getenv("SOCKS_PROXY_PORT"); p != "" {
		if v, e := parseInt(p); e == nil {
			socksPort = v
		}
	}
	cfg, _, h, err := BuildConfigFromDB(db, httpPort, socksPort)
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

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
