package dto

type RuntimeStatusResponse struct {
	Data RuntimeStatusData `json:"data"`
}

type RuntimeStatusData struct {
	ConfigVersion     int          `json:"config_version"`
	ConfigHash        string       `json:"config_hash"`
	ForwardingRunning bool         `json:"forwarding_running"`
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
