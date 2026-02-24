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
