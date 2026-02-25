package service

import (
	"context"
	"database/sql"
	"log"
	"time"

	"boxpilot/server/internal/store/repo"
)

// StartSubscriptionScheduler runs periodic auto-refresh checks.
// It refreshes subscriptions that are enabled, auto-update enabled, and due by interval.
func StartSubscriptionScheduler(ctx context.Context, db *sql.DB, tick time.Duration) {
	if tick <= 0 {
		tick = 30 * time.Second
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runSubscriptionAutoRefresh(ctx, db)
		}
	}
}

func runSubscriptionAutoRefresh(ctx context.Context, db *sql.DB) {
	subs, err := repo.ListSubscriptions(db, false)
	if err != nil {
		log.Printf("scheduler: list subscriptions failed: %v", err)
		return
	}
	now := time.Now().UTC()
	for _, s := range subs {
		if s.Enabled != 1 || s.AutoUpdateEnabled != 1 {
			continue
		}
		interval := s.RefreshIntervalSec
		if interval < 60 {
			interval = 3600
		}
		if !shouldRefreshByInterval(s, now, time.Duration(interval)*time.Second) {
			continue
		}
		if _, _, _, err := RefreshSubscription(db, s.ID); err != nil {
			log.Printf("scheduler: refresh %s failed: %v", s.ID, err)
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func shouldRefreshByInterval(s repo.SubscriptionRow, now time.Time, interval time.Duration) bool {
	if !s.LastFetchAt.Valid || s.LastFetchAt.String == "" {
		return true
	}
	last, err := time.Parse(time.RFC3339, s.LastFetchAt.String)
	if err != nil {
		return true
	}
	return now.Sub(last) >= interval
}
