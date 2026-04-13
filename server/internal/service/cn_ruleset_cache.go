package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"
)

const (
	cnGeositeName = "geosite-cn"
	cnGeoipName   = "geoip-cn"
	cnGeositeURL  = "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/cn.srs"
	cnGeoipURL    = "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/cn.srs"
)

// CNRuleSetCache holds resolved local paths for CN rule-sets and stale metadata.
type CNRuleSetCache struct {
	RuleSets []generator.RouteRuleSetRef
	Stale    bool
	Warning  string
}

// CNRuleSetFetcher fetches raw bytes from a URL. Injected for testing.
type CNRuleSetFetcher func(ctx context.Context, url string) ([]byte, error)

// DefaultCNRuleSetFetcher is the production HTTP fetcher.
func DefaultCNRuleSetFetcher(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

// EnsureCNRulesetsReady downloads (or reuses cached) CN rule-sets and returns
// local RouteRuleSetRef entries suitable for passing as CNRuleSets in RoutingExtras.
//
// - On fresh download success: returns cache with Stale=false.
// - On download failure with existing cache: returns cache with Stale=true and Warning set.
// - On download failure with no cache: returns nil, error.
func EnsureCNRulesetsReady(ctx context.Context, configPath string, fetch CNRuleSetFetcher) (*CNRuleSetCache, error) {
	ruleDir := filepath.Join(filepath.Dir(configPath), "ruleset")

	type entry struct {
		tag      string
		filename string
		url      string
	}
	entries := []entry{
		{cnGeositeName, cnGeositeName + ".srs", cnGeositeURL},
		{cnGeoipName, cnGeoipName + ".srs", cnGeoipURL},
	}

	// Attempt refresh.
	var refreshErr error
	if err := os.MkdirAll(ruleDir, 0755); err == nil {
		for _, e := range entries {
			data, err := fetch(ctx, e.url)
			if err != nil {
				refreshErr = fmt.Errorf("fetch %s: %w", e.tag, err)
				break
			}
			if writeErr := util.AtomicWrite(ruleDir, e.filename, data); writeErr != nil {
				refreshErr = fmt.Errorf("write %s: %w", e.tag, writeErr)
				break
			}
		}
	} else {
		refreshErr = fmt.Errorf("create ruleset dir: %w", err)
	}

	// Build the ruleset refs from local files if they exist.
	refs := make([]generator.RouteRuleSetRef, 0, len(entries))
	missingFiles := false
	for _, e := range entries {
		path := filepath.Join(ruleDir, e.filename)
		if _, err := os.Stat(path); err != nil {
			missingFiles = true
			break
		}
		refs = append(refs, generator.RouteRuleSetRef{
			Tag:        e.tag,
			SourceType: "local",
			Format:     "binary",
			Path:       path,
		})
	}

	if refreshErr == nil {
		return &CNRuleSetCache{RuleSets: refs}, nil
	}

	// Refresh failed; fall back to stale cache if all files are present.
	if missingFiles {
		return nil, errorx.New(errorx.CFGBuildFailed,
			fmt.Sprintf("CN rule-set unavailable: %v; no local cache found", refreshErr))
	}

	return &CNRuleSetCache{
		RuleSets: refs,
		Stale:    true,
		Warning:  fmt.Sprintf("using stale CN rule-set cache (refresh failed: %v)", refreshErr),
	}, nil
}
