package runtime

import (
	"context"
	"os/exec"

	"boxpilot/server/internal/util/errorx"
)

func DockerRestart(ctx context.Context, container string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", "restart", container)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, errorx.New(errorx.RTRestartFailed, "docker restart failed").WithDetails(map[string]any{
			"container": container, "output": string(truncate(out, 2048)),
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
