package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"boxpilot/server/internal/util/errorx"
)

func TestValidateRestartContract(t *testing.T) {
	tmp := t.TempDir()
	cfg := filepath.Join(tmp, "sing-box.json")
	if err := os.WriteFile(cfg, []byte(`{}`), 0644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	t.Run("missing restart cmd", func(t *testing.T) {
		t.Setenv("SINGBOX_RESTART_CMD", "")
		t.Setenv("SINGBOX_CONFIG", cfg)
		_, err := ValidateRestartContract(cfg)
		assertCode(t, err, errorx.REQMissingField)
	})

	t.Run("missing config env", func(t *testing.T) {
		t.Setenv("SINGBOX_RESTART_CMD", "echo ok")
		t.Setenv("SINGBOX_CONFIG", "")
		_, err := ValidateRestartContract(cfg)
		assertCode(t, err, errorx.REQMissingField)
	})

	t.Run("config mismatch", func(t *testing.T) {
		t.Setenv("SINGBOX_RESTART_CMD", "echo ok")
		t.Setenv("SINGBOX_CONFIG", filepath.Join(tmp, "other.json"))
		_, err := ValidateRestartContract(cfg)
		assertCode(t, err, errorx.REQInvalidField)
	})

	t.Run("ok", func(t *testing.T) {
		t.Setenv("SINGBOX_RESTART_CMD", "echo ok")
		t.Setenv("SINGBOX_CONFIG", cfg)
		cmd, err := ValidateRestartContract(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cmd != "echo ok" {
			t.Fatalf("unexpected command: %s", cmd)
		}
	})
}

func TestCheck(t *testing.T) {
	tmp := t.TempDir()
	cfg := filepath.Join(tmp, "candidate.json")
	if err := os.WriteFile(cfg, []byte(`{"ok":true}`), 0644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}

	t.Setenv("SINGBOX_CONFIG", cfg)
	t.Run("success", func(t *testing.T) {
		t.Setenv("SINGBOX_CHECK_CMD", `test -s "$SINGBOX_CONFIG" && echo checked`)
		out, err := Check(context.Background(), cfg)
		if err != nil {
			t.Fatalf("unexpected check error: %v", err)
		}
		if !strings.Contains(string(out), "checked") {
			t.Fatalf("expected check output, got %q", string(out))
		}
	})

	t.Run("fail", func(t *testing.T) {
		t.Setenv("SINGBOX_CHECK_CMD", `echo broken >&2; exit 2`)
		_, err := Check(context.Background(), cfg)
		assertCode(t, err, errorx.CFGCheckFailed)
	})
}

func TestRestart(t *testing.T) {
	tmp := t.TempDir()
	cfg := filepath.Join(tmp, "runtime.json")
	if err := os.WriteFile(cfg, []byte(`{"ok":true}`), 0644); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	t.Setenv("SINGBOX_CONFIG", cfg)

	t.Run("success", func(t *testing.T) {
		t.Setenv("SINGBOX_RESTART_CMD", `test -s "$SINGBOX_CONFIG" && echo restarted`)
		out, err := Restart(context.Background(), cfg)
		if err != nil {
			t.Fatalf("unexpected restart error: %v", err)
		}
		if !strings.Contains(string(out), "restarted") {
			t.Fatalf("unexpected restart output: %q", string(out))
		}
	})

	t.Run("fail", func(t *testing.T) {
		t.Setenv("SINGBOX_RESTART_CMD", `echo restart-failed >&2; exit 1`)
		_, err := Restart(context.Background(), cfg)
		assertCode(t, err, errorx.RTRestartFailed)
	})
}

func assertCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %s, got nil", code)
	}
	appErr, ok := err.(*errorx.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.Code != code {
		t.Fatalf("expected code %s, got %s (%v)", code, appErr.Code, appErr)
	}
}
