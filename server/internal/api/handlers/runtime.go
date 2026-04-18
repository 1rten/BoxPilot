package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"boxpilot/server/internal/api/dto"
	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/service"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/proxy"
)

type Runtime struct {
	DB *sql.DB
}

type runtimeTrafficSnapshot struct {
	mu          sync.Mutex
	lastAt      time.Time
	rxTotal     uint64
	txTotal     uint64
	initialized bool
}

type proxyTrafficSample struct {
	source    string
	rxRateBps int64
	txRateBps int64
	rxTotal   uint64
	txTotal   uint64
	hasTotals bool
}

type clashProxyState struct {
	nowByTag map[string]string
}

var trafficSnapshot runtimeTrafficSnapshot

const (
	autoProbeTimeoutMS  = 5000
	autoProbeAttempts   = 8
	autoProbeInterval   = 350 * time.Millisecond
	manualProbeAttempts = 3
	manualProbeInterval = 250 * time.Millisecond
)

func (h *Runtime) Status(c *gin.Context) {
	row, err := repo.GetRuntimeState(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get runtime state"))
		return
	}
	cfgVersion := 0
	cfgHash := ""
	forwardingRunning := false
	nodesIncluded := 0
	lastApplyDuration := 0
	var lastReloadAt, lastReloadError, lastApplySuccess *string
	if row != nil {
		cfgVersion = row.ConfigVersion
		cfgHash = row.ConfigHash
		forwardingRunning = row.ForwardingRunning == 1
		nodesIncluded = row.LastNodesIncluded
		lastApplyDuration = row.LastApplyDuration
		if row.LastReloadAt.Valid {
			lastReloadAt = &row.LastReloadAt.String
		}
		if row.LastApplySuccess.Valid {
			lastApplySuccess = &row.LastApplySuccess.String
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
		if forwardingRunning && lastReloadError == nil {
			httpProxy, socksProxy := runtimeProxyRowsToInbounds(settings["http"], settings["socks"])
			if healthErr := service.ObserveRuntimeHealth(c.Request.Context(), httpProxy, socksProxy).ListenerError(); healthErr != nil {
				msg := healthErr.Error()
				lastReloadError = &msg
			}
		}
	} else if forwardingRunning && lastReloadError == nil {
		msg := "proxy settings unavailable; unable to verify runtime listeners"
		lastReloadError = &msg
	}
	c.JSON(http.StatusOK, dto.RuntimeStatusResponse{
		Data: dto.RuntimeStatusData{
			ConfigVersion:     cfgVersion,
			ConfigHash:        cfgHash,
			ForwardingRunning: forwardingRunning,
			NodesIncluded:     nodesIncluded,
			LastApplyDuration: lastApplyDuration,
			LastApplySuccess:  lastApplySuccess,
			LastReloadAt:      lastReloadAt,
			LastReloadError:   lastReloadError,
			Ports:             dto.RuntimePorts{HTTP: httpPort, Socks: socksPort},
		},
	})
}

func (h *Runtime) Traffic(c *gin.Context) {
	now := time.Now().UTC()
	sample, err := fetchProxyTraffic(c.Request.Context())
	source := sample.source
	if source == "" {
		source = "singbox_clash_api_unavailable"
	}

	rxRate := sample.rxRateBps
	txRate := sample.txRateBps
	if err != nil {
		rxRate = 0
		txRate = 0
	}

	trafficSnapshot.mu.Lock()
	rxTotal := trafficSnapshot.rxTotal
	txTotal := trafficSnapshot.txTotal
	if err == nil {
		switch {
		case sample.hasTotals:
			rxTotal = sample.rxTotal
			txTotal = sample.txTotal
		case trafficSnapshot.initialized:
			elapsed := now.Sub(trafficSnapshot.lastAt).Seconds()
			if elapsed > 0 {
				rxTotal += uint64(float64(rxRate) * elapsed)
				txTotal += uint64(float64(txRate) * elapsed)
			}
		}
		trafficSnapshot.initialized = true
		trafficSnapshot.lastAt = now
		trafficSnapshot.rxTotal = rxTotal
		trafficSnapshot.txTotal = txTotal
	}
	trafficSnapshot.mu.Unlock()

	c.JSON(http.StatusOK, dto.RuntimeTrafficResponse{
		Data: dto.RuntimeTrafficData{
			SampledAt:    now.Format(time.RFC3339),
			Source:       source,
			RXRateBps:    rxRate,
			TXRateBps:    txRate,
			RXTotalBytes: clampUint64ToInt64(rxTotal),
			TXTotalBytes: clampUint64ToInt64(txTotal),
		},
	})
}

func (h *Runtime) Connections(c *gin.Context) {
	nodes, err := repo.ListEnabledForwardingNodes(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "list runtime connections"))
		return
	}

	items := make([]dto.RuntimeConnection, 0, len(nodes))
	for _, n := range nodes {
		target := parseTargetFromOutbound(n.OutboundJSON)
		status := "selected"
		if n.LastTestStatus.Valid && n.LastTestStatus.String != "" {
			status = strings.ToLower(n.LastTestStatus.String)
		}

		var lastTestAt *string
		if n.LastTestAt.Valid {
			v := n.LastTestAt.String
			lastTestAt = &v
		}

		var latency *int64
		if n.LastLatencyMs.Valid {
			v := n.LastLatencyMs.Int64
			latency = &v
		}

		var errMsg *string
		if n.LastTestError.Valid && n.LastTestError.String != "" {
			v := n.LastTestError.String
			errMsg = &v
		}

		lastUpdated := n.CreatedAt
		if lastTestAt != nil {
			lastUpdated = *lastTestAt
		}

		items = append(items, dto.RuntimeConnection{
			ID:          n.ID,
			NodeID:      n.ID,
			NodeName:    fallbackNodeName(n.Name, n.Tag),
			NodeType:    n.Type,
			Target:      target,
			Status:      status,
			LastTestAt:  lastTestAt,
			LatencyMs:   latency,
			Error:       errMsg,
			Forwarding:  n.ForwardingEnabled == 1,
			LastUpdated: lastUpdated,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].LastUpdated > items[j].LastUpdated
	})

	q := strings.ToLower(strings.TrimSpace(c.Query("q")))
	if q != "" {
		filtered := make([]dto.RuntimeConnection, 0, len(items))
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.NodeName), q) ||
				strings.Contains(strings.ToLower(item.NodeType), q) ||
				strings.Contains(strings.ToLower(item.Target), q) ||
				strings.Contains(strings.ToLower(item.Status), q) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	c.JSON(http.StatusOK, dto.RuntimeConnectionsResponse{
		Data: dto.RuntimeConnectionsData{
			ActiveCount: len(items),
			Items:       items,
		},
	})
}

