package dto

type RuntimeStatusResponse struct {
	Data RuntimeStatusData `json:"data"`
}

type RuntimeStatusData struct {
	ConfigVersion     int          `json:"config_version"`
	ConfigHash        string       `json:"config_hash"`
	ForwardingRunning bool         `json:"forwarding_running"`
	NodesIncluded     int          `json:"nodes_included"`
	LastApplyDuration int          `json:"last_apply_duration_ms"`
	LastApplySuccess  *string      `json:"last_apply_success_at,omitempty"`
	LastReloadAt      *string      `json:"last_reload_at,omitempty"`
	LastReloadError   *string      `json:"last_reload_error,omitempty"`
	Ports             RuntimePorts `json:"ports"`
}

type RuntimePorts struct {
	HTTP  int `json:"http"`
	Socks int `json:"socks"`
}

type RuntimePlanRequest struct {
	IncludeDisabledNodes bool `json:"include_disabled_nodes"`
}

type RuntimePlanResponse struct {
	Data RuntimePlanData `json:"data"`
}

type RuntimePlanData struct {
	NodesIncluded int      `json:"nodes_included"`
	Tags          []string `json:"tags"`
	ConfigHash    string   `json:"config_hash"`
}

type RuntimeReloadRequest struct {
	ForceRestart bool `json:"force_restart"`
}

type RuntimeReloadResponse struct {
	Data RuntimeReloadData `json:"data"`
}

type RuntimeReloadData struct {
	ConfigVersion int    `json:"config_version"`
	ConfigHash    string `json:"config_hash"`
	NodesIncluded int    `json:"nodes_included"`
	RestartOutput string `json:"restart_output"`
	ReloadedAt    string `json:"reloaded_at"`
}

type RuntimeProxyCheckRequest struct {
	TargetURL string `json:"target_url"`
	TimeoutMS int    `json:"timeout_ms"`
}

type RuntimeProxyCheckResponse struct {
	Data RuntimeProxyCheckData `json:"data"`
}

type RuntimeProxyCheckData struct {
	TargetURL string                `json:"target_url"`
	CheckedAt string                `json:"checked_at"`
	HTTP      RuntimeProxyCheckItem `json:"http"`
	Socks     RuntimeProxyCheckItem `json:"socks"`
}

type RuntimeProxyCheckItem struct {
	Enabled    bool    `json:"enabled"`
	ProxyURL   string  `json:"proxy_url"`
	Connected  bool    `json:"connected"`
	TLSOK      bool    `json:"tls_ok"`
	StatusCode *int    `json:"status_code,omitempty"`
	LatencyMS  *int64  `json:"latency_ms,omitempty"`
	Error      *string `json:"error,omitempty"`
	EgressIP   *string `json:"egress_ip,omitempty"`
}

type RuntimeTrafficResponse struct {
	Data RuntimeTrafficData `json:"data"`
}

type RuntimeTrafficData struct {
	SampledAt    string `json:"sampled_at"`
	Source       string `json:"source"`
	RXRateBps    int64  `json:"rx_rate_bps"`
	TXRateBps    int64  `json:"tx_rate_bps"`
	RXTotalBytes int64  `json:"rx_total_bytes"`
	TXTotalBytes int64  `json:"tx_total_bytes"`
}

type RuntimeConnectionsResponse struct {
	Data RuntimeConnectionsData `json:"data"`
}

type RuntimeConnectionsData struct {
	ActiveCount int                 `json:"active_count"`
	Items       []RuntimeConnection `json:"items"`
}

type RuntimeConnection struct {
	ID          string  `json:"id"`
	NodeID      string  `json:"node_id"`
	NodeName    string  `json:"node_name"`
	NodeType    string  `json:"node_type"`
	Target      string  `json:"target"`
	Status      string  `json:"status"`
	LastTestAt  *string `json:"last_test_at,omitempty"`
	LatencyMs   *int64  `json:"latency_ms,omitempty"`
	Error       *string `json:"error,omitempty"`
	Forwarding  bool    `json:"forwarding"`
	LastUpdated string  `json:"last_updated"`
}

type RuntimeLogsResponse struct {
	Data RuntimeLogsData `json:"data"`
}

type RuntimeLogsData struct {
	Items []RuntimeLogItem `json:"items"`
}

type RuntimeLogItem struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Source    string `json:"source"`
	Message   string `json:"message"`
}
