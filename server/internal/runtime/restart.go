package runtime

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"boxpilot/server/internal/util/errorx"
)

// Restart always uses process mode via SINGBOX_RESTART_CMD.
func Restart(ctx context.Context, configPath string) ([]byte, error) {
	cmdline := strings.TrimSpace(os.Getenv("SINGBOX_RESTART_CMD"))
	if cmdline == "" {
		return nil, errorx.New(errorx.REQMissingField, "SINGBOX_RESTART_CMD is required")
	}
	cmd := exec.CommandContext(ctx, "sh", "-lc", cmdline)
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

func truncate(b []byte, max int) []byte {
	if len(b) <= max {
		return b
	}
	return b[:max]
}