func (h *Runtime) Logs(c *gin.Context) {
	level := strings.ToLower(strings.TrimSpace(c.DefaultQuery("level", "all")))
	keyword := strings.ToLower(strings.TrimSpace(c.Query("q")))
	limit := parseLimit(c.DefaultQuery("limit", "80"), 80, 1, 500)

	items := make([]dto.RuntimeLogItem, 0, 128)
	now := time.Now().UTC().Format(time.RFC3339)

	row, err := repo.GetRuntimeState(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get runtime state"))
		return
	}
	if row != nil {
		if row.LastReloadAt.Valid {
			items = append(items, dto.RuntimeLogItem{
				Timestamp: row.LastReloadAt.String,
				Level:     "info",
				Source:    "runtime",
				Message:   "runtime config applied",
			})
		}
		if row.ForwardingRunning == 1 {
			items = append(items, dto.RuntimeLogItem{
				Timestamp: now,
				Level:     "info",
				Source:    "runtime",
				Message:   "forwarding runtime running",
			})
		} else {
			items = append(items, dto.RuntimeLogItem{
				Timestamp: now,
				Level:     "warn",
				Source:    "runtime",
				Message:   "forwarding runtime stopped",
			})
		}
		if row.LastReloadError.Valid && strings.TrimSpace(row.LastReloadError.String) != "" {
			items = append(items, dto.RuntimeLogItem{
				Timestamp: fallbackTimestamp(row.LastReloadAt, now),
				Level:     "error",
				Source:    "runtime",
				Message:   row.LastReloadError.String,
			})
		}
	}

	nodes, err := repo.ListEnabledForwardingNodes(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "list nodes for logs"))
		return
	}
	for _, n := range nodes {
		timestamp := n.CreatedAt
		if n.LastTestAt.Valid {
			timestamp = n.LastTestAt.String
		}
		name := fallbackNodeName(n.Name, n.Tag)
		if n.LastTestError.Valid && strings.TrimSpace(n.LastTestError.String) != "" {
			items = append(items, dto.RuntimeLogItem{
				Timestamp: timestamp,
				Level:     "error",
				Source:    "probe",
				Message:   name + " probe failed: " + n.LastTestError.String,
			})
			continue
		}
		if n.LastTestStatus.Valid && n.LastTestStatus.String != "" {
			testLevel := "info"
			status := strings.ToLower(n.LastTestStatus.String)
			if status != "ok" {
				testLevel = "warn"
			}
			msg := name + " probe status: " + strings.ToUpper(status)
			if n.LastLatencyMs.Valid {
				msg += " (" + strconv.FormatInt(n.LastLatencyMs.Int64, 10) + "ms)"
			}
			items = append(items, dto.RuntimeLogItem{
				Timestamp: timestamp,
				Level:     testLevel,
				Source:    "probe",
				Message:   msg,
			})
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Timestamp > items[j].Timestamp
	})

	filtered := make([]dto.RuntimeLogItem, 0, len(items))
	for _, item := range items {
		if level != "all" && item.Level != level {
			continue
		}
		if keyword != "" {
			hay := strings.ToLower(item.Source + " " + item.Message)
			if !strings.Contains(hay, keyword) {
				continue
			}
		}
		filtered = append(filtered, item)
		if len(filtered) >= limit {
			break
		}
	}

	c.JSON(http.StatusOK, dto.RuntimeLogsResponse{
		Data: dto.RuntimeLogsData{
			Items: filtered,
		},
	})
}

