package dto

type Node struct {
	ID                string  `json:"id"`
	SubID             string  `json:"sub_id"`
	Tag               string  `json:"tag"`
	Name              string  `json:"name"`
	Type              string  `json:"type"`
	Enabled           bool    `json:"enabled"`
	ForwardingEnabled bool    `json:"forwarding_enabled"`
	Server            string  `json:"server,omitempty"`
	ServerPort        int     `json:"server_port,omitempty"`
	Network           string  `json:"network,omitempty"`
	TLSEnabled        bool    `json:"tls_enabled,omitempty"`
	LastTestAt        *string `json:"last_test_at,omitempty"`
	LastLatencyMs     *int    `json:"last_latency_ms,omitempty"`
	LastTestStatus    *string `json:"last_test_status,omitempty"`
	LastTestError     *string `json:"last_test_error,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

type UpdateNodeRequest struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Enabled           *bool  `json:"enabled"`
	ForwardingEnabled *bool  `json:"forwarding_enabled"`
}

type ManualNodeCreateRequest struct {
	Mode     string               `json:"mode"`
	RawInput string               `json:"raw_input"`
	Form     *ManualNodeFormInput `json:"form,omitempty"`
}

type ManualNodeFormInput struct {
	Type             string `json:"type"`
	Tag              string `json:"tag"`
	Name             string `json:"name"`
	Server           string `json:"server"`
	ServerPort       int    `json:"server_port"`
	UUID             string `json:"uuid"`
	Password         string `json:"password"`
	Method           string `json:"method"`
	Flow             string `json:"flow"`
	Network          string `json:"network"`
	WSPath           string `json:"ws_path"`
	WSHost           string `json:"ws_host"`
	TLSEnabled       bool   `json:"tls_enabled"`
	TLSServerName    string `json:"tls_server_name"`
	TLSInsecure      bool   `json:"tls_insecure"`
	RealityPublicKey string `json:"reality_public_key"`
	RealityShortID    string `json:"reality_short_id"`
	RealitySpiderX    string `json:"reality_spider_x"`
	UTLSFingerprint   string `json:"utls_fingerprint"`
	Hysteria2UpMbps   int    `json:"hysteria2_up_mbps"`
	Hysteria2DownMbps int    `json:"hysteria2_down_mbps"`
}

type ManualNodeCreateResponse struct {
	Data ManualNodeCreateData `json:"data"`
}

type ManualNodeCreateData struct {
	Count int    `json:"count"`
	Mode  string `json:"mode"`
	Nodes []Node `json:"nodes"`
}
