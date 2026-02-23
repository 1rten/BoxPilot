CREATE TABLE IF NOT EXISTS runtime_state (
  id TEXT PRIMARY KEY,
  config_version INTEGER NOT NULL DEFAULT 0,
  config_hash TEXT NOT NULL DEFAULT '',
  last_reload_at TEXT,
  last_reload_error TEXT
);

INSERT OR IGNORE INTO runtime_state (id, config_version, config_hash)
VALUES ('runtime', 0, '');