func (h *Runtime) Plan(c *gin.Context) {
	var req dto.RuntimePlanRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
			return
		}
	}

	cfg, tags, err := h.buildRuntimeConfig(req.IncludeDisabledNodes, true, true)
	if err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.CFGBuildFailed, "build plan config"))
		return
	}

	c.JSON(http.StatusOK, dto.RuntimePlanResponse{
		Data: dto.RuntimePlanData{
			NodesIncluded: len(tags),
			Tags:          tags,
			ConfigHash:    util.JSONHash(cfg),
		},
	})
}

func (h *Runtime) Groups(c *gin.Context) {
	cfg, _, err := h.buildRuntimeConfig(false, false, false)
	if err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.CFGBuildFailed, "build runtime groups"))
		return
	}
	selectionRows, err := repo.ListRuntimeGroupSelections(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "list runtime group selections"))
		return
	}
	selectionByTag := map[string]repo.RuntimeGroupSelectionRow{}
	for _, row := range selectionRows {
		selectionByTag[row.GroupTag] = row
	}
	clashState, _ := fetchClashProxyState(c.Request.Context())
	groups, err := parseSelectorGroups(cfg, selectionByTag, clashState)
	if err != nil {
		writeError(c, errorx.New(errorx.CFGJSONInvalid, "parse runtime groups"))
		return
	}
	c.JSON(http.StatusOK, dto.RuntimeGroupSummaryResponse{
		Data: dto.RuntimeGroupSummaryData{
			Items: groups,
		},
	})
}

func (h *Runtime) SelectGroup(c *gin.Context) {
	groupTag := strings.TrimSpace(c.Param("tag"))
	if groupTag == "" {
		writeError(c, errorx.New(errorx.REQInvalidField, "missing group tag"))
		return
	}
	var req dto.RuntimeGroupSelectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	selected := strings.TrimSpace(req.SelectedOutbound)
	if selected == "" {
		writeError(c, errorx.New(errorx.REQMissingField, "selected_outbound required"))
		return
	}

	cfg, _, err := h.buildRuntimeConfig(false, false, false)
	if err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.CFGBuildFailed, "build runtime groups"))
		return
	}
	groups, err := parseSelectorGroups(cfg, nil, nil)
	if err != nil {
		writeError(c, errorx.New(errorx.CFGJSONInvalid, "parse runtime groups"))
		return
	}
	group := findGroup(groups, groupTag)
	if group == nil {
		writeError(c, errorx.New(errorx.REQInvalidField, "group not found").WithDetails(map[string]any{"group_tag": groupTag}))
		return
	}
	if !containsString(group.Outbounds, selected) {
		writeError(c, errorx.New(errorx.REQInvalidField, "selected outbound not allowed").WithDetails(map[string]any{
			"group_tag":         groupTag,
			"selected_outbound": selected,
		}))
		return
	}
	selectedIsAuto := false
	if group.AutoOutbound != nil && strings.TrimSpace(*group.AutoOutbound) == selected {
		selectedIsAuto = true
	}

	prevSelection, hadPrevSelection, err := repo.GetRuntimeGroupSelection(h.DB, groupTag)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "load previous runtime group selection"))
		return
	}

	now := util.NowRFC3339()
	if err := repo.UpsertRuntimeGroupSelection(h.DB, groupTag, selected, now); err != nil {
		writeError(c, errorx.New(errorx.DBError, "save runtime group selection"))
		return
	}
	configPath := service.ResolveConfigPath()
	version, hash, _, reloadErr := service.Reload(c.Request.Context(), h.DB, configPath)
	if reloadErr != nil {
		rollbackErr := rollbackRuntimeGroupSelection(h.DB, groupTag, prevSelection, hadPrevSelection)
		details := map[string]any{
			"group_tag":         groupTag,
			"selected_outbound": selected,
		}
		if rollbackErr != nil {
			details["rollback_err"] = rollbackErr.Error()
		}
		if appErr, ok := reloadErr.(*errorx.AppError); ok {
			if len(appErr.Details) > 0 {
				details["reload_details"] = appErr.Details
			}
			writeError(c, errorx.New(appErr.Code, appErr.Message).WithDetails(details))
			return
		}
		writeError(c, errorx.New(errorx.RTRestartFailed, reloadErr.Error()).WithDetails(details))
		return
	}
	policy, policyErr := service.LoadForwardingPolicy(h.DB)
	probeTimeout := autoProbeTimeoutMS
	if policyErr == nil && policy.NodeTestTimeoutMs > 0 {
		probeTimeout = policy.NodeTestTimeoutMs
	}
	runtimeSelected, runtimeEffective, autoProbeError := resolveRuntimeSelectionAfterGroupSelect(
		c.Request.Context(),
		group,
		selected,
		probeTimeout,
	)

	c.JSON(http.StatusOK, dto.RuntimeGroupSelectResponse{
		Data: dto.RuntimeGroupSelectData{
			GroupTag:                 groupTag,
			SelectedOutbound:         selected,
			SelectedIsAuto:           selectedIsAuto,
			UpdatedAt:                now,
			ConfigVersion:            version,
			ConfigHash:               hash,
			RuntimeSelectedOutbound:  optionalString(runtimeSelected),
			RuntimeEffectiveOutbound: optionalString(runtimeEffective),
			AutoProbeError:           optionalString(autoProbeError),
		},
	})
}

