package handlers

import (
	"database/sql"
	"net/http"

	"boxpilot/server/internal/api/dto"
	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/service"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"

	"github.com/gin-gonic/gin"
)

type Runtime struct {
	DB *sql.DB
}

func (h *Runtime) Status(c *gin.Context) {
	row, err := repo.GetRuntimeState(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get runtime state"))
		return
	}
	cfgVersion := 0
	cfgHash := ""
	forwardingRunning := false
	var lastReloadAt, lastReloadError *string
	if row != nil {
		cfgVersion = row.ConfigVersion
		cfgHash = row.ConfigHash
		forwardingRunning = row.ForwardingRunning == 1
		if row.LastReloadAt.Valid {
			lastReloadAt = &row.LastReloadAt.String
		}
		if row.LastReloadError.Valid {
			lastReloadError = &row.LastReloadError.String
		}
	}
	httpPort := 7890
	socksPort := 7891
	if settings, err := repo.GetProxySettings(h.DB); err == nil {
		if httpRow, ok := settings["http"]; ok && httpRow.Port > 0 {
			httpPort = httpRow.Port
		}
		if socksRow, ok := settings["socks"]; ok && socksRow.Port > 0 {
			socksPort = socksRow.Port
		}
	}
	c.JSON(http.StatusOK, dto.RuntimeStatusResponse{
		Data: dto.RuntimeStatusData{
			ConfigVersion:     cfgVersion,
			ConfigHash:        cfgHash,
			ForwardingRunning: forwardingRunning,
			LastReloadAt:      lastReloadAt,
			LastReloadError:   lastReloadError,
			Ports:             dto.RuntimePorts{HTTP: httpPort, Socks: socksPort},
		},
	})
}

func (h *Runtime) Plan(c *gin.Context) {
	var req dto.RuntimePlanRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
			return
		}
	}

	settings, err := repo.GetProxySettings(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get proxy settings"))
		return
	}
	httpProxy, socksProxy := runtimeProxyRowsToInbounds(settings["http"], settings["socks"])

	row, err := repo.GetRuntimeState(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get runtime state"))
		return
	}
	forwardingRunning := row != nil && row.ForwardingRunning == 1
	if !forwardingRunning {
		httpProxy.Enabled = false
		socksProxy.Enabled = false
	}

	nodes := []repo.NodeRow{}
	if req.IncludeDisabledNodes {
		nodes, err = repo.ListNodes(h.DB, "", nil)
	} else {
		nodes, err = repo.ListEnabledForwardingNodes(h.DB)
	}
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "list nodes for plan"))
		return
	}

	if forwardingRunning && (httpProxy.Enabled || socksProxy.Enabled) && len(nodes) == 0 {
		writeError(c, errorx.New(errorx.CFGNoEnabledNodes, "no forwarding nodes enabled"))
		return
	}

	routing, _, err := service.LoadRoutingSettings(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get routing settings"))
		return
	}

	jsons := make([]string, 0, len(nodes))
	tags := make([]string, 0, len(nodes))
	for _, node := range nodes {
		jsons = append(jsons, node.OutboundJSON)
		tags = append(tags, node.Tag)
	}
	cfg, err := generator.BuildConfig(httpProxy, socksProxy, routing, jsons)
	if err != nil {
		writeError(c, errorx.New(errorx.CFGBuildFailed, "build plan config"))
		return
	}

	c.JSON(http.StatusOK, dto.RuntimePlanResponse{
		Data: dto.RuntimePlanData{
			NodesIncluded: len(nodes),
			Tags:          tags,
			ConfigHash:    util.JSONHash(cfg),
		},
	})
}

func (h *Runtime) Reload(c *gin.Context) {
	configPath := service.ResolveConfigPath()
	v, hsh, out, err := service.Reload(c.Request.Context(), h.DB, configPath)
	if err != nil {
		writeError(c, errorx.New(errorx.RTRestartFailed, err.Error()))
		return
	}
	nodesIncluded := 0
	if nodes, listErr := repo.ListEnabledForwardingNodes(h.DB); listErr == nil {
		nodesIncluded = len(nodes)
	}
	c.JSON(http.StatusOK, dto.RuntimeReloadResponse{
		Data: dto.RuntimeReloadData{
			ConfigVersion: v,
			ConfigHash:    hsh,
			NodesIncluded: nodesIncluded,
			RestartOutput: out,
			ReloadedAt:    util.NowRFC3339(),
		},
	})
}

func runtimeProxyRowsToInbounds(httpRow, socksRow repo.ProxySettingsRow) (generator.ProxyInbound, generator.ProxyInbound) {
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
	return httpProxy, socksProxy
}
