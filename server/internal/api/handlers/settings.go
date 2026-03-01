package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"boxpilot/server/internal/api/dto"
	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/service"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"

	"github.com/gin-gonic/gin"
)

type Settings struct {
	DB *sql.DB
}

func (h *Settings) GetProxySettings(c *gin.Context) {
	settings, err := repo.GetProxySettings(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get proxy settings"))
		return
	}
	httpRow := settings["http"]
	socksRow := settings["socks"]
	_, status, errMsg := runtimeStatus(h.DB)
	c.JSON(http.StatusOK, dto.ProxySettingsResponse{
		Data: dto.ProxySettingsData{
			HTTP:  proxyRowToDTO(httpRow, status, errMsg, "global"),
			Socks: proxyRowToDTO(socksRow, status, errMsg, "global"),
		},
	})
}

func (h *Settings) UpdateProxySettings(c *gin.Context) {
	var req dto.UpdateProxySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	if req.ProxyType != "http" && req.ProxyType != "socks" {
		writeError(c, errorx.New(errorx.REQInvalidField, "invalid proxy_type"))
		return
	}
	if req.Enabled == nil {
		writeError(c, errorx.New(errorx.REQMissingField, "enabled required"))
		return
	}
	if req.ListenAddress != "127.0.0.1" && req.ListenAddress != "0.0.0.0" {
		writeError(c, errorx.New(errorx.REQInvalidField, "invalid listen_address"))
		return
	}
	if req.Port < 1 || req.Port > 65535 {
		writeError(c, errorx.New(errorx.REQInvalidField, "port must be between 1 and 65535"))
		return
	}
	if req.AuthMode != "none" && req.AuthMode != "basic" {
		writeError(c, errorx.New(errorx.REQInvalidField, "invalid auth_mode"))
		return
	}
	if req.AuthMode == "basic" && (req.Username == "" || req.Password == "") {
		writeError(c, errorx.New(errorx.REQMissingField, "username/password required for basic auth"))
		return
	}

	otherType := "http"
	if req.ProxyType == "http" {
		otherType = "socks"
	}
	other, err := repo.GetProxySetting(h.DB, otherType)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get proxy settings"))
		return
	}
	if other != nil && *req.Enabled && other.Enabled == 1 && other.Port == req.Port {
		sameInterface := other.ListenAddress == req.ListenAddress || other.ListenAddress == "0.0.0.0" || req.ListenAddress == "0.0.0.0"
		if sameInterface {
			writeError(c, errorx.New(errorx.REQInvalidField, "HTTP and SOCKS ports conflict on "+req.ListenAddress+":"+itoa(req.Port)))
			return
		}
	}

	row := repo.ProxySettingsRow{
		ProxyType:     req.ProxyType,
		Enabled:       boolToInt(*req.Enabled),
		ListenAddress: req.ListenAddress,
		Port:          req.Port,
		AuthMode:      req.AuthMode,
		Username:      req.Username,
		Password:      req.Password,
		UpdatedAt:     util.NowRFC3339(),
	}
	if err := repo.UpsertProxySetting(h.DB, row); err != nil {
		writeError(c, errorx.New(errorx.DBError, "update proxy settings"))
		return
	}
	_, status, errMsg := runtimeStatus(h.DB)
	c.JSON(http.StatusOK, dto.ProxySettingsResponse{
		Data: dto.ProxySettingsData{
			HTTP:  proxyRowToDTO(mustGetProxy(h.DB, "http"), status, errMsg, "global"),
			Socks: proxyRowToDTO(mustGetProxy(h.DB, "socks"), status, errMsg, "global"),
		},
	})
}

func (h *Settings) ApplyProxySettings(c *gin.Context) {
	configPath := service.ResolveConfigPath()
	v, hsh, out, err := service.Reload(c.Request.Context(), h.DB, configPath)
	if err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.RTRestartFailed, err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.ProxyApplyResponse{
		Data: dto.ProxyApplyData{
			ConfigVersion: v,
			ConfigHash:    hsh,
			RestartOutput: out,
			ReloadedAt:    util.NowRFC3339(),
		},
	})
}

func (h *Settings) GetRoutingSettings(c *gin.Context) {
	settings, updatedAt, err := service.LoadRoutingSettings(h.DB)
	if err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.DBError, "get routing settings"))
		return
	}
	c.JSON(http.StatusOK, dto.RoutingSettingsResponse{
		Data: dto.RoutingSettingsData{
			BypassPrivateEnabled: settings.BypassPrivateEnabled,
			BypassDomains:        settings.BypassDomains,
			BypassCIDRs:          settings.BypassCIDRs,
			UpdatedAt:            updatedAt,
		},
	})
}