func (h *Runtime) Reload(c *gin.Context) {
	configPath := service.ResolveConfigPath()
	v, hsh, out, err := service.Reload(c.Request.Context(), h.DB, configPath)
	if err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.RTRestartFailed, err.Error()))
		return
	}
	nodesIncluded := 0
	if nodes, listErr := repo.ListEnabledForwardingNodes(h.DB); listErr == nil {
		nodesIncluded = len(nodes)
	}
	c.JSON(http.StatusOK, dto.RuntimeReloadResponse{
		Data: dto.RuntimeReloadData{
			ConfigVersion: v,
			ConfigHash:    hsh,
			NodesIncluded: nodesIncluded,
			RestartOutput: out,
			ReloadedAt:    util.NowRFC3339(),
		},
	})
}

func (h *Runtime) ProxyCheck(c *gin.Context) {
	var req dto.RuntimeProxyCheckRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
			return
		}
	}

	targetURL := strings.TrimSpace(req.TargetURL)
	if targetURL == "" {
		targetURL = "https://www.gstatic.com/generate_204"
	}
	parsedTarget, err := url.Parse(targetURL)
	if err != nil || parsedTarget.Scheme == "" || parsedTarget.Host == "" {
		writeError(c, errorx.New(errorx.REQInvalidField, "invalid target_url"))
		return
	}

	timeoutMS := req.TimeoutMS
	if timeoutMS == 0 {
		timeoutMS = 5000
	}
	if timeoutMS < 500 || timeoutMS > 30000 {
		writeError(c, errorx.New(errorx.REQInvalidField, "timeout_ms must be between 500 and 30000"))
		return
	}

	settings, err := repo.GetProxySettings(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get proxy settings"))
		return
	}

	timeout := time.Duration(timeoutMS) * time.Millisecond
	httpRow := settings["http"]
	socksRow := settings["socks"]
	httpResult := probeProxyEndpoint(parsedTarget, "http", httpRow, timeout)
	socksResult := probeProxyEndpoint(parsedTarget, "socks", socksRow, timeout)

	c.JSON(http.StatusOK, dto.RuntimeProxyCheckResponse{
		Data: dto.RuntimeProxyCheckData{
			TargetURL: targetURL,
			CheckedAt: util.NowRFC3339(),
			HTTP:      httpResult,
			Socks:     socksResult,
		},
	})
}

func runtimeProxyRowsToInbounds(httpRow, socksRow repo.ProxySettingsRow) (generator.ProxyInbound, generator.ProxyInbound) {
	httpProxy := generator.ProxyInbound{
		Type:          "http",
		ListenAddress: httpRow.ListenAddress,
		Port:          httpRow.Port,
		Enabled:       httpRow.Enabled == 1,
		AuthMode:      httpRow.AuthMode,
		Username:      httpRow.Username,
		Password:      httpRow.Password,
	}
	socksProxy := generator.ProxyInbound{
		Type:          "socks",
		ListenAddress: socksRow.ListenAddress,
		Port:          socksRow.Port,
		Enabled:       socksRow.Enabled == 1,
		AuthMode:      socksRow.AuthMode,
		Username:      socksRow.Username,
		Password:      socksRow.Password,
	}
	if httpProxy.ListenAddress == "" {
		httpProxy.ListenAddress = "0.0.0.0"
	}
	if httpProxy.Port == 0 {
		httpProxy.Port = 7890
	}
	if socksProxy.ListenAddress == "" {
		socksProxy.ListenAddress = "0.0.0.0"
	}
	if socksProxy.Port == 0 {
		socksProxy.Port = 7891
	}
	return httpProxy, socksProxy
}

