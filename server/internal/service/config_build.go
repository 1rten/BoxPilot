package service

import (
	"database/sql"

	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
)

func BuildConfigFromDB(db *sql.DB, httpProxy, socksProxy generator.ProxyInbound) ([]byte, []string, string, error) {
	nodes, err := repo.ListEnabledNodes(db)
	if err != nil {
		return nil, nil, "", err
	}
	var jsons []string
	var tags []string
	for _, n := range nodes {
		jsons = append(jsons, n.OutboundJSON)
		tags = append(tags, n.Tag)
	}
	cfg, err := generator.BuildConfig(httpProxy, socksProxy, jsons)
	if err != nil {
		return nil, nil, "", err
	}
	hash := util.JSONHash(cfg)
	return cfg, tags, hash, nil
}
