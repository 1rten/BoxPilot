package util

import (
	"os"
	"path/filepath"
)

// AtomicWrite writes data to a file atomically (write to temp then rename).
// dir is the directory for the target file; target is the final filename (e.g. "sing-box.json").
func AtomicWrite(dir, target string, data []byte) error {
	tmp := filepath.Join(dir, target+".tmp")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, target))
}
