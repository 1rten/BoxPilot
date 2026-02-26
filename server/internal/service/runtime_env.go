package service

import (
	"os"
	"path/filepath"
	"strings"
)

// ResolveConfigPath returns a runtime config path suitable for both local and container runs.
func ResolveConfigPath() string {
	if p := strings.TrimSpace(os.Getenv("SINGBOX_CONFIG")); p != "" {
		return p
	}
	if stat, err := os.Stat("/data"); err == nil && stat.IsDir() {
		return "/data/sing-box.json"
	}
	return filepath.Join("data", "sing-box.json")
}