func (h *Settings) UpdateRoutingSettings(c *gin.Context) {
	var req dto.UpdateRoutingSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	if req.BypassPrivateEnabled == nil {
		writeError(c, errorx.New(errorx.REQMissingField, "bypass_private_enabled required"))
		return
	}
	saved, updatedAt, err := service.SaveRoutingSettings(h.DB, generator.RoutingSettings{
		BypassPrivateEnabled: *req.BypassPrivateEnabled,
		BypassDomains:        req.BypassDomains,
		BypassCIDRs:          req.BypassCIDRs,
	})
	if err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.DBError, "update routing settings"))
		return
	}
	c.JSON(http.StatusOK, dto.RoutingSettingsResponse{
		Data: dto.RoutingSettingsData{
			BypassPrivateEnabled: saved.BypassPrivateEnabled,
			BypassDomains:        saved.BypassDomains,
			BypassCIDRs:          saved.BypassCIDRs,
			UpdatedAt:            updatedAt,
		},
	})
}

func (h *Settings) RoutingSummary(c *gin.Context) {
	settings, updatedAt, err := service.LoadRoutingSettings(h.DB)
	if err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.DBError, "get routing settings"))
		return
	}

	notes := []string{
		"Routing bypass is applied during runtime config build.",
	}
	if settings.BypassPrivateEnabled {
		notes = append(notes, "Private/local CIDR bypass is enabled.")
	} else {
		notes = append(notes, "Private/local CIDR bypass is disabled.")
	}

	c.JSON(http.StatusOK, dto.RoutingSummaryResponse{
		Data: dto.RoutingSummaryData{
			BypassPrivateEnabled: settings.BypassPrivateEnabled,
			BypassDomainsCount:   len(settings.BypassDomains),
			BypassCIDRsCount:     len(settings.BypassCIDRs),
			UpdatedAt:            updatedAt,
			Notes:                notes,
		},
	})
}

func (h *Settings) ForwardingStatus(c *gin.Context) {
	running, status, errMsg := runtimeStatus(h.DB)
	c.JSON(http.StatusOK, dto.ForwardingRuntimeStatusResponse{
		Data: dto.ForwardingRuntimeStatus{
			Running:      running,
			Status:       status,
			ErrorMessage: errMsg,
		},
	})
}

func (h *Settings) ForwardingSummary(c *gin.Context) {
	running, status, errMsg := runtimeStatus(h.DB)
	nodes, err := repo.ListEnabledForwardingNodes(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "list forwarding nodes"))
		return
	}

	maxNodes := 10
	if len(nodes) < maxNodes {
		maxNodes = len(nodes)
	}
	items := make([]dto.ForwardingSummaryNode, 0, maxNodes)
	for _, n := range nodes[:maxNodes] {
		var lastStatus *string
		var lastLatencyMs *int64
		if n.LastTestStatus.Valid {
			lastStatus = &n.LastTestStatus.String
		}
		if n.LastLatencyMs.Valid {
			v := n.LastLatencyMs.Int64
			lastLatencyMs = &v
		}
		items = append(items, dto.ForwardingSummaryNode{
			ID:            n.ID,
			Name:          n.Name,
			Tag:           n.Tag,
			Type:          n.Type,
			LastStatus:    lastStatus,
			LastLatencyMs: lastLatencyMs,
		})
	}

	c.JSON(http.StatusOK, dto.ForwardingSummaryResponse{
		Data: dto.ForwardingSummaryData{
			Running:            running,
			Status:             status,
			ErrorMessage:       errMsg,
			SelectedNodesCount: len(nodes),
			Nodes:              items,
		},
	})
}

func (h *Settings) GetForwardingPolicy(c *gin.Context) {
	policy, err := service.LoadForwardingPolicy(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get forwarding policy"))
		return
	}
	c.JSON(http.StatusOK, dto.ForwardingPolicyResponse{
		Data: forwardingPolicyToDTO(policy),
	})
}

func (h *Settings) UpdateForwardingPolicy(c *gin.Context) {
	var req dto.UpdateForwardingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	if req.HealthyOnlyEnabled == nil {
		writeError(c, errorx.New(errorx.REQMissingField, "healthy_only_enabled required"))
		return
	}
	if req.AllowUntested == nil {
		writeError(c, errorx.New(errorx.REQMissingField, "allow_untested required"))
		return
	}
	if req.MaxLatencyMs < 1 || req.MaxLatencyMs > 10000 {
		writeError(c, errorx.New(errorx.REQInvalidField, "max_latency_ms must be between 1 and 10000"))
		return
	}
	if req.NodeTestTimeoutMs < 500 || req.NodeTestTimeoutMs > 10000 {
		writeError(c, errorx.New(errorx.REQInvalidField, "node_test_timeout_ms must be between 500 and 10000"))
		return
	}
	if req.NodeTestConcurrency < 1 || req.NodeTestConcurrency > 64 {
		writeError(c, errorx.New(errorx.REQInvalidField, "node_test_concurrency must be between 1 and 64"))
		return
	}
	policy, err := service.SaveForwardingPolicy(h.DB, service.ForwardingPolicy{
		HealthyOnlyEnabled:  *req.HealthyOnlyEnabled,
		MaxLatencyMs:        req.MaxLatencyMs,
		AllowUntested:       *req.AllowUntested,
		NodeTestTimeoutMs:   req.NodeTestTimeoutMs,
		NodeTestConcurrency: req.NodeTestConcurrency,
	})
	if err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.DBError, "update forwarding policy"))
		return
	}
	c.JSON(http.StatusOK, dto.ForwardingPolicyResponse{
		Data: forwardingPolicyToDTO(policy),
	})
}

