package handlers

import (
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"boxpilot/server/internal/api/dto"
	"boxpilot/server/internal/service"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
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
		data = append(data, nodeRowToDTO(r))
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
	var forwardingEnabled *int
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
	if req.ForwardingEnabled != nil {
		v := 0
		if *req.ForwardingEnabled {
			v = 1
		}
		forwardingEnabled = &v
	}
	ok, err := repo.UpdateNode(h.DB, req.ID, name, enabled, forwardingEnabled)
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
		c.JSON(http.StatusOK, gin.H{"data": nodeRowToDTO(*row)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": nil})
}

func (h *Nodes) BatchForwarding(c *gin.Context) {
	var req struct {
		NodeIDs           []string `json:"node_ids"`
		ForwardingEnabled *bool    `json:"forwarding_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	if len(req.NodeIDs) == 0 {
		writeError(c, errorx.New(errorx.REQMissingField, "node_ids required"))
		return
	}
	if req.ForwardingEnabled == nil {
		writeError(c, errorx.New(errorx.REQMissingField, "forwarding_enabled required"))
		return
	}
	forwardingEnabled := 0
	if *req.ForwardingEnabled {
		forwardingEnabled = 1
	}
	updated := 0
	for _, id := range req.NodeIDs {
		ok, err := repo.UpdateNode(h.DB, id, nil, nil, &forwardingEnabled)
		if err != nil {
			writeError(c, errorx.New(errorx.NODEUpdateFailed, "batch update forwarding").WithDetails(map[string]any{
				"id":  id,
				"err": err.Error(),
			}))
			return
		}
		if ok {
			updated++
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"updated": updated}})
}

func (h *Nodes) Test(c *gin.Context) {
	var req struct {
		NodeIDs []string `json:"node_ids"`
		Mode    string   `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	if len(req.NodeIDs) == 0 {
		writeError(c, errorx.New(errorx.REQMissingField, "node_ids required"))
		return
	}
	if req.Mode == "" {
		req.Mode = "http"
	}
	if req.Mode != "ping" && req.Mode != "http" {
		writeError(c, errorx.New(errorx.REQInvalidField, "mode must be ping/http"))
		return
	}
	policy, err := service.LoadForwardingPolicy(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get forwarding policy"))
		return
	}

	type probeTask struct {
		index  int
		nodeID string
		row    *repo.NodeRow
	}
	type probeResult struct {
		index      int
		nodeID     string
		status     string
		latency    *int
		errMessage string
	}

	results := make([]map[string]any, len(req.NodeIDs))
	tasks := make([]probeTask, 0, len(req.NodeIDs))
	for idx, nodeID := range req.NodeIDs {
		row, err := repo.GetNode(h.DB, nodeID)
		if err != nil {
			results[idx] = map[string]any{
				"node_id": nodeID,
				"status":  "error",
				"error":   err.Error(),
			}
			continue
		}
		if row == nil {
			results[idx] = map[string]any{
				"node_id": nodeID,
				"status":  "error",
				"error":   "node not found",
			}
			continue
		}
		tasks = append(tasks, probeTask{index: idx, nodeID: nodeID, row: row})
	}

	// Probe network concurrently to speed up "test all" while keeping DB writes serialized.
	probeOut := make([]probeResult, len(tasks))
	if len(tasks) > 0 {
		workers := minInt(policy.NodeTestConcurrency, len(tasks))
		if workers < 1 {
			workers = 1
		}
		var wg sync.WaitGroup
		taskCh := make(chan probeTask)
		resultCh := make(chan probeResult, len(tasks))
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for task := range taskCh {
					latency, status, errMsg := probeNode(task.row.OutboundJSON, req.Mode, time.Duration(policy.NodeTestTimeoutMs)*time.Millisecond)
					var latencyPtr *int
					if latency >= 0 {
						latencyPtr = &latency
					}
					resultCh <- probeResult{
						index:      task.index,
						nodeID:     task.nodeID,
						status:     status,
						latency:    latencyPtr,
						errMessage: errMsg,
					}
				}
			}()
		}
		for _, task := range tasks {
			taskCh <- task
		}
		close(taskCh)
		wg.Wait()
		close(resultCh)
		i := 0
		for r := range resultCh {
			probeOut[i] = r
			i++
		}
	}

	for _, r := range probeOut {
		if r.nodeID == "" {
			continue
		}
		if err := repo.SetNodeProbeResult(h.DB, r.nodeID, r.latency, r.status, r.errMessage); err != nil {
			results[r.index] = map[string]any{
				"node_id": r.nodeID,
				"status":  "error",
				"error":   err.Error(),
			}
			continue
		}
		results[r.index] = map[string]any{
			"node_id":    r.nodeID,
			"status":     r.status,
			"latency_ms": r.latency,
			"error":      nullIfEmpty(r.errMessage),
		}
	}

	final := make([]map[string]any, 0, len(results))
	for _, item := range results {
		if item != nil {
			final = append(final, item)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": final})
}

func (h *Nodes) Forwarding(c *gin.Context) {
	nodeID := c.Query("node_id")
	if nodeID == "" {
		writeError(c, errorx.New(errorx.REQMissingField, "node_id required"))
		return
	}
	if err := repo.EnsureNodeExists(h.DB, nodeID); err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.DBError, err.Error()))
		return
	}
	settings, err := repo.GetProxySettings(h.DB)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get proxy settings"))
		return
	}
	overrides, err := repo.GetNodeProxyOverrides(h.DB, nodeID)
	if err != nil {
		writeError(c, errorx.New(errorx.DBError, "get node proxy overrides"))
		return
	}
	_, status, errMsg := runtimeStatus(h.DB)
	httpCfg := buildForwardingConfig(settings["http"], overrides["http"], status, errMsg)
	socksCfg := buildForwardingConfig(settings["socks"], overrides["socks"], status, errMsg)
	c.JSON(http.StatusOK, dto.NodeForwardingResponse{
		Data: dto.NodeForwardingData{
			NodeID: nodeID,
			HTTP:   httpCfg,
			Socks:  socksCfg,
		},
	})
}

