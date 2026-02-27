package service

import (
	"database/sql"

	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"
)

func BuildConfigFromDB(db *sql.DB, httpProxy, socksProxy generator.ProxyInbound, routing generator.RoutingSettings, forwardingRunning bool) ([]byte, []string, string, error) {
	if !forwardingRunning {
		httpProxy.Enabled = false
		socksProxy.Enabled = false
	}
	nodes, err := repo.ListEnabledForwardingNodes(db)
	if err != nil {
		return nil, nil, "", err
	}
	if forwardingRunning && (httpProxy.Enabled || socksProxy.Enabled) && len(nodes) == 0 {
		return nil, nil, "", errorx.New(errorx.CFGNoEnabledNodes, "no forwarding nodes enabled")
	}
	var jsons []string
	var tags []string
	for _, n := range nodes {
		jsons = append(jsons, n.OutboundJSON)
		tags = append(tags, n.Tag)
	}
	cfg, err := generator.BuildConfig(httpProxy, socksProxy, routing, jsons)
	if err != nil {
		return nil, nil, "", err
	}
	hash := util.JSONHash(cfg)
	return cfg, tags, hash, nil
}