func (h *Settings) StartForwarding(c *gin.Context) {
	nodes, err := repo.ListEnabledForwardingNodes(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "list forwarding nodes"))
		return
	}
	if len(nodes) == 0 {
		writeError(c, errorx.New(errorx.CFGNoEnabledNodes, "no forwarding nodes selected"))
		return
	}
	settings, err := repo.GetProxySettings(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get proxy settings"))
		return
	}
	httpRow := settings["http"]
	socksRow := settings["socks"]
	if httpRow.Enabled != 1 && socksRow.Enabled != 1 {
		writeError(c, errorx.New(errorx.REQInvalidField, "no proxy inbound enabled"))
		return
	}
	if err := h.setForwardingRunningAndReload(c, true); err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.RTRestartFailed, err.Error()))
		return
	}
	h.ForwardingStatus(c)
}

func (h *Settings) StopForwarding(c *gin.Context) {
	if err := h.setForwardingRunningAndReload(c, false); err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.RTRestartFailed, err.Error()))
		return
	}
	h.ForwardingStatus(c)
}

func (h *Settings) setForwardingRunningAndReload(c *gin.Context, running bool) error {
	row, err := repo.GetRuntimeState(h.DB)
	if err != nil {
		return errorx.New(errorx.DBError, "get runtime state")
	}
	prev := 0
	if row != nil {
		prev = row.ForwardingRunning
	}
	next := 0
	if running {
		next = 1
	}
	if err := repo.SetForwardingRunning(h.DB, next); err != nil {
		return errorx.New(errorx.DBError, "update forwarding state")
	}
	configPath := service.ResolveConfigPath()
	if _, _, _, err := service.Reload(c.Request.Context(), h.DB, configPath); err != nil {
		_ = repo.SetForwardingRunning(h.DB, prev)
		return err
	}
	return nil
}

func proxyRowToDTO(row repo.ProxySettingsRow, status string, errMsg *string, source string) dto.ProxyConfig {
	return dto.ProxyConfig{
		ProxyType:     row.ProxyType,
		Enabled:       row.Enabled == 1,
		ListenAddress: row.ListenAddress,
		Port:          row.Port,
		AuthMode:      row.AuthMode,
		Username:      row.Username,
		Password:      row.Password,
		Status:        statusFor(row.Enabled == 1, status),
		ErrorMessage:  errMsg,
		Source:        source,
	}
}

func statusFor(enabled bool, runtimeStatus string) string {
	if !enabled {
		return "stopped"
	}
	return runtimeStatus
}

func runtimeStatus(db *sql.DB) (bool, string, *string) {
	row, err := repo.GetRuntimeState(db)
	if err != nil || row == nil {
		return false, "stopped", nil
	}
	if row.LastReloadError.Valid && row.LastReloadError.String != "" {
		msg := row.LastReloadError.String
		return row.ForwardingRunning == 1, "error", &msg
	}
	if row.ForwardingRunning != 1 {
		return false, "stopped", nil
	}
	return true, "running", nil
}

func forwardingPolicyToDTO(p service.ForwardingPolicy) dto.ForwardingPolicyData {
	return dto.ForwardingPolicyData{
		HealthyOnlyEnabled:  p.HealthyOnlyEnabled,
		MaxLatencyMs:        p.MaxLatencyMs,
		AllowUntested:       p.AllowUntested,
		NodeTestTimeoutMs:   p.NodeTestTimeoutMs,
		NodeTestConcurrency: p.NodeTestConcurrency,
		UpdatedAt:           p.UpdatedAt,
	}
}

func mustGetProxy(db *sql.DB, proxyType string) repo.ProxySettingsRow {
	row, err := repo.GetProxySetting(db, proxyType)
	if err != nil || row == nil {
		return repo.ProxySettingsRow{
			ProxyType:     proxyType,
			Enabled:       0,
			ListenAddress: "0.0.0.0",
			Port:          defaultPort(proxyType),
			AuthMode:      "none",
		}
	}
	return *row
}

func defaultPort(proxyType string) int {
	if proxyType == "socks" {
		return 7891
	}
	return 7890
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func itoa(v int) string {
	return strconv.Itoa(v)
}