func (h *Runtime) buildRuntimeConfig(includeDisabledNodes bool, applyForwardingPolicy bool, requireForwardingNodes bool) ([]byte, []string, error) {
	settings, err := repo.GetProxySettings(h.DB)
	if err != nil {
		return nil, nil, errorx.New(errorx.DBError, "get proxy settings")
	}
	httpProxy, socksProxy := runtimeProxyRowsToInbounds(settings["http"], settings["socks"])

	row, err := repo.GetRuntimeState(h.DB)
	if err != nil {
		return nil, nil, errorx.New(errorx.DBError, "get runtime state")
	}
	forwardingRunning := row != nil && row.ForwardingRunning == 1
	if !forwardingRunning {
		httpProxy.Enabled = false
		socksProxy.Enabled = false
	}

	nodes := []repo.NodeRow{}
	if includeDisabledNodes {
		nodes, err = repo.ListNodes(h.DB, "", nil)
	} else {
		nodes, err = repo.ListEnabledForwardingNodes(h.DB)
	}
	if err != nil {
		return nil, nil, errorx.New(errorx.DBError, "list nodes for runtime config")
	}
	policy, policyErr := service.LoadForwardingPolicy(h.DB)
	if policyErr != nil {
		return nil, nil, errorx.New(errorx.DBError, "get forwarding policy")
	}
	if !includeDisabledNodes && applyForwardingPolicy {
		nodes = service.FilterForwardingNodes(nodes, policy)
	}

	if requireForwardingNodes && forwardingRunning && (httpProxy.Enabled || socksProxy.Enabled) && len(nodes) == 0 {
		return nil, nil, errorx.New(errorx.CFGNoEnabledNodes, "no forwarding nodes enabled")
	}

	routing, _, err := service.LoadRoutingSettings(h.DB)
	if err != nil {
		return nil, nil, errorx.New(errorx.DBError, "get routing settings")
	}

	outbounds := make([]generator.NodeOutbound, 0, len(nodes))
	tags := make([]string, 0, len(nodes))
	for _, node := range nodes {
		outbounds = append(outbounds, generator.NodeOutbound{
			Tag:     node.Tag,
			RawJSON: node.OutboundJSON,
		})
		tags = append(tags, node.Tag)
	}

	ruleSetRows, err := repo.ListEnabledSubscriptionRuleSets(h.DB)
	if err != nil {
		return nil, nil, errorx.New(errorx.DBError, "list subscription rule sets")
	}
	ruleRows, err := repo.ListEnabledSubscriptionRules(h.DB)
	if err != nil {
		return nil, nil, errorx.New(errorx.DBError, "list subscription rules")
	}
	groupMemberRows, err := repo.ListEnabledSubscriptionGroupMembers(h.DB)
	if err != nil {
		return nil, nil, errorx.New(errorx.DBError, "list subscription group members")
	}
	selectionRows, err := repo.ListRuntimeGroupSelections(h.DB)
	if err != nil {
		return nil, nil, errorx.New(errorx.DBError, "list runtime group selections")
	}

	extras := generator.RoutingExtras{
		RuleSets:          make([]generator.RouteRuleSetRef, 0, len(ruleSetRows)),
		Rules:             make([]generator.RouteRule, 0, len(ruleRows)),
		GroupSelections:   map[string]string{},
		BusinessNodePools: map[string][]string{},
		AutoTestURL:       generator.DefaultAutoTestURL,
		AutoTestInterval:  service.BizAutoIntervalDuration(policy.BizAutoIntervalSec),
	}
	for _, rs := range ruleSetRows {
		extras.RuleSets = append(extras.RuleSets, generator.RouteRuleSetRef{
			Tag:        rs.Tag,
			SourceType: rs.SourceType,
			Format:     rs.Format,
			URL:        rs.URL,
			Path:       rs.Path,
		})
	}
	for _, r := range ruleRows {
		extras.Rules = append(extras.Rules, generator.RouteRule{
			Priority:       r.Priority,
			RuleOrder:      r.RuleOrder,
			MatcherType:    r.MatcherType,
			MatcherValue:   r.MatcherValue,
			TargetOutbound: r.TargetOutbound,
		})
	}
	for _, g := range groupMemberRows {
		target := strings.TrimSpace(g.TargetOutbound)
		tag := strings.TrimSpace(g.NodeTag)
		if target == "" || tag == "" {
			continue
		}
		extras.BusinessNodePools[target] = append(extras.BusinessNodePools[target], tag)
	}
	for _, s := range selectionRows {
		extras.GroupSelections[s.GroupTag] = s.SelectedOutbound
	}

	cfg, err := generator.BuildConfigWithRuntime(httpProxy, socksProxy, routing, outbounds, extras)
	if err != nil {
		return nil, nil, errorx.New(errorx.CFGBuildFailed, "build runtime config")
	}
	return cfg, tags, nil
}