func (h *Nodes) UpdateForwarding(c *gin.Context) {
	var req dto.UpdateNodeForwardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	if req.NodeID == "" {
		writeError(c, errorx.New(errorx.REQMissingField, "node_id required"))
		return
	}
	if req.ProxyType != "http" && req.ProxyType != "socks" {
		writeError(c, errorx.New(errorx.REQInvalidField, "invalid proxy_type"))
		return
	}
	if req.UseGlobal {
		if err := repo.DeleteNodeProxyOverride(h.DB, req.NodeID, req.ProxyType); err != nil {
			writeError(c, errorx.New(errorx.DBError, "delete node proxy override"))
			return
		}
		h.Forwarding(c)
		return
	}
	if req.Enabled == nil {
		writeError(c, errorx.New(errorx.REQMissingField, "enabled required"))
		return
	}
	if req.Port < 1 || req.Port > 65535 {
		writeError(c, errorx.New(errorx.REQInvalidField, "port must be between 1 and 65535"))
		return
	}
	if req.AuthMode != "none" && req.AuthMode != "basic" {
		writeError(c, errorx.New(errorx.REQInvalidField, "invalid auth_mode"))
		return
	}
	if req.AuthMode == "basic" && (req.Username == "" || req.Password == "") {
		writeError(c, errorx.New(errorx.REQMissingField, "username/password required for basic auth"))
		return
	}
	row := repo.NodeProxyOverrideRow{
		NodeID:    req.NodeID,
		ProxyType: req.ProxyType,
		Enabled:   boolToInt(*req.Enabled),
		Port:      req.Port,
		AuthMode:  req.AuthMode,
		Username:  req.Username,
		Password:  req.Password,
		CreatedAt: util.NowRFC3339(),
		UpdatedAt: util.NowRFC3339(),
	}
	if err := repo.UpsertNodeProxyOverride(h.DB, row); err != nil {
		writeError(c, errorx.New(errorx.DBError, "update node proxy override"))
		return
	}
	h.Forwarding(c)
}

