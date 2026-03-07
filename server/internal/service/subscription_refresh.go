package service

import (
	"database/sql"
	"io"
	"net/http"
	"strconv"
	"strings"
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
	parsed, err := parser.ParseSubscriptionBundle(body)
	if err != nil {
		repo.SetSubscriptionFetchResult(db, row.ID, row.Etag, row.LastModified, err.Error(), false)
		return false, 0, 0, err
	}
	subShort := row.ID
	if len(subShort) > 8 {
		subShort = subShort[:8]
	}
	nodes := make([]repo.NodeRow, 0, len(parsed.Outbounds))
	sourceToFinalTag := map[string]string{}
	for i, o := range parsed.Outbounds {
		tag := o.Tag
		if tag == "" {
			tag = subShort + "-" + strconv.Itoa(i) + "-node"
		}
		if strings.TrimSpace(o.Tag) != "" {
			sourceToFinalTag[strings.TrimSpace(o.Tag)] = tag
		}
		nodes = append(nodes, repo.NodeRow{
			ID: util.NewID(), SubID: row.ID, Tag: tag, Name: tag, Type: o.Type, Enabled: 1,
			OutboundJSON: string(o.Raw), CreatedAt: util.NowRFC3339(),
		})
	}
	if err := repo.ReplaceNodesForSubscription(db, row.ID, nodes); err != nil {
		return false, 0, 0, errorx.New(errorx.SUBReplaceNodesFailed, "replace nodes").WithDetails(map[string]any{"id": subID})
	}
	ruleSets := make([]repo.SubscriptionRuleSetRow, 0, len(parsed.RuleSets))
	for _, rs := range parsed.RuleSets {
		ruleSets = append(ruleSets, repo.SubscriptionRuleSetRow{
			ID:         util.NewID(),
			SubID:      row.ID,
			Tag:        rs.Tag,
			SourceType: rs.SourceType,
			Format:     rs.Format,
			URL:        rs.URL,
			Path:       rs.Path,
			CreatedAt:  util.NowRFC3339(),
		})
	}
	rules := make([]repo.SubscriptionRuleRow, 0, len(parsed.Rules))
	for _, rule := range parsed.Rules {
		rules = append(rules, repo.SubscriptionRuleRow{
			ID:             util.NewID(),
			SubID:          row.ID,
			SourceKind:     rule.SourceKind,
			Priority:       rule.Priority,
			RuleOrder:      rule.RuleOrder,
			MatcherType:    rule.MatcherType,
			MatcherValue:   rule.MatcherValue,
			TargetOutbound: rule.TargetOutbound,
			CreatedAt:      util.NowRFC3339(),
		})
	}
	availableTags := map[string]struct{}{}
	for _, node := range nodes {
		availableTags[node.Tag] = struct{}{}
	}
	groupMembers := make([]repo.SubscriptionGroupMemberRow, 0)
	memberDedup := map[string]struct{}{}
	for _, group := range parsed.BusinessGroups {
		target := strings.TrimSpace(group.TargetOutbound)
		if target == "" {
			continue
		}
		for _, rawTag := range group.NodeTags {
			tag := strings.TrimSpace(rawTag)
			if tag == "" {
				continue
			}
			if mapped, ok := sourceToFinalTag[tag]; ok {
				tag = mapped
			}
			if _, ok := availableTags[tag]; !ok {
				continue
			}
			key := target + "\x00" + tag
			if _, ok := memberDedup[key]; ok {
				continue
			}
			memberDedup[key] = struct{}{}
			groupMembers = append(groupMembers, repo.SubscriptionGroupMemberRow{
				ID:             util.NewID(),
				SubID:          row.ID,
				TargetOutbound: target,
				NodeTag:        tag,
				CreatedAt:      util.NowRFC3339(),
			})
		}
	}
	if err := repo.ReplaceSubscriptionRouting(db, row.ID, ruleSets, rules, groupMembers); err != nil {
		return false, 0, 0, errorx.New(errorx.DBError, "replace subscription routing").WithDetails(map[string]any{"id": subID})
	}
	etag := resp.Header.Get("Etag")
	lastMod := resp.Header.Get("Last-Modified")
	repo.SetSubscriptionFetchResult(db, row.ID, etag, lastMod, "", true)
	meta := parseSubscriptionUsageMeta(resp.Header)
	if err := repo.UpdateSubscriptionUsageMeta(db, row.ID, meta); err != nil {
		return false, 0, 0, errorx.New(errorx.DBError, "update subscription usage meta").WithDetails(map[string]any{
			"id":  subID,
			"err": err.Error(),
		})
	}
	return false, len(nodes), len(nodes), nil
}

func parseSubscriptionUsageMeta(headers http.Header) repo.SubscriptionUsageMeta {
	meta := repo.SubscriptionUsageMeta{}

	userinfo := strings.TrimSpace(headers.Get("subscription-userinfo"))
	if userinfo != "" {
		meta.UserinfoRaw = &userinfo
		for _, part := range strings.Split(userinfo, ";") {
			pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(pair) != 2 {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(pair[0]))
			value := strings.TrimSpace(pair[1])
			n, ok := parseNonNegativeInt64(value)
			if !ok {
				continue
			}
			switch key {
			case "upload":
				meta.UploadBytes = &n
			case "download":
				meta.DownloadBytes = &n
			case "total":
				meta.TotalBytes = &n
			case "expire":
				meta.ExpireUnix = &n
			}
		}
	}

	if page := strings.TrimSpace(headers.Get("profile-web-page")); page != "" {
		meta.ProfileWebPage = &page
	}

	if intervalRaw := strings.TrimSpace(headers.Get("profile-update-interval")); intervalRaw != "" {
		if interval, err := strconv.Atoi(intervalRaw); err == nil && interval > 0 {
			meta.ProfileUpdateSeconds = &interval
		}
	}

	if userinfo != "" || meta.ProfileWebPage != nil || meta.ProfileUpdateSeconds != nil {
		now := util.NowRFC3339()
		meta.UserinfoUpdatedAt = &now
	}

	return meta
}

func parseNonNegativeInt64(raw string) (int64, bool) {
	n, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}
