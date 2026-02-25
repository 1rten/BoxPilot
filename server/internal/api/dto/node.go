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
