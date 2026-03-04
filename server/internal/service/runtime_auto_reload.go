package service

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"

	"boxpilot/server/internal/store/repo"
)

var (
	autoReloadMu     sync.Mutex
	autoReloadTimer  *time.Timer
	autoReloadQueued bool
)

const autoReloadDebounce = 1200 * time.Millisecond

// ReloadIfForwardingRunning schedules a debounced runtime reload when forwarding is running.
// Multiple rapid updates are coalesced into one reload.
func ReloadIfForwardingRunning(ctx context.Context, db *sql.DB) error {
	running, err := isForwardingRunning(db)
	if err != nil {
		return err
	}
	if !running {
		return nil
	}
	_ = ctx // keep signature compatible for call sites.
	queueAutoReload(db)
	return nil
}

func isForwardingRunning(db *sql.DB) (bool, error) {
	row, err := repo.GetRuntimeState(db)
	if err != nil {
		return false, err
	}
	return row != nil && row.ForwardingRunning == 1, nil
}

func queueAutoReload(db *sql.DB) {
	autoReloadMu.Lock()
	defer autoReloadMu.Unlock()

	autoReloadQueued = true
	if autoReloadTimer != nil {
		autoReloadTimer.Reset(autoReloadDebounce)
		return
	}
	autoReloadTimer = time.AfterFunc(autoReloadDebounce, func() {
		runQueuedReload(db)
	})
}

func runQueuedReload(db *sql.DB) {
	autoReloadMu.Lock()
	if !autoReloadQueued {
		autoReloadTimer = nil
		autoReloadMu.Unlock()
		return
	}
	autoReloadQueued = false
	autoReloadTimer = nil
	autoReloadMu.Unlock()

	running, err := isForwardingRunning(db)
	if err != nil {
		log.Printf("auto-reload: get runtime state failed: %v", err)
		return
	}
	if !running {
		return
	}

	configPath := ResolveConfigPath()
	if _, _, _, err := Reload(context.Background(), db, configPath); err != nil {
		log.Printf("auto-reload: runtime reload failed: %v", err)
	}
}
