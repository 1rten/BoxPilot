package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// System handles /healthz etc.
type System struct{}

func (h *System) Healthz(c *gin.Context) {
	c.Data(http.StatusOK, "text/plain", []byte("ok"))
}
