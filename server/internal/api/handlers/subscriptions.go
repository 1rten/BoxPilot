package handlers

import (
	"database/sql"
	"net/http"

	"boxpilot/server/internal/api/dto"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"

	"github.com/gin-gonic/gin"
)

type Subscriptions struct {
	DB *sql.DB
}

func (h *Subscriptions) List(c *gin.Context) {
	list, err := repo.ListSubscriptions(h.DB, false)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "list subscriptions").WithDetails(map[string]any{"err": err.Error()}))
		return
	}
	data := make([]dto.Subscription, 0, len(list))
	for _, r := range list {
		data = append(data, subRowToDTO(r))
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *Subscriptions) Create(c *gin.Context) {
	var req dto.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	if req.URL == "" {
		writeError(c, errorx.New(errorx.REQMissingField, "url required").WithDetails(map[string]any{"field": "url"}))
		return
	}
	if req.Type == "" {
		req.Type = "singbox"
	}
	if req.RefreshIntervalSec < 60 {
		req.RefreshIntervalSec = 3600
	}
	if req.Name == "" {
		req.Name = req.URL
	}
	id := util.NewID()
	if err := repo.CreateSubscription(h.DB, id, req.Name, req.URL, req.Type, 1, req.RefreshIntervalSec); err != nil {
		writeError(c, errorx.New(errorx.DBError, "create subscription"))
		return
	}
	row, _ := repo.GetSubscription(h.DB, id)
	if row != nil {
		c.JSON(http.StatusOK, gin.H{"data": subRowToDTO(*row)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": dto.Subscription{
		ID: id, Name: req.Name, URL: req.URL, Type: req.Type, Enabled: true,
		RefreshIntervalSec: req.RefreshIntervalSec, CreatedAt: util.NowRFC3339(), UpdatedAt: util.NowRFC3339(),
	}})
}

func (h *Subscriptions) Update(c *gin.Context) {
	var req dto.UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	if req.ID == "" {
		writeError(c, errorx.New(errorx.REQMissingField, "id required"))
		return
	}
	if err := repo.EnsureSubscriptionExists(h.DB, req.ID); err != nil {
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
	var refresh *int
	if req.RefreshIntervalSec != nil && *req.RefreshIntervalSec > 0 {
		refresh = req.RefreshIntervalSec
	}
	if err := repo.UpdateSubscription(h.DB, req.ID, name, enabled, refresh); err != nil {
		writeError(c, errorx.New(errorx.DBError, "update subscription"))
		return
	}
	row, _ := repo.GetSubscription(h.DB, req.ID)
	if row != nil {
		c.JSON(http.StatusOK, gin.H{"data": subRowToDTO(*row)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": nil})
}

func (h *Subscriptions) Delete(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" {
		writeError(c, errorx.New(errorx.REQMissingField, "id required"))
		return
	}
	ok, err := repo.DeleteSubscription(h.DB, req.ID)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "delete subscription"))
		return
	}
	if !ok {
		writeError(c, errorx.New(errorx.SUBNotFound, "subscription not found").WithDetails(map[string]any{"id": req.ID}))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Subscriptions) Refresh(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ID == "" {
		writeError(c, errorx.New(errorx.REQMissingField, "id required"))
		return
	}
	if err := repo.EnsureSubscriptionExists(h.DB, req.ID); err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.DBError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"sub_id": req.ID, "not_modified": false, "nodes_total": 0, "nodes_enabled": 0,
		"fetched_at": util.NowRFC3339(),
	})
}

func subRowToDTO(r repo.SubscriptionRow) dto.Subscription {
	d := dto.Subscription{
		ID: r.ID, Name: r.Name, URL: r.URL, Type: r.Type, Enabled: r.Enabled == 1,
		RefreshIntervalSec: r.RefreshIntervalSec, Etag: r.Etag, LastModified: r.LastModified,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
	if r.LastFetchAt.Valid {
		d.LastFetchAt = &r.LastFetchAt.String
	}
	if r.LastSuccessAt.Valid {
		d.LastSuccessAt = &r.LastSuccessAt.String
	}
	if r.LastError.Valid {
		d.LastError = &r.LastError.String
	}
	return d
}
