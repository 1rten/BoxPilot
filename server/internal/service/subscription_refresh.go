package service

import (
	"database/sql"
	"io"
	"net/http"
	"strconv"
	"time"

	"boxpilot/server/internal/parser"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"
)

// RefreshSubscription fetches one subscription URL, parses, and replaces nodes.
func RefreshSubscription(db *sql.DB, subID string) (notModified bool, nodesTotal, nodesEnabled int, err error) {
	row, err := repo.GetSubscription(db, subID)
	if err != nil || row == nil {
		return false, 0, 0, errorx.New(errorx.SUBNotFound, "subscription not found").WithDetails(map[string]any{"id": subID})
	}
	req, _ := http.NewRequest(http.MethodGet, row.URL, nil)
	if row.Etag != "" {
		req.Header.Set("If-None-Match", row.Etag)
	}
	if row.LastModified != "" {
		req.Header.Set("If-Modified-Since", row.LastModified)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		repo.SetSubscriptionFetchResult(db, row.ID, row.Etag, row.LastModified, err.Error(), false)
		return false, 0, 0, errorx.New(errorx.SUBFetchFailed, "fetch failed").WithDetails(map[string]any{"id": subID, "err": err.Error()})
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		repo.SetSubscriptionFetchResult(db, row.ID, row.Etag, row.LastModified, "", false)
		return true, 0, 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		repo.SetSubscriptionFetchResult(db, row.ID, row.Etag, row.LastModified, resp.Status, false)
		return false, 0, 0, errorx.New(errorx.SUBHTTPStatusError, "bad status").WithDetails(map[string]any{"id": subID, "status": resp.StatusCode})
	}
	maxSize := 5 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxSize)))
	if err != nil {
		return false, 0, 0, err
	}
	outbounds, err := parser.ParseSubscription(body)
	if err != nil {
		repo.SetSubscriptionFetchResult(db, row.ID, row.Etag, row.LastModified, err.Error(), false)
		return false, 0, 0, err
	}
	subShort := row.ID
	if len(subShort) > 8 {
		subShort = subShort[:8]
	}
	nodes := make([]repo.NodeRow, 0, len(outbounds))
	for i, o := range outbounds {
		tag := o.Tag
		if tag == "" {
			tag = subShort + "-" + strconv.Itoa(i) + "-node"
		}
		nodes = append(nodes, repo.NodeRow{
			ID: util.NewID(), SubID: row.ID, Tag: tag, Name: tag, Type: o.Type, Enabled: 1,
			OutboundJSON: string(o.Raw), CreatedAt: util.NowRFC3339(),
		})
	}
	if err := repo.ReplaceNodesForSubscription(db, row.ID, nodes); err != nil {
		return false, 0, 0, errorx.New(errorx.SUBReplaceNodesFailed, "replace nodes").WithDetails(map[string]any{"id": subID})
	}
	etag := resp.Header.Get("Etag")
	lastMod := resp.Header.Get("Last-Modified")
	repo.SetSubscriptionFetchResult(db, row.ID, etag, lastMod, "", true)
	return false, len(nodes), len(nodes), nil
}

