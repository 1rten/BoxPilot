PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS subscriptions (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  url TEXT NOT NULL,
  type TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  auto_update_enabled INTEGER NOT NULL DEFAULT 0,
  refresh_interval_sec INTEGER NOT NULL DEFAULT 3600,
  etag TEXT NOT NULL DEFAULT '',
  last_modified TEXT NOT NULL DEFAULT '',
  last_fetch_at TEXT,
  last_success_at TEXT,
  last_error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_enabled ON subscriptions(enabled);
CREATE INDEX IF NOT EXISTS idx_subscriptions_auto_update_enabled ON subscriptions(auto_update_enabled);

CREATE TABLE IF NOT EXISTS nodes (
  id TEXT PRIMARY KEY,
  sub_id TEXT NOT NULL,
  tag TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  forwarding_enabled INTEGER NOT NULL DEFAULT 0,
  outbound_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  last_test_at TEXT,
  last_latency_ms INTEGER,
  last_test_status TEXT,
  last_test_error TEXT,
  FOREIGN KEY (sub_id) REFERENCES subscriptions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_nodes_sub_id ON nodes(sub_id);
CREATE INDEX IF NOT EXISTS idx_nodes_enabled ON nodes(enabled);
CREATE UNIQUE INDEX IF NOT EXISTS uq_nodes_tag ON nodes(tag);

CREATE TABLE IF NOT EXISTS runtime_state (
  id TEXT PRIMARY KEY,
  config_version INTEGER NOT NULL DEFAULT 0,
  config_hash TEXT NOT NULL DEFAULT '',
  forwarding_running INTEGER NOT NULL DEFAULT 0,
  last_reload_at TEXT,
  last_reload_error TEXT
);

INSERT OR IGNORE INTO runtime_state (id, config_version, config_hash, forwarding_running)
VALUES ('runtime', 0, '', 0);

CREATE TABLE IF NOT EXISTS proxy_settings (
  proxy_type TEXT PRIMARY KEY,
  enabled INTEGER NOT NULL DEFAULT 1,
  listen_address TEXT NOT NULL DEFAULT '0.0.0.0',
  port INTEGER NOT NULL,
  auth_mode TEXT NOT NULL DEFAULT 'none',
  username TEXT NOT NULL DEFAULT '',
  password TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);

INSERT OR IGNORE INTO proxy_settings (proxy_type, enabled, listen_address, port, auth_mode, username, password, updated_at)
VALUES
  ('http', 1, '0.0.0.0', 7890, 'none', '', '', ''),
  ('socks', 1, '0.0.0.0', 7891, 'none', '', '', '');

CREATE TABLE IF NOT EXISTS node_proxy_overrides (
  id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  proxy_type TEXT NOT NULL,
  enabled INTEGER NOT NULL,
  port INTEGER NOT NULL,
  auth_mode TEXT NOT NULL DEFAULT 'none',
  username TEXT NOT NULL DEFAULT '',
  password TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (node_id, proxy_type),
  FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_node_proxy_overrides_node_id ON node_proxy_overrides(node_id);
