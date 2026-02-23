PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS subscriptions (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  url TEXT NOT NULL,
  type TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
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

CREATE TABLE IF NOT EXISTS nodes (
  id TEXT PRIMARY KEY,
  sub_id TEXT NOT NULL,
  tag TEXT NOT NULL,
  name TEXT NOT NULL,
  type TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  outbound_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (sub_id) REFERENCES subscriptions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_nodes_sub_id ON nodes(sub_id);
CREATE INDEX IF NOT EXISTS idx_nodes_enabled ON nodes(enabled);
CREATE UNIQUE INDEX IF NOT EXISTS uq_nodes_tag ON nodes(tag);
