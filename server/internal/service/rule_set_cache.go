package service

import (
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"boxpilot/server/internal/generator"
)

type ruleSetEntry struct {
	Filename string
	URL      string
}

var builtinRuleSets = []ruleSetEntry{
	{Filename: "geosite-cn.srs", URL: generator.DefaultGeoSiteCNURL},
	{Filename: "geoip-cn.srs", URL: generator.DefaultGeoIPCNURL},
}

// EnsureBuiltinRuleSets downloads the built-in rule set files if they don't
// already exist locally. This avoids sing-box having to download them at
// startup, which can cause listener startup timeouts.
func EnsureBuiltinRuleSets() {
	dir := ResolveRuleSetDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("rule-sets: create dir %s: %v", dir, err)
		return
	}

	client := &http.Client{Timeout: 120 * time.Second}
	for _, rs := range builtinRuleSets {
		target := filepath.Join(dir, rs.Filename)
		if _, err := os.Stat(target); err == nil {
			continue // already cached
		}
		log.Printf("rule-sets: downloading %s from %s", rs.Filename, rs.URL)
		if err := downloadFile(client, rs.URL, target); err != nil {
			log.Printf("rule-sets: download %s failed: %v (sing-box will download on demand)", rs.Filename, err)
			continue
		}
		log.Printf("rule-sets: cached %s", rs.Filename)
	}
}

// CachedRuleSetPath returns the local path for a built-in rule set file if it
// exists, or empty string if it hasn't been downloaded yet.
func CachedRuleSetPath(filename string) string {
	p := filepath.Join(ResolveRuleSetDir(), filename)
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

func downloadFile(client *http.Client, url, path string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tmp)
		return closeErr
	}
	if resp.StatusCode != http.StatusOK {
		os.Remove(tmp)
		return nil // treat non-200 as non-fatal; sing-box will download on demand
	}
	return os.Rename(tmp, path)
}
