package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"boxpilot/server/internal/api/dto"
	"boxpilot/server/internal/store/repo"
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
