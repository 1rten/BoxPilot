package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"boxpilot/server/internal/util/errorx"
)

func TestApplyConfigWithPreflight_CheckFailedNoWrite(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "sing-box.json")
	if err := os.WriteFile(configPath, []byte(`{"mode":"good"}`), 0644); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	marker := filepath.Join(tmp, "restarted.marker")
	t.Setenv("SINGBOX_CONFIG", configPath)
	t.Setenv("SINGBOX_RESTART_CMD", `echo restarted > "$TEST_RESTART_MARKER"`)
	t.Setenv("SINGBOX_CHECK_CMD", `echo invalid >&2; exit 2`)
	t.Setenv("TEST_RESTART_MARKER", marker)

	_, err := applyConfigWithPreflight(context.Background(), configPath, []byte(`{"mode":"new"}`))
	assertAppErrorCode(t, err, errorx.CFGCheckFailed)

	got, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}
	if string(got) != `{"mode":"good"}` {
		t.Fatalf("config changed on check failure: %s", string(got))
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("restart command should not run on check failure")
	}
}

func TestApplyConfigWithPreflight_RestartFailedRollbackSucceeded(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "sing-box.json")
	if err := os.WriteFile(configPath, []byte(`{"mode":"good"}`), 0644); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	t.Setenv("SINGBOX_CONFIG", configPath)
	t.Setenv("SINGBOX_CHECK_CMD", `test -s "$SINGBOX_CONFIG"`)
	t.Setenv("SINGBOX_RESTART_CMD", `if grep -q '"mode":"new"' "$SINGBOX_CONFIG"; then echo broken >&2; exit 1; fi; echo restarted`)

	out, err := applyConfigWithPreflight(context.Background(), configPath, []byte(`{"mode":"new"}`))
	assertAppErrorCode(t, err, errorx.RTRestartFailed)
	appErr := err.(*errorx.AppError)
	if appErr.Details["rollback_success"] != true {
		t.Fatalf("expected rollback_success=true, got %#v", appErr.Details["rollback_success"])
	}
	if !strings.Contains(string(out), "broken") {
		t.Fatalf("expected restart failure output, got %q", string(out))
	}

	got, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("read config: %v", readErr)
	}
	if string(got) != `{"mode":"good"}` {
		t.Fatalf("expected rolled back config, got %s", string(got))
	}
}

func TestApplyConfigWithPreflight_SavesLastKnownGoodOnSuccess(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "sing-box.json")
	t.Setenv("SINGBOX_CONFIG", configPath)
	t.Setenv("SINGBOX_CHECK_CMD", `test -s "$SINGBOX_CONFIG"`)
	t.Setenv("SINGBOX_RESTART_CMD", `echo restarted`)

	_, err := applyConfigWithPreflight(context.Background(), configPath, []byte(`{"mode":"stable"}`))
	if err != nil {
		t.Fatalf("apply config failed: %v", err)
	}

	backupPath := configPath + lastKnownGoodSuffix
	backup, readErr := os.ReadFile(backupPath)
	if readErr != nil {
		t.Fatalf("read last-known-good config: %v", readErr)
	}
	if string(backup) != `{"mode":"stable"}` {
		t.Fatalf("unexpected backup content: %s", string(backup))
	}
}

func assertAppErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %s, got nil", code)
	}
	appErr, ok := err.(*errorx.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.Code != code {
		t.Fatalf("expected code %s, got %s", code, appErr.Code)
	}
}
