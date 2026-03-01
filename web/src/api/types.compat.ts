export type Subscription = {
  id: string;
  name: string;
  url: string;
  type: string;
  enabled: boolean;
  refresh_interval_sec: number;
  created_at: string;
  updated_at: string;
  last_fetch_at?: string | null;
  last_success_at?: string | null;
  last_error?: string | null;
  auto_update_enabled?: boolean;
  total_bytes?: number;
  used_bytes?: number;
  usage_percent?: number;
  expire_at?: string | null;
  profile_web_page?: string | null;
};

export type Node = {
  id: string;
  sub_id: string;
  tag: string;
  name: string;
  type: string;
  enabled: boolean;
  forwarding_enabled: boolean;
  created_at: string;
  server?: string;
  server_port?: number;
  network?: string;
  tls_enabled?: boolean;
  last_test_at?: string | null;
  last_latency_ms?: number | null;
  last_test_status?: string | null;
  last_test_error?: string | null;
};

export type RuntimeStatusData = {
  config_version: number;
  config_hash: string;
  forwarding_running?: boolean;
  nodes_included?: number;
  last_apply_duration_ms?: number;
  last_apply_success_at?: string | null;
  last_reload_at?: string | null;
  last_reload_error?: string | null;
  ports: { http: number; socks: number };
  runtime_mode?: string;
  singbox_container?: string;
};

export type RuntimeProxyCheckItem = {
  enabled: boolean;
  proxy_url: string;
  connected: boolean;
  tls_ok: boolean;
  status_code?: number | null;
  latency_ms?: number | null;
  error?: string | null;
  egress_ip?: string | null;
};

export type RuntimeProxyCheckData = {
  target_url: string;
  checked_at: string;
  http: RuntimeProxyCheckItem;
  socks: RuntimeProxyCheckItem;
};

export type ForwardingPolicyData = {
  healthy_only_enabled: boolean;
  max_latency_ms: number;
  allow_untested: boolean;
  updated_at?: string;
};

export type RuntimeTrafficData = {
  sampled_at: string;
  source: string;
  rx_rate_bps: number;
  tx_rate_bps: number;
  rx_total_bytes: number;
  tx_total_bytes: number;
};

export type RuntimeConnection = {
  id: string;
  node_id: string;
  node_name: string;
  node_type: string;
  target: string;
  status: string;
  last_test_at?: string | null;
  latency_ms?: number | null;
  error?: string | null;
  forwarding: boolean;
  last_updated: string;
};

export type RuntimeConnectionsData = {
  active_count: number;
  items: RuntimeConnection[];
};

export type RuntimeLogItem = {
  timestamp: string;
  level: string;
  source: string;
  message: string;
};

export type RuntimeLogsData = {
  items: RuntimeLogItem[];
};

export type RoutingSummaryData = {
  bypass_private_enabled: boolean;
  bypass_domains_count: number;
  bypass_cidrs_count: number;
  updated_at?: string;
  geoip_status?: string;
  geosite_status?: string;
  notes?: string[];
};

export type ProxyType = "http" | "socks";

export type ProxyConfig = {
  proxy_type: string;
  enabled: boolean;
  listen_address: string;
  port: number;
  auth_mode: string;
  username?: string;
  password?: string;
  status?: string;
  error_message?: string | null;
  source?: string;
};

export type ProxySettingsData = {
  http: ProxyConfig;
  socks: ProxyConfig;
};

export type ForwardingRuntimeStatus = {
  running: boolean;
  status: string;
  error_message?: string | null;
};

export type ForwardingSummaryNode = {
  id: string;
  name: string;
  tag: string;
  type: string;
  last_status?: string | null;
  last_latency_ms?: number | null;
};

export type ForwardingSummaryData = {
  running: boolean;
  status: string;
  error_message?: string | null;
  selected_nodes_count: number;
  nodes: ForwardingSummaryNode[];
};

export type NodeForwardingData = {
  node_id: string;
  http: ProxyConfig;
  socks: ProxyConfig;
};

export type RoutingSettingsData = {
  bypass_private_enabled: boolean;
  bypass_domains: string[];
  bypass_cidrs: string[];
  updated_at?: string;
};
