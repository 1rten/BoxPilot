package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"boxpilot/server/internal/runtime"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"
)

const (
	lastKnownGoodSuffix = ".last-good"
)

func applyConfigWithPreflight(ctx context.Context, configPath string, cfg []byte) ([]byte, error) {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		return nil, errorx.New(errorx.REQMissingField, "config path required")
	}

	// Ensure restart env contract is valid before touching runtime config files.
	if _, err := runtime.ValidateRestartContract(configPath); err != nil {
		return nil, err
	}

	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errorx.New(errorx.CFGWriteFailed, "create config dir failed").WithDetails(map[string]any{
			"path": configPath,
			"err":  err.Error(),
		})
	}

	candidatePath := configPath + ".candidate"
	if err := os.WriteFile(candidatePath, cfg, 0644); err != nil {
		return nil, errorx.New(errorx.CFGWriteFailed, "write candidate config failed").WithDetails(map[string]any{
			"path": candidatePath,
			"err":  err.Error(),
		})
	}
	defer func() { _ = os.Remove(candidatePath) }()

	if _, err := runtime.Check(ctx, candidatePath); err != nil {
		return nil, err
	}

	prevConfig, prevErr := os.ReadFile(configPath)
	hadPrevConfig := prevErr == nil
	if prevErr != nil && !errors.Is(prevErr, os.ErrNotExist) {
		return nil, errorx.New(errorx.CFGWriteFailed, "read current config failed").WithDetails(map[string]any{
			"path": configPath,
			"err":  prevErr.Error(),
		})
	}

	if err := util.AtomicWrite(dir, base, cfg); err != nil {
		return nil, errorx.New(errorx.CFGWriteFailed, "write runtime config failed").WithDetails(map[string]any{
			"path": configPath,
			"err":  err.Error(),
		})
	}

	restartOut, restartErr := runtime.Restart(ctx, configPath)
	if restartErr == nil {
		_ = saveLastKnownGoodConfig(configPath, cfg)
		return restartOut, nil
	}

	rollbackConfig := []byte(nil)
	rollbackSource := ""
	if hadPrevConfig && len(prevConfig) > 0 {
		rollbackConfig = prevConfig
		rollbackSource = "previous_config"
	} else if backup, err := loadLastKnownGoodConfig(configPath); err == nil && len(backup) > 0 {
		rollbackConfig = backup
		rollbackSource = "last_known_good"
	}

	if len(rollbackConfig) == 0 {
		return restartOut, attachRollbackDetails(restartErr, false, false, "")
	}

	if err := util.AtomicWrite(dir, base, rollbackConfig); err != nil {
		return restartOut, errorx.New(errorx.CFGRollbackFailed, "restart failed and rollback write failed").WithDetails(map[string]any{
			"path":            configPath,
			"rollback_source": rollbackSource,
			"restart_output":  string(truncateOutput(restartOut, 2048)),
			"rollback_error":  err.Error(),
		})
	}

	rollbackOut, rollbackErr := runtime.Restart(ctx, configPath)
	if rollbackErr != nil {
		return joinOutputs(restartOut, rollbackOut), errorx.New(errorx.CFGRollbackFailed, "restart failed and rollback restart failed").WithDetails(map[string]any{
			"path":             configPath,
			"rollback_source":  rollbackSource,
			"restart_output":   string(truncateOutput(restartOut, 2048)),
			"rollback_output":  string(truncateOutput(rollbackOut, 2048)),
			"rollback_restart": rollbackErr.Error(),
		})
	}

	return joinOutputs(restartOut, rollbackOut), errorx.New(errorx.RTRestartFailed, "restart failed; rollback succeeded").WithDetails(map[string]any{
		"rollback_attempted": true,
		"rollback_success":   true,
		"rollback_source":    rollbackSource,
		"restart_output":     string(truncateOutput(restartOut, 2048)),
		"rollback_output":    string(truncateOutput(rollbackOut, 2048)),
	})
}

func saveLastKnownGoodConfig(configPath string, cfg []byte) error {
	dir := filepath.Dir(configPath)
	target := filepath.Base(configPath) + lastKnownGoodSuffix
	return util.AtomicWrite(dir, target, cfg)
}

func loadLastKnownGoodConfig(configPath string) ([]byte, error) {
	path := configPath + lastKnownGoodSuffix
	return os.ReadFile(path)
}

func attachRollbackDetails(err error, attempted, success bool, source string) error {
	appErr, ok := err.(*errorx.AppError)
	if !ok {
		return err
	}
	details := map[string]any{}
	for k, v := range appErr.Details {
		details[k] = v
	}
	details["rollback_attempted"] = attempted
	details["rollback_success"] = success
	if source != "" {
		details["rollback_source"] = source
	}
	appErr.Details = details
	return appErr
}

func joinOutputs(parts ...[]byte) []byte {
	buf := make([]byte, 0, 1024)
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		if len(buf) > 0 {
			buf = append(buf, '\n')
		}
		buf = append(buf, p...)
	}
	return buf
}

func truncateOutput(out []byte, max int) []byte {
	if len(out) <= max {
		return out
	}
	return out[:max]
}