func (h *Nodes) RestartForwarding(c *gin.Context) {
	var req struct {
		NodeID string `json:"node_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, errorx.New(errorx.REQValidationFailed, "invalid body"))
		return
	}
	if req.NodeID == "" {
		writeError(c, errorx.New(errorx.REQMissingField, "node_id required"))
		return
	}
	configPath := service.ResolveConfigPath()
	if _, _, _, err := service.Reload(c.Request.Context(), h.DB, configPath); err != nil {
		if appErr, ok := err.(*errorx.AppError); ok {
			writeError(c, appErr)
			return
		}
		writeError(c, errorx.New(errorx.RTRestartFailed, err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

func buildForwardingConfig(global repo.ProxySettingsRow, override repo.NodeProxyOverrideRow, runtimeStatus string, errMsg *string) dto.ProxyConfig {
	cfg := dto.ProxyConfig{
		ProxyType:     global.ProxyType,
		Enabled:       global.Enabled == 1,
		ListenAddress: global.ListenAddress,
		Port:          global.Port,
		AuthMode:      global.AuthMode,
		Username:      global.Username,
		Password:      global.Password,
		Source:        "global",
	}
	if override.ProxyType != "" {
		cfg.Enabled = override.Enabled == 1
		cfg.Port = override.Port
		cfg.AuthMode = override.AuthMode
		cfg.Username = override.Username
		cfg.Password = override.Password
		cfg.Source = "override"
	}
	cfg.Status = statusFor(cfg.Enabled, runtimeStatus)
	cfg.ErrorMessage = errMsg
	return cfg
}

func nodeRowToDTO(r repo.NodeRow) dto.Node {
	d := dto.Node{
		ID:                r.ID,
		SubID:             r.SubID,
		Tag:               r.Tag,
		Name:              r.Name,
		Type:              r.Type,
		Enabled:           r.Enabled == 1,
		ForwardingEnabled: r.ForwardingEnabled == 1,
		CreatedAt:         r.CreatedAt,
	}
	meta := parseNodeMeta(r.OutboundJSON)
	d.Server = meta.Server
	d.ServerPort = meta.ServerPort
	d.Network = meta.Network
	d.TLSEnabled = meta.TLSEnabled
	if r.LastTestAt.Valid {
		d.LastTestAt = &r.LastTestAt.String
	}
	if r.LastLatencyMs.Valid {
		v := int(r.LastLatencyMs.Int64)
		d.LastLatencyMs = &v
	}
	if r.LastTestStatus.Valid {
		d.LastTestStatus = &r.LastTestStatus.String
	}
	if r.LastTestError.Valid {
		d.LastTestError = &r.LastTestError.String
	}
	return d
}

type nodeMeta struct {
	Server     string
	ServerPort int
	Network    string
	TLSEnabled bool
}

func parseNodeMeta(raw string) nodeMeta {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nodeMeta{}
	}
	out := nodeMeta{}
	if v, ok := m["server"].(string); ok {
		out.Server = v
	}
	switch p := m["server_port"].(type) {
	case float64:
		out.ServerPort = int(p)
	case int:
		out.ServerPort = p
	}
	if t, ok := m["transport"].(map[string]any); ok {
		if n, ok := t["type"].(string); ok {
			out.Network = n
		}
	}
	if t, ok := m["tls"].(map[string]any); ok {
		if enabled, ok := t["enabled"].(bool); ok {
			out.TLSEnabled = enabled
		}
	}
	return out
}

func probeNodePing(rawOutbound string, timeout time.Duration) (latencyMs int, status string, errMsg string) {
	meta := parseNodeMeta(rawOutbound)
	if meta.Server == "" || meta.ServerPort <= 0 {
		return -1, "error", "node has no server/server_port"
	}
	addr := net.JoinHostPort(meta.Server, strconv.Itoa(meta.ServerPort))
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return -1, "error", err.Error()
	}
	_ = conn.Close()
	return int(time.Since(start).Milliseconds()), "ok", ""
}

func probeNodeHTTP(rawOutbound string, timeout time.Duration) (latencyMs int, status string, errMsg string) {
	meta := parseNodeMeta(rawOutbound)
	if meta.Server == "" || meta.ServerPort <= 0 {
		return -1, "error", "node has no server/server_port"
	}
	scheme := "http"
	if meta.TLSEnabled {
		scheme = "https"
	}
	target := scheme + "://" + net.JoinHostPort(meta.Server, strconv.Itoa(meta.ServerPort)) + "/"
	client := &http.Client{
		Timeout: timeout,
	}
	req, err := http.NewRequest(http.MethodHead, target, nil)
	if err != nil {
		return -1, "error", err.Error()
	}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return -1, "error", err.Error()
	}
	_ = resp.Body.Close()
	return int(time.Since(start).Milliseconds()), "ok", ""
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func probeNode(rawOutbound, mode string, timeout time.Duration) (latencyMs int, status string, errMsg string) {
	if mode == "http" {
		return probeNodeHTTP(rawOutbound, timeout)
	}
	return probeNodePing(rawOutbound, timeout)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
