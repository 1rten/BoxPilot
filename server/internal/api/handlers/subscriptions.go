package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"boxpilot/server/internal/api/dto"
	"boxpilot/server/internal/service"
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
	autoUpdateEnabled := 0
	if req.AutoUpdateEnabled != nil && *req.AutoUpdateEnabled {
		autoUpdateEnabled = 1
	}
	if req.Name == "" {
		req.Name = req.URL
	}
	id := util.NewID()
	if err := repo.CreateSubscription(h.DB, id, req.Name, req.URL, req.Type, 1, autoUpdateEnabled, req.RefreshIntervalSec); err != nil {
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
		AutoUpdateEnabled: autoUpdateEnabled == 1, RefreshIntervalSec: req.RefreshIntervalSec,
		CreatedAt: util.NowRFC3339(), UpdatedAt: util.NowRFC3339(),
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
	before, err := repo.GetSubscription(h.DB, req.ID)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get subscription").WithDetails(map[string]any{"err": err.Error()}))
		return
	}
	if before == nil {
		writeError(c, errorx.New(errorx.SUBNotFound, "subscription not found").WithDetails(map[string]any{"id": req.ID}))
		return
	}
	var name *string
	var subURL *string
	var enabled *int
	var autoUpdateEnabled *int
	if req.Name != "" {
		name = &req.Name
	}
	if req.URL != "" {
		subURL = &req.URL
	}
	if req.Enabled != nil {
		v := 0
		if *req.Enabled {
			v = 1
		}
		enabled = &v
	}
	if req.AutoUpdateEnabled != nil {
		v := 0
		if *req.AutoUpdateEnabled {
			v = 1
		}
		autoUpdateEnabled = &v
	}
	var refresh *int
	if req.RefreshIntervalSec != nil && *req.RefreshIntervalSec > 0 {
		if *req.RefreshIntervalSec < 60 {
			writeError(c, errorx.New(errorx.REQInvalidField, "refresh_interval_sec must be >= 60"))
			return
		}
		refresh = req.RefreshIntervalSec
	}
	if err := repo.UpdateSubscription(h.DB, req.ID, name, subURL, enabled, autoUpdateEnabled, refresh); err != nil {
		writeError(c, errorx.New(errorx.DBError, "update subscription"))
		return
	}

	urlChanged := subURL != nil && *subURL != before.URL
	if urlChanged {
		if _, _, _, err := service.RefreshSubscription(h.DB, req.ID); err != nil {
			oldURL := before.URL
			_ = repo.UpdateSubscription(h.DB, req.ID, nil, &oldURL, nil, nil, nil)
			if appErr, ok := err.(*errorx.AppError); ok {
				writeError(c, appErr)
				return
			}
			writeError(c, errorx.New(errorx.SUBFetchFailed, "refresh after url update failed").WithDetails(map[string]any{
				"id":  req.ID,
				"err": err.Error(),
			}))
			return
		}
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
	notModified, total, enabled, err := service.RefreshSubscription(h.DB, req.ID)
	if err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.SUBFetchFailed, "subscription refresh failed").WithDetails(map[string]any{
			"id":  req.ID,
			"err": err.Error(),
		}))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"sub_id": req.ID, "not_modified": notModified, "nodes_total": total, "nodes_enabled": enabled,
		"fetched_at": util.NowRFC3339(),
	})
}

func subRowToDTO(r repo.SubscriptionRow) dto.Subscription {
	d := dto.Subscription{
		ID: r.ID, Name: r.Name, URL: r.URL, Type: r.Type, Enabled: r.Enabled == 1,
		AutoUpdateEnabled:  r.AutoUpdateEnabled == 1,
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
	var used int64
	var hasUsed bool
	if r.SubUploadBytes.Valid {
		used += r.SubUploadBytes.Int64
		hasUsed = true
	}
	if r.SubDownloadBytes.Valid {
		used += r.SubDownloadBytes.Int64
		hasUsed = true
	}
	if hasUsed {
		d.UsedBytes = &used
	}
	if r.SubTotalBytes.Valid {
		total := r.SubTotalBytes.Int64
		d.TotalBytes = &total
		if hasUsed {
			remaining := total - used
			if remaining < 0 {
				remaining = 0
			}
			d.RemainingBytes = &remaining
			if total > 0 {
				pct := float64(used) * 100 / float64(total)
				if pct < 0 {
					pct = 0
				}
				if pct > 100 {
					pct = 100
				}
				d.UsagePercent = &pct
			}
		}
	}
	if r.SubExpireUnix.Valid && r.SubExpireUnix.Int64 > 0 {
		exp := time.Unix(r.SubExpireUnix.Int64, 0).UTC().Format(time.RFC3339)
		d.ExpireAt = &exp
	}
	if r.SubProfileWebPage.Valid {
		d.ProfileWebPage = &r.SubProfileWebPage.String
	}
	if r.SubProfileInterval.Valid {
		n := int(r.SubProfileInterval.Int64)
		d.ProfileUpdateSec = &n
	}
	return d
}
