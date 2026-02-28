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

type RoutingSettingsResponse struct {
	Data RoutingSettingsData `json:"data"`
}

type RoutingSettingsData struct {
	BypassPrivateEnabled bool     `json:"bypass_private_enabled"`
	BypassDomains        []string `json:"bypass_domains"`
	BypassCIDRs          []string `json:"bypass_cidrs"`
	UpdatedAt            string   `json:"updated_at,omitempty"`
}

type UpdateRoutingSettingsRequest struct {
	BypassPrivateEnabled *bool    `json:"bypass_private_enabled"`
	BypassDomains        []string `json:"bypass_domains"`
	BypassCIDRs          []string `json:"bypass_cidrs"`
}

type RoutingSummaryResponse struct {
	Data RoutingSummaryData `json:"data"`
}

type RoutingSummaryData struct {
	BypassPrivateEnabled bool     `json:"bypass_private_enabled"`
	BypassDomainsCount   int      `json:"bypass_domains_count"`
	BypassCIDRsCount     int      `json:"bypass_cidrs_count"`
	UpdatedAt            string   `json:"updated_at,omitempty"`
	GeoIPStatus          string   `json:"geoip_status"`
	GeoSiteStatus        string   `json:"geosite_status"`
	Notes                []string `json:"notes"`
}

type ForwardingRuntimeStatusResponse struct {
	Data ForwardingRuntimeStatus `json:"data"`
}

type ForwardingRuntimeStatus struct {
	Running      bool    `json:"running"`
	Status       string  `json:"status"`
	ErrorMessage *string `json:"error_message,omitempty"`
}

type ForwardingSummaryResponse struct {
	Data ForwardingSummaryData `json:"data"`
}

type ForwardingSummaryData struct {
	Running            bool                    `json:"running"`
	Status             string                  `json:"status"`
	ErrorMessage       *string                 `json:"error_message,omitempty"`
	SelectedNodesCount int                     `json:"selected_nodes_count"`
	Nodes              []ForwardingSummaryNode `json:"nodes"`
}

type ForwardingSummaryNode struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Tag           string  `json:"tag"`
	Type          string  `json:"type"`
	LastStatus    *string `json:"last_status,omitempty"`
	LastLatencyMs *int64  `json:"last_latency_ms,omitempty"`
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
