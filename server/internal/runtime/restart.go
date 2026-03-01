package runtime

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"boxpilot/server/internal/util/errorx"
)

const (
	defaultCheckCmd = `sing-box check -c "$SINGBOX_CONFIG"`
)

// Restart always uses process mode via SINGBOX_RESTART_CMD.
func Restart(ctx context.Context, configPath string) ([]byte, error) {
	cmdline, err := ValidateRestartContract(configPath)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "sh", "-lc", cmdline)
	cmd.Env = append(os.Environ(), "SINGBOX_CONFIG="+configPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, errorx.New(errorx.RTRestartFailed, "process restart failed").WithDetails(map[string]any{
			"cmd":    cmdline,
			"config": configPath,
			"output": string(truncate(out, 2048)),
		})
	}
	return out, nil
}

// Check validates generated config before restarting runtime.
func Check(ctx context.Context, configPath string) ([]byte, error) {
	if configPath == "" {
		return nil, errorx.New(errorx.REQMissingField, "config path required")
	}
	cmdline := strings.TrimSpace(os.Getenv("SINGBOX_CHECK_CMD"))
	if cmdline == "" {
		cmdline = defaultCheckCmd
	}
	cmd := exec.CommandContext(ctx, "sh", "-lc", cmdline)
	cmd.Env = append(os.Environ(), "SINGBOX_CONFIG="+configPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, errorx.New(errorx.CFGCheckFailed, "sing-box config preflight failed").WithDetails(map[string]any{
			"cmd":    cmdline,
			"config": configPath,
			"output": string(truncate(out, 2048)),
		})
	}
	return out, nil
}

// ValidateRestartContract enforces required runtime env contract.
func ValidateRestartContract(configPath string) (string, error) {
	restartCmd := strings.TrimSpace(os.Getenv("SINGBOX_RESTART_CMD"))
	if restartCmd == "" {
		return "", errorx.New(errorx.REQMissingField, "SINGBOX_RESTART_CMD is required")
	}

	envConfig := strings.TrimSpace(os.Getenv("SINGBOX_CONFIG"))
	if envConfig == "" {
		return "", errorx.New(errorx.REQMissingField, "SINGBOX_CONFIG is required for runtime restart").WithDetails(map[string]any{
			"expected_config_path": configPath,
		})
	}
	if configPath == "" {
		return "", errorx.New(errorx.REQMissingField, "config path required")
	}

	if !samePath(envConfig, configPath) {
		return "", errorx.New(errorx.REQInvalidField, fmt.Sprintf("SINGBOX_CONFIG (%s) does not match runtime config path (%s)", envConfig, configPath)).WithDetails(map[string]any{
			"env_config":     envConfig,
			"runtime_config": configPath,
		})
	}

	return restartCmd, nil
}

func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil && errB == nil {
		return filepath.Clean(absA) == filepath.Clean(absB)
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func truncate(b []byte, max int) []byte {
	if len(b) <= max {
		return b
	}
	return b[:max]
}
