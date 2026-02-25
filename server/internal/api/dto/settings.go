package dto

type ProxyConfig struct {
	ProxyType     string  `json:"proxy_type"`
	Enabled       bool    `json:"enabled"`
	ListenAddress string  `json:"listen_address"`
	Port          int     `json:"port"`
	AuthMode      string  `json:"auth_mode"`
	Username      string  `json:"username,omitempty"`
	Password      string  `json:"password,omitempty"`
	Status        string  `json:"status,omitempty"`
	ErrorMessage  *string `json:"error_message,omitempty"`
	Source        string  `json:"source,omitempty"`
}

type ProxySettingsResponse struct {
	Data ProxySettingsData `json:"data"`
}

type ProxySettingsData struct {
	HTTP  ProxyConfig `json:"http"`
	Socks ProxyConfig `json:"socks"`
}

type UpdateProxySettingsRequest struct {
	ProxyType     string `json:"proxy_type"`
	Enabled       *bool  `json:"enabled"`
	ListenAddress string `json:"listen_address"`
	Port          int    `json:"port"`
	AuthMode      string `json:"auth_mode"`
	Username      string `json:"username"`
	Password      string `json:"password"`
}

type ProxyApplyResponse struct {
	Data ProxyApplyData `json:"data"`
}

type ProxyApplyData struct {
	ConfigVersion int    `json:"config_version"`
	ConfigHash    string `json:"config_hash"`
	RestartOutput string `json:"restart_output"`
	ReloadedAt    string `json:"reloaded_at"`
}

type ForwardingRuntimeStatusResponse struct {
	Data ForwardingRuntimeStatus `json:"data"`
}

type ForwardingRuntimeStatus struct {
	Running      bool    `json:"running"`
	Status       string  `json:"status"`
	ErrorMessage *string `json:"error_message,omitempty"`
}

type NodeForwardingResponse struct {
	Data NodeForwardingData `json:"data"`
}

type NodeForwardingData struct {
	NodeID string      `json:"node_id"`
	HTTP   ProxyConfig `json:"http"`
	Socks  ProxyConfig `json:"socks"`
}

type UpdateNodeForwardingRequest struct {
	NodeID    string `json:"node_id"`
	ProxyType string `json:"proxy_type"`
	UseGlobal bool   `json:"use_global"`
	Enabled   *bool  `json:"enabled"`
	Port      int    `json:"port"`
	AuthMode  string `json:"auth_mode"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}
