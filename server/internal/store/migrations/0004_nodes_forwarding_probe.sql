ALTER TABLE nodes ADD COLUMN forwarding_enabled INTEGER NOT NULL DEFAULT 1;
ALTER TABLE nodes ADD COLUMN last_test_at TEXT;
ALTER TABLE nodes ADD COLUMN last_latency_ms INTEGER;
ALTER TABLE nodes ADD COLUMN last_test_status TEXT;
ALTER TABLE nodes ADD COLUMN last_test_error TEXT;

