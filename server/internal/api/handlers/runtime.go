package handlers

import (
	"database/sql"
	"net/http"
	"os"

	"boxpilot/server/internal/api/dto"
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
	var lastReloadAt, lastReloadError *string
	if row != nil {
		cfgVersion = row.ConfigVersion
		cfgHash = row.ConfigHash
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
	mode := "docker"
	if os.Getenv("RUNTIME_MODE") != "" {
		mode = os.Getenv("RUNTIME_MODE")
	}
	container := os.Getenv("SINGBOX_CONTAINER")
	if container == "" {
		container = "singbox"
	}
	c.JSON(http.StatusOK, dto.RuntimeStatusResponse{
		Data: dto.RuntimeStatusData{
			ConfigVersion:    cfgVersion,
			ConfigHash:       cfgHash,
			LastReloadAt:     lastReloadAt,
			LastReloadError:  lastReloadError,
			Ports:            dto.RuntimePorts{HTTP: httpPort, Socks: socksPort},
			RuntimeMode:      mode,
			SingboxContainer: container,
		},
	})
}

func (h *Runtime) Plan(c *gin.Context) {
	// TODO: build config in memory, return nodes_included, tags, config_hash
	c.JSON(http.StatusOK, dto.RuntimePlanResponse{
		Data: dto.RuntimePlanData{
			NodesIncluded: 0,
			Tags:          []string{},
			ConfigHash:    "",
		},
	})
}

func (h *Runtime) Reload(c *gin.Context) {
	configPath := os.Getenv("SINGBOX_CONFIG")
	if configPath == "" {
		configPath = "/data/sing-box.json"
	}
	v, hsh, out, err := service.Reload(c.Request.Context(), h.DB, configPath)
	if err != nil {
		writeError(c, errorx.New(errorx.RTRestartFailed, err.Error()))
		return
	}
	c.JSON(http.StatusOK, dto.RuntimeReloadResponse{
		Data: dto.RuntimeReloadData{
			ConfigVersion: v,
			ConfigHash:    hsh,
			NodesIncluded: 0,
			RestartOutput: out,
			ReloadedAt:    util.NowRFC3339(),
		},
	})
}
