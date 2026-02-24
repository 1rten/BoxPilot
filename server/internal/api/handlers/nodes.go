package handlers

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"

	"boxpilot/server/internal/api/dto"
	"boxpilot/server/internal/service"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"

	"github.com/gin-gonic/gin"
)

type Nodes struct {
	DB *sql.DB
}

func (h *Nodes) List(c *gin.Context) {
	subID := c.Query("sub_id")
	var enabled *int
	if e := c.Query("enabled"); e != "" {
		v, _ := strconv.Atoi(e)
		enabled = &v
	}
	list, err := repo.ListNodes(h.DB, subID, enabled)
	if err != nil {
		writeError(c, errorx.New(errorx.NODEListFailed, "list nodes").WithDetails(map[string]any{"err": err.Error()}))
		return
	}
	data := make([]dto.Node, 0, len(list))
	for _, r := range list {
		data = append(data, dto.Node{
			ID: r.ID, SubID: r.SubID, Tag: r.Tag, Name: r.Name, Type: r.Type,
			Enabled: r.Enabled == 1, CreatedAt: r.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Nodes) Update(c *gin.Context) {
	var req dto.UpdateNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	if req.ID == "" {
		writeError(c, errorx.New(errorx.REQMissingField, "id required"))
		return
	}
	if err := repo.EnsureNodeExists(h.DB, req.ID); err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.DBError, err.Error()))
		return
	}
	var name *string
	var enabled *int
	if req.Name != "" {
		name = &req.Name
	}
	if req.Enabled != nil {
		v := 0
		if *req.Enabled {
			v = 1
		}
		enabled = &v
	}
	ok, err := repo.UpdateNode(h.DB, req.ID, name, enabled)
	if err != nil {
		writeError(c, errorx.New(errorx.NODEUpdateFailed, "update node"))
		return
	}
	if !ok {
		writeError(c, errorx.New(errorx.NODENotFound, "node not found"))
		return
	}
	row, _ := repo.GetNode(h.DB, req.ID)
	if row != nil {
		c.JSON(http.StatusOK, gin.H{"data": dto.Node{
			ID: row.ID, SubID: row.SubID, Tag: row.Tag, Name: row.Name, Type: row.Type,
			Enabled: row.Enabled == 1, CreatedAt: row.CreatedAt,
		}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": nil})
}

func (h *Nodes) Forwarding(c *gin.Context) {
	nodeID := c.Query("node_id")
	if nodeID == "" {
		writeError(c, errorx.New(errorx.REQMissingField, "node_id required"))
		return
	}
	if err := repo.EnsureNodeExists(h.DB, nodeID); err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.DBError, err.Error()))
		return
	}
	settings, err := repo.GetProxySettings(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get proxy settings"))
		return
	}
	overrides, err := repo.GetNodeProxyOverrides(h.DB, nodeID)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get node proxy overrides"))
		return
	}
	status, errMsg := runtimeStatus(h.DB)
	httpCfg := buildForwardingConfig(settings["http"], overrides["http"], status, errMsg)
	socksCfg := buildForwardingConfig(settings["socks"], overrides["socks"], status, errMsg)
	c.JSON(http.StatusOK, dto.NodeForwardingResponse{
		Data: dto.NodeForwardingData{
			NodeID: nodeID,
			HTTP:   httpCfg,
			Socks:  socksCfg,
		},
	})
}

func (h *Nodes) UpdateForwarding(c *gin.Context) {
	var req dto.UpdateNodeForwardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	if req.NodeID == "" {
		writeError(c, errorx.New(errorx.REQMissingField, "node_id required"))
		return
	}
	if req.ProxyType != "http" && req.ProxyType != "socks" {
		writeError(c, errorx.New(errorx.REQInvalidField, "invalid proxy_type"))
		return
	}
	if req.UseGlobal {
		if err := repo.DeleteNodeProxyOverride(h.DB, req.NodeID, req.ProxyType); err != nil {
			writeError(c, errorx.New(errorx.DBError, "delete node proxy override"))
			return
		}
		h.Forwarding(c)
		return
	}
	if req.Enabled == nil {
		writeError(c, errorx.New(errorx.REQMissingField, "enabled required"))
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
	row := repo.NodeProxyOverrideRow{
		NodeID:    req.NodeID,
		ProxyType: req.ProxyType,
		Enabled:   boolToInt(*req.Enabled),
		Port:      req.Port,
		AuthMode:  req.AuthMode,
		Username:  req.Username,
		Password:  req.Password,
		CreatedAt: util.NowRFC3339(),
		UpdatedAt: util.NowRFC3339(),
	}
	if err := repo.UpsertNodeProxyOverride(h.DB, row); err != nil {
		writeError(c, errorx.New(errorx.DBError, "update node proxy override"))
		return
	}
	h.Forwarding(c)
}

func (h *Nodes) RestartForwarding(c *gin.Context) {
	var req struct {
		NodeID string `json:"node_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	if req.NodeID == "" {
		writeError(c, errorx.New(errorx.REQMissingField, "node_id required"))
		return
	}
	configPath := os.Getenv("SINGBOX_CONFIG")
	if configPath == "" {
		configPath = "/data/sing-box.json"
	}
	if _, _, _, err := service.Reload(c.Request.Context(), h.DB, configPath); err != nil {
		writeError(c, errorx.New(errorx.RTRestartFailed, err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

func buildForwardingConfig(global repo.ProxySettingsRow, override repo.NodeProxyOverrideRow, runtimeStatus string, errMsg *string) dto.ProxyConfig {
	cfg := dto.ProxyConfig{
		ProxyType:     global.ProxyType,
		Enabled:       global.Enabled == 1,
		ListenAddress: global.ListenAddress,
		Port:          global.Port,
		AuthMode:      global.AuthMode,
		Username:      global.Username,
		Password:      global.Password,
		Source:        "global",
	}
	if override.ProxyType != "" {
		cfg.Enabled = override.Enabled == 1
		cfg.Port = override.Port
		cfg.AuthMode = override.AuthMode
		cfg.Username = override.Username
		cfg.Password = override.Password
		cfg.Source = "override"
	}
	cfg.Status = statusFor(cfg.Enabled, runtimeStatus)
	cfg.ErrorMessage = errMsg
	return cfg
}
