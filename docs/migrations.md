# BoxPilot Migration Guide

[中文](./zh-CN/migrations.md)

The project currently has a single migration file: `server/internal/store/migrations/0001_init.sql`.

That should not become the permanent place for every future schema change. New schema updates should be added as new numbered migration files.

## Current Behavior

At startup the app:

1. opens SQLite
2. ensures `schema_migrations` exists
3. reads `server/internal/store/migrations/*.sql`
4. applies unapplied versions in ascending order
5. fails startup on any migration error

## Naming

Use:

```text
0001_init.sql
0002_add_xxx.sql
0003_add_yyy.sql
```

Rules:

- 4-digit numeric prefix
- strictly increasing
- descriptive suffix
- `.sql` extension

## Current Schema Areas

`0001_init.sql` creates tables for:

- subscriptions
- nodes
- runtime state
- proxy settings
- routing settings
- forwarding policy
- subscription-derived routing metadata
- runtime group selections

## Guidelines

- forward-only migrations
- one file per version
- safe for existing databases
- keep schema and data migration in the same version when needed
- do not hide schema mutations in Go handlers or services

## SQLite Notes

For incompatible table changes, use the standard rebuild pattern:

1. create new table
2. copy data
3. drop old table
4. rename new table
5. recreate indexes

## When to Add a New Migration

Add a new file for:

- new columns
- new tables
- new indexes
- backfills
- default value changes requiring upgrade logic
- table replacement or cleanup

## Important Cross-Checks

Schema changes affecting routing/runtime metadata must also be reviewed in:

- `server/internal/store/repo/subscription_routing.go`
- `server/internal/generator/singbox.go`
- `server/internal/api/handlers/runtime.go`