func parseSelectorGroups(cfg []byte, persisted map[string]repo.RuntimeGroupSelectionRow, clashState *clashProxyState) ([]dto.RuntimeGroupItem, error) {
	var parsed struct {
		Outbounds []map[string]any `json:"outbounds"`
	}
	if err := json.Unmarshal(cfg, &parsed); err != nil {
		return nil, err
	}
	urltestMembers := map[string][]string{}
	for _, outbound := range parsed.Outbounds {
		typ := strings.TrimSpace(fmt.Sprintf("%v", outbound["type"]))
		if typ != "urltest" {
			continue
		}
		tag := strings.TrimSpace(fmt.Sprintf("%v", outbound["tag"]))
		if tag == "" {
			continue
		}
		memberAny, _ := outbound["outbounds"].([]any)
		members := make([]string, 0, len(memberAny))
		for _, m := range memberAny {
			member := strings.TrimSpace(fmt.Sprintf("%v", m))
			if member != "" {
				members = append(members, member)
			}
		}
		if len(members) > 0 {
			urltestMembers[tag] = members
		}
	}

	items := make([]dto.RuntimeGroupItem, 0, len(parsed.Outbounds))
	for _, outbound := range parsed.Outbounds {
		typ := strings.TrimSpace(fmt.Sprintf("%v", outbound["type"]))
		if typ != "selector" {
			continue
		}
		tag := strings.TrimSpace(fmt.Sprintf("%v", outbound["tag"]))
		if tag == "" {
			continue
		}
		defaultOutbound := strings.TrimSpace(fmt.Sprintf("%v", outbound["default"]))
		memberAny, _ := outbound["outbounds"].([]any)
		members := make([]string, 0, len(memberAny))
		for _, m := range memberAny {
			member := strings.TrimSpace(fmt.Sprintf("%v", m))
			if member != "" {
				members = append(members, member)
			}
		}
		item := dto.RuntimeGroupItem{
			Tag:       tag,
			Type:      typ,
			Outbounds: members,
			Default:   defaultOutbound,
		}
		nodeCandidates := make([]string, 0, len(members))
		autoTag := ""
		autoCandidates := make([]string, 0)
		autoSeen := map[string]struct{}{}
		for _, memberTag := range members {
			candidates, ok := urltestMembers[memberTag]
			if ok {
				if autoTag == "" {
					autoTag = memberTag
				}
				for _, nodeTag := range candidates {
					if _, exists := autoSeen[nodeTag]; exists {
						continue
					}
					autoSeen[nodeTag] = struct{}{}
					autoCandidates = append(autoCandidates, nodeTag)
				}
				continue
			}
			nodeCandidates = append(nodeCandidates, memberTag)
		}
		if autoTag != "" {
			item.AutoOutbound = &autoTag
		}
		if len(nodeCandidates) > 0 {
			item.NodeCandidates = nodeCandidates
		}
		if len(autoCandidates) > 0 {
			item.AutoCandidates = autoCandidates
		}
		if clashState != nil && clashState.nowByTag != nil {
			selected := strings.TrimSpace(clashState.nowByTag[tag])
			if selected != "" {
				item.RuntimeSelectedOutbound = &selected
				if resolved := resolveEffectiveOutbound(selected, clashState.nowByTag); resolved != "" {
					item.RuntimeEffectiveOutbound = &resolved
				}
			}
		}
		if persisted != nil {
			if row, ok := persisted[tag]; ok {
				selected := strings.TrimSpace(row.SelectedOutbound)
				if selected != "" {
					item.PersistedSelectedOutbound = &selected
				}
				updatedAt := strings.TrimSpace(row.UpdatedAt)
				if updatedAt != "" {
					item.PersistedUpdatedAt = &updatedAt
				}
			}
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Tag < items[j].Tag })
	return items, nil
}

func findGroup(items []dto.RuntimeGroupItem, tag string) *dto.RuntimeGroupItem {
	for i := range items {
		if items[i].Tag == tag {
			return &items[i]
		}
	}
	return nil
}

func containsString(items []string, candidate string) bool {
	for _, v := range items {
		if v == candidate {
			return true
		}
	}
	return false
}

func probeProxyEndpoint(target *url.URL, proxyType string, row repo.ProxySettingsRow, timeout time.Duration) dto.RuntimeProxyCheckItem {
	result := dto.RuntimeProxyCheckItem{
		Enabled:         row.Enabled == 1,
		ProxyURL:        proxyURL(proxyType, row.Port),
		ProxyReachable:  false,
		TargetReachable: false,
	}
	if !result.Enabled {
		return result
	}

	start := time.Now()
	client, closeFn, err := buildProxyHTTPClient(proxyType, row.Port, timeout)
	if err != nil {
		msg := err.Error()
		result.Error = &msg
		return result
	}
	result.ProxyReachable = true
	defer closeFn()

	req, err := http.NewRequest(http.MethodGet, target.String(), nil)
	if err != nil {
		msg := err.Error()
		result.Error = &msg
		return result
	}
	resp, err := client.Do(req)
	latency := time.Since(start).Milliseconds()
	result.LatencyMS = &latency
	if err != nil {
		msg := err.Error()
		result.Error = &msg
		return result
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))

	code := resp.StatusCode
	result.StatusCode = &code
	result.Connected = true
	result.TargetReachable = true
	if target.Scheme == "https" {
		result.TLSOK = resp.TLS != nil
	} else {
		result.TLSOK = true
	}
	return result
}

func buildProxyHTTPClient(proxyType string, port int, timeout time.Duration) (*http.Client, func(), error) {
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	transport := &http.Transport{
		ForceAttemptHTTP2: true,
	}

	switch proxyType {
	case "http":
		u, err := url.Parse("http://" + address)
		if err != nil {
			return nil, func() {}, err
		}
		transport.Proxy = http.ProxyURL(u)
	case "socks":
		dialer, err := proxy.SOCKS5("tcp", address, nil, &net.Dialer{Timeout: timeout})
		if err != nil {
			return nil, func() {}, err
		}
		if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
			transport.DialContext = contextDialer.DialContext
		} else {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		}
	default:
		return nil, func() {}, fmt.Errorf("unsupported proxy type: %s", proxyType)
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	return client, transport.CloseIdleConnections, nil
}

func proxyURL(proxyType string, port int) string {
	scheme := "http"
	if proxyType == "socks" {
		scheme = "socks5"
	}
	return scheme + "://127.0.0.1:" + strconv.Itoa(port)
}

