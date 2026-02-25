-- New and existing subscription nodes should default to no forwarding.
UPDATE nodes SET forwarding_enabled = 0;

