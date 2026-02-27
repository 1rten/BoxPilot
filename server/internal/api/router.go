package api

import (
	"database/sql"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"boxpilot/server/internal/api/handlers"
	"boxpilot/server/internal/api/middleware"

	"github.com/gin-gonic/gin"
)

// Router returns the HTTP router.
func Router(db *sql.DB) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recover(), middleware.RequestID(), middleware.CORS())

	sys := &handlers.System{}
	r.GET("/healthz", sys.Healthz)

	v1 := r.Group("/api/v1")
	{
		sub := &handlers.Subscriptions{DB: db}
		v1.GET("/subscriptions", sub.List)
		v1.POST("/subscriptions/create", sub.Create)
		v1.POST("/subscriptions/update", sub.Update)
		v1.POST("/subscriptions/delete", sub.Delete)
		v1.POST("/subscriptions/refresh", sub.Refresh)

		node := &handlers.Nodes{DB: db}
		v1.GET("/nodes", node.List)
		v1.POST("/nodes/update", node.Update)
		v1.POST("/nodes/forwarding/batch", node.BatchForwarding)
		v1.POST("/nodes/test", node.Test)
		v1.GET("/nodes/forwarding", node.Forwarding)
		v1.POST("/nodes/forwarding/update", node.UpdateForwarding)
		v1.POST("/nodes/forwarding/restart", node.RestartForwarding)

		rt := &handlers.Runtime{DB: db}
		v1.GET("/runtime/status", rt.Status)
		v1.POST("/runtime/plan", rt.Plan)
		v1.POST("/runtime/reload", rt.Reload)

		settings := &handlers.Settings{DB: db}
		v1.GET("/settings/proxy", settings.GetProxySettings)
		v1.POST("/settings/proxy/update", settings.UpdateProxySettings)
		v1.POST("/settings/proxy/apply", settings.ApplyProxySettings)
		v1.GET("/settings/routing", settings.GetRoutingSettings)
		v1.POST("/settings/routing/update", settings.UpdateRoutingSettings)
		v1.GET("/settings/forwarding/status", settings.ForwardingStatus)
		v1.GET("/settings/forwarding/summary", settings.ForwardingSummary)
		v1.POST("/settings/forwarding/start", settings.StartForwarding)
		v1.POST("/settings/forwarding/stop", settings.StopForwarding)
	}

	// Static files when WEB_ROOT is set (e.g. production)
	if dir := os.Getenv("WEB_ROOT"); dir != "" {
		indexPath := filepath.Join(dir, "index.html")
		r.NoRoute(func(c *gin.Context) {
			p := c.Request.URL.Path
			if strings.HasPrefix(p, "/api/") {
				c.String(http.StatusNotFound, "404 page not found")
				return
			}
			cleaned := strings.TrimPrefix(path.Clean("/"+p), "/")
			if cleaned == "" {
				c.File(indexPath)
				return
			}

			target := filepath.Join(dir, filepath.FromSlash(cleaned))
			if stat, err := os.Stat(target); err == nil && !stat.IsDir() {
				c.File(target)
				return
			}

			// Missing files with extension are likely asset requests; keep 404.
			if strings.Contains(filepath.Base(cleaned), ".") {
				c.String(http.StatusNotFound, "404 page not found")
				return
			}

			// SPA route fallback, e.g. /nodes or /subscriptions/123.
			c.File(indexPath)
		})
	} else {
		r.NoRoute(func(c *gin.Context) {
			c.String(http.StatusOK, "BoxPilot API. Set WEB_ROOT to serve frontend.")
		})
	}
	return r
}
