package handlers

import (
	"net/http"
	"database/sql"
	"os"
	"strconv"
	"github.com/gin-gonic/gin"
	"boxpilot/server/internal/api/dto"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util/errorx"
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
	if p := os.Getenv("HTTP_PROXY_PORT"); p != "" {
		if v, err := parseInt(p); err == nil {
			httpPort = v
		}
	}
	if p := os.Getenv("SOCKS_PROXY_PORT"); p != "" {
		if v, err := parseInt(p); err == nil {
			socksPort = v
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
			ConfigVersion:     cfgVersion,
			ConfigHash:        cfgHash,
			LastReloadAt:      lastReloadAt,
			LastReloadError:   lastReloadError,
			Ports:             dto.RuntimePorts{HTTP: httpPort, Socks: socksPort},
			RuntimeMode:       mode,
			SingboxContainer:  container,
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
	// TODO: config build + atomic write + docker restart
	c.JSON(http.StatusOK, dto.RuntimeReloadResponse{
		Data: dto.RuntimeReloadData{
			ConfigVersion: 0,
			ConfigHash:    "",
			NodesIncluded: 0,
			RestartOutput:  "",
			ReloadedAt:    "", // util.NowRFC3339() when implemented
		},
	})
}

func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}
