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
	status, errMsg := runtimeStatus(h.DB)
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
	status, errMsg := runtimeStatus(h.DB)
	c.JSON(http.StatusOK, dto.ProxySettingsResponse{
		Data: dto.ProxySettingsData{
			HTTP:  proxyRowToDTO(mustGetProxy(h.DB, "http"), status, errMsg, "global"),
			Socks: proxyRowToDTO(mustGetProxy(h.DB, "socks"), status, errMsg, "global"),
		},
	})
}

func (h *Settings) ApplyProxySettings(c *gin.Context) {
	configPath := os.Getenv("SINGBOX_CONFIG")
	if configPath == "" {
		configPath = "/data/sing-box.json"
	}
	v, hsh, out, err := service.Reload(c.Request.Context(), h.DB, configPath)
	if err != nil {
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

func runtimeStatus(db *sql.DB) (string, *string) {
	row, err := repo.GetRuntimeState(db)
	if err != nil || row == nil {
		return "running", nil
	}
	if row.LastReloadError.Valid && row.LastReloadError.String != "" {
		msg := row.LastReloadError.String
		return "error", &msg
	}
	return "running", nil
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