func fetchProxyTraffic(parent context.Context) (proxyTrafficSample, error) {
	baseURL, enabled := resolveClashAPIBaseURL()
	if !enabled {
		return proxyTrafficSample{source: "singbox_clash_api_disabled"}, nil
	}

	sample := proxyTrafficSample{source: "singbox_clash_api_unavailable"}
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/traffic", nil)
	if err != nil {
		return sample, err
	}
	if secret := strings.TrimSpace(os.Getenv("SINGBOX_CLASH_API_SECRET")); secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return sample, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return sample, fmt.Errorf("clash api /traffic status %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	payload := map[string]any{}
	if err := decoder.Decode(&payload); err != nil {
		return sample, err
	}

	downRate, downOK := pickInt64FromMap(payload, "down", "download")
	upRate, upOK := pickInt64FromMap(payload, "up", "upload")
	if !downOK && !upOK {
		return sample, fmt.Errorf("clash api /traffic missing rate fields")
	}
	if downRate < 0 {
		downRate = 0
	}
	if upRate < 0 {
		upRate = 0
	}

	sample.source = "singbox_clash_api"
	sample.rxRateBps = downRate
	sample.txRateBps = upRate

	downTotal, downTotalOK := pickUint64FromMap(payload,
		"down_total", "download_total", "total_download", "downTotal", "downloadTotal")
	upTotal, upTotalOK := pickUint64FromMap(payload,
		"up_total", "upload_total", "total_upload", "upTotal", "uploadTotal")
	if downTotalOK && upTotalOK {
		sample.hasTotals = true
		sample.rxTotal = downTotal
		sample.txTotal = upTotal
	}

	return sample, nil
}

func fetchClashProxyState(parent context.Context) (*clashProxyState, error) {
	baseURL, enabled := resolveClashAPIBaseURL()
	if !enabled {
		return nil, fmt.Errorf("clash api disabled")
	}
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/proxies", nil)
	if err != nil {
		return nil, err
	}
	if secret := strings.TrimSpace(os.Getenv("SINGBOX_CLASH_API_SECRET")); secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("clash api /proxies status %d", resp.StatusCode)
	}
	var payload struct {
		Proxies map[string]map[string]any `json:"proxies"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	nowByTag := map[string]string{}
	for tag, data := range payload.Proxies {
		if data == nil {
			continue
		}
		now := ""
		if raw, ok := data["now"]; ok {
			switch v := raw.(type) {
			case string:
				now = strings.TrimSpace(v)
			default:
				now = strings.TrimSpace(fmt.Sprintf("%v", raw))
			}
		}
		if now != "" {
			nowByTag[strings.TrimSpace(tag)] = now
		}
	}
	return &clashProxyState{nowByTag: nowByTag}, nil
}

func resolveRuntimeSelectionAfterGroupSelect(parent context.Context, group *dto.RuntimeGroupItem, selected string, probeTimeoutMS int) (string, string, string) {
	if group == nil {
		return "", "", ""
	}
	selected = strings.TrimSpace(selected)
	groupTag := strings.TrimSpace(group.Tag)
	if selected == "" || groupTag == "" {
		return "", "", ""
	}

	autoSelected := false
	probeError := ""
	autoCandidateSet := map[string]struct{}{}
	if group.AutoOutbound != nil {
		autoTag := strings.TrimSpace(*group.AutoOutbound)
		if autoTag != "" && autoTag == selected {
			autoSelected = true
			if err := triggerClashProxyDelayTest(parent, autoTag, generator.DefaultAutoTestURL, probeTimeoutMS); err != nil {
				probeError = summarizeAutoProbeError(err)
			}
			for _, candidate := range group.AutoCandidates {
				tag := strings.TrimSpace(candidate)
				if tag != "" {
					autoCandidateSet[tag] = struct{}{}
				}
			}
		}
	}

	attempts := manualProbeAttempts
	interval := manualProbeInterval
	if autoSelected {
		attempts = autoProbeAttempts
		interval = autoProbeInterval
	}

	lastSelected := ""
	lastEffective := ""
	for i := 0; i < attempts; i++ {
		state, err := fetchClashProxyState(parent)
		if err == nil {
			currentSelected, currentEffective := runtimeSelectionFromClashState(groupTag, state)
			if currentSelected != "" {
				lastSelected = currentSelected
				lastEffective = currentEffective
			}
			if currentSelected == selected {
				if !autoSelected {
					return currentSelected, currentEffective, ""
				}
				if currentEffective != "" {
					if len(autoCandidateSet) == 0 {
						return currentSelected, currentEffective, probeError
					}
					if _, ok := autoCandidateSet[currentEffective]; ok {
						return currentSelected, currentEffective, probeError
					}
				}
			}
		}
		if i+1 >= attempts {
			break
		}
		timer := time.NewTimer(interval)
		select {
		case <-parent.Done():
			timer.Stop()
			return lastSelected, lastEffective, probeError
		case <-timer.C:
		}
	}
	return lastSelected, lastEffective, probeError
}

func rollbackRuntimeGroupSelection(
	db *sql.DB,
	groupTag string,
	prev repo.RuntimeGroupSelectionRow,
	hadPrev bool,
) error {
	if hadPrev {
		return repo.UpsertRuntimeGroupSelection(db, prev.GroupTag, prev.SelectedOutbound, prev.UpdatedAt)
	}
	return repo.DeleteRuntimeGroupSelection(db, groupTag)
}

func triggerClashProxyDelayTest(parent context.Context, proxyTag, targetURL string, timeoutMS int) error {
	proxyTag = strings.TrimSpace(proxyTag)
	if proxyTag == "" {
		return fmt.Errorf("empty proxy tag")
	}
	baseURL, enabled := resolveClashAPIBaseURL()
	if !enabled {
		return fmt.Errorf("clash api disabled")
	}
	if timeoutMS <= 0 {
		timeoutMS = autoProbeTimeoutMS
	}
	targetURL = strings.TrimSpace(targetURL)
	if targetURL == "" {
		targetURL = generator.DefaultAutoTestURL
	}
	requestURL := fmt.Sprintf(
		"%s/proxies/%s/delay?url=%s&timeout=%d",
		baseURL,
		url.PathEscape(proxyTag),
		url.QueryEscape(targetURL),
		timeoutMS,
	)
	ctx, cancel := context.WithTimeout(parent, time.Duration(timeoutMS+2000)*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return err
	}
	if secret := strings.TrimSpace(os.Getenv("SINGBOX_CLASH_API_SECRET")); secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("clash api delay status %d", resp.StatusCode)
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	return nil
}

func runtimeSelectionFromClashState(groupTag string, state *clashProxyState) (string, string) {
	if state == nil || state.nowByTag == nil {
		return "", ""
	}
	selected := strings.TrimSpace(state.nowByTag[groupTag])
	if selected == "" {
		return "", ""
	}
	effective := resolveEffectiveOutbound(selected, state.nowByTag)
	return selected, effective
}

func resolveEffectiveOutbound(outbound string, nowByTag map[string]string) string {
	current := strings.TrimSpace(outbound)
	if current == "" || nowByTag == nil {
		return current
	}
	visited := map[string]struct{}{}
	for {
		if _, ok := visited[current]; ok {
			return current
		}
		visited[current] = struct{}{}
		next := strings.TrimSpace(nowByTag[current])
		if next == "" {
			return current
		}
		current = next
	}
}

func optionalString(value string) *string {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	return &v
}

func summarizeAutoProbeError(err error) string {
	if err == nil {
		return ""
	}
	raw := strings.TrimSpace(err.Error())
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "clash api disabled"),
		strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "i/o timeout"),
		strings.Contains(lower, "context deadline exceeded"),
		strings.Contains(lower, "no such host"):
		return "clash api unavailable"
	case strings.Contains(lower, "status 401"),
		strings.Contains(lower, "status 403"):
		return "clash api unauthorized"
	}
	if raw == "" {
		return "auto probe failed"
	}
	return raw
}

func resolveClashAPIBaseURL() (string, bool) {
	controller := strings.TrimSpace(os.Getenv("SINGBOX_CLASH_API_ADDR"))
	if controller == "" {
		controller = "127.0.0.1:9090"
	}
	if strings.EqualFold(controller, "off") {
		return "", false
	}
	if !strings.HasPrefix(controller, "http://") && !strings.HasPrefix(controller, "https://") {
		controller = "http://" + controller
	}
	return strings.TrimRight(controller, "/"), true
}

func pickInt64FromMap(payload map[string]any, keys ...string) (int64, bool) {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		f, ok := numberToFloat64(value)
		if !ok {
			continue
		}
		return int64(f + 0.5), true
	}
	return 0, false
}

func pickUint64FromMap(payload map[string]any, keys ...string) (uint64, bool) {
	for _, key := range keys {
		value, ok := payload[key]
		if !ok {
			continue
		}
		f, ok := numberToFloat64(value)
		if !ok || f < 0 {
			continue
		}
		return uint64(f), true
	}
	return 0, false
}

func numberToFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case json.Number:
		f, err := v.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func clampUint64ToInt64(v uint64) int64 {
	if v > uint64(math.MaxInt64) {
		return math.MaxInt64
	}
	return int64(v)
}

func parseTargetFromOutbound(raw string) string {
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return "-"
	}
	server, _ := out["server"].(string)
	port := extractPort(out["server_port"])
	if strings.TrimSpace(server) == "" {
		return "-"
	}
	if port > 0 {
		return server + ":" + strconv.Itoa(port)
	}
	return server
}

func extractPort(v any) int {
	switch value := v.(type) {
	case float64:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	case string:
		p, _ := strconv.Atoi(value)
		return p
	default:
		return 0
	}
}

func fallbackNodeName(name, tag string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return tag
}

func fallbackTimestamp(ts sql.NullString, fallback string) string {
	if ts.Valid && ts.String != "" {
		return ts.String
	}
	return fallback
}

func parseLimit(raw string, defVal, minVal, maxVal int) int {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return defVal
	}
	if n < minVal {
		return minVal
	}
	if n > maxVal {
		return maxVal
	}
	return n
}
