ALTER TABLE subscriptions ADD COLUMN auto_update_enabled INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_subscriptions_auto_update_enabled ON subscriptions(auto_update_enabled);
