package handlers

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
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
)

type Runtime struct {
	DB *sql.DB
}

type runtimeTrafficSnapshot struct {
	mu          sync.Mutex
	lastAt      time.Time
	lastRX      uint64
	lastTX      uint64
	initialized bool
}

var trafficSnapshot runtimeTrafficSnapshot

func (h *Runtime) Status(c *gin.Context) {
	row, err := repo.GetRuntimeState(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get runtime state"))
		return
	}
	cfgVersion := 0
	cfgHash := ""
	forwardingRunning := false
	var lastReloadAt, lastReloadError *string
	if row != nil {
		cfgVersion = row.ConfigVersion
		cfgHash = row.ConfigHash
		forwardingRunning = row.ForwardingRunning == 1
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
	c.JSON(http.StatusOK, dto.RuntimeStatusResponse{
		Data: dto.RuntimeStatusData{
			ConfigVersion:     cfgVersion,
			ConfigHash:        cfgHash,
			ForwardingRunning: forwardingRunning,
			LastReloadAt:      lastReloadAt,
			LastReloadError:   lastReloadError,
			Ports:             dto.RuntimePorts{HTTP: httpPort, Socks: socksPort},
		},
	})
}

func (h *Runtime) Traffic(c *gin.Context) {
	rxTotal, txTotal, err := readNetDevTotals()
	now := time.Now().UTC()
	source := "proc_net_dev"
	if err != nil {
		source = "unavailable"
	}

	var rxRate, txRate int64
	trafficSnapshot.mu.Lock()
	if err == nil && trafficSnapshot.initialized {
		elapsed := now.Sub(trafficSnapshot.lastAt).Seconds()
		if elapsed > 0 {
			if rxTotal >= trafficSnapshot.lastRX {
				rxRate = int64(float64(rxTotal-trafficSnapshot.lastRX) / elapsed)
			}
			if txTotal >= trafficSnapshot.lastTX {
				txRate = int64(float64(txTotal-trafficSnapshot.lastTX) / elapsed)
			}
		}
	}
	if err == nil {
		trafficSnapshot.initialized = true
		trafficSnapshot.lastAt = now
		trafficSnapshot.lastRX = rxTotal
		trafficSnapshot.lastTX = txTotal
	}
	trafficSnapshot.mu.Unlock()

	c.JSON(http.StatusOK, dto.RuntimeTrafficResponse{
		Data: dto.RuntimeTrafficData{
			SampledAt:    now.Format(time.RFC3339),
			Source:       source,
			RXRateBps:    rxRate,
			TXRateBps:    txRate,
			RXTotalBytes: int64(rxTotal),
			TXTotalBytes: int64(txTotal),
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

	settings, err := repo.GetProxySettings(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get proxy settings"))
		return
	}
	httpProxy, socksProxy := runtimeProxyRowsToInbounds(settings["http"], settings["socks"])

	row, err := repo.GetRuntimeState(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get runtime state"))
		return
	}
	forwardingRunning := row != nil && row.ForwardingRunning == 1
	if !forwardingRunning {
		httpProxy.Enabled = false
		socksProxy.Enabled = false
	}

	nodes := []repo.NodeRow{}
	if req.IncludeDisabledNodes {
		nodes, err = repo.ListNodes(h.DB, "", nil)
	} else {
		nodes, err = repo.ListEnabledForwardingNodes(h.DB)
	}
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "list nodes for plan"))
		return
	}

	if forwardingRunning && (httpProxy.Enabled || socksProxy.Enabled) && len(nodes) == 0 {
		writeError(c, errorx.New(errorx.CFGNoEnabledNodes, "no forwarding nodes enabled"))
		return
	}

	routing, _, err := service.LoadRoutingSettings(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get routing settings"))
		return
	}

	jsons := make([]string, 0, len(nodes))
	tags := make([]string, 0, len(nodes))
	for _, node := range nodes {
		jsons = append(jsons, node.OutboundJSON)
		tags = append(tags, node.Tag)
	}
	cfg, err := generator.BuildConfig(httpProxy, socksProxy, routing, jsons)
	if err != nil {
		writeError(c, errorx.New(errorx.CFGBuildFailed, "build plan config"))
		return
	}

	c.JSON(http.StatusOK, dto.RuntimePlanResponse{
		Data: dto.RuntimePlanData{
			NodesIncluded: len(nodes),
			Tags:          tags,
			ConfigHash:    util.JSONHash(cfg),
		},
	})
}

func (h *Runtime) Reload(c *gin.Context) {
	configPath := service.ResolveConfigPath()
	v, hsh, out, err := service.Reload(c.Request.Context(), h.DB, configPath)
	if err != nil {
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

func readNetDevTotals() (uint64, uint64, error) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0, err
	}
	var rxTotal uint64
	var txTotal uint64

	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		// Skip headers.
		if lineNo <= 2 {
			continue
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}
		rx, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			continue
		}
		tx, err := strconv.ParseUint(fields[8], 10, 64)
		if err != nil {
			continue
		}
		rxTotal += rx
		txTotal += tx
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, err
	}
	return rxTotal, txTotal, nil
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
