package service

import (
	"os"
	"path/filepath"
	"strings"
)

func resolveDataDir() string {
	if stat, err := os.Stat("/data"); err == nil && stat.IsDir() {
		return "/data"
	}
	return "data"
}

// ResolveConfigPath returns a runtime config path suitable for both local and container runs.
func ResolveConfigPath() string {
	if p := strings.TrimSpace(os.Getenv("SINGBOX_CONFIG")); p != "" {
		return p
	}
	return filepath.Join(resolveDataDir(), "sing-box.json")
}

// ResolveRuleSetDir returns the directory where downloaded rule set files are cached.
func ResolveRuleSetDir() string {
	return filepath.Join(resolveDataDir(), "rule-sets")
}
