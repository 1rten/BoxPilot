package service

import (
	"database/sql"
	"encoding/json"
	"net"
	"strings"

	"boxpilot/server/internal/generator"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"
)

func LoadRoutingSettings(db *sql.DB) (generator.RoutingSettings, string, error) {
	def := generator.DefaultRoutingSettings()
	row, err := repo.GetRoutingSettings(db)
	if err != nil {
		return def, "", err
	}
	if row == nil {
		return def, "", nil
	}
	settings := generator.RoutingSettings{
		BypassPrivateEnabled: row.BypassPrivateEnabled == 1,
		BypassDomains:        decodeStringArray(row.BypassDomainsJSON, def.BypassDomains),
		BypassCIDRs:          decodeStringArray(row.BypassCIDRsJSON, def.BypassCIDRs),
	}
	normalized, err := NormalizeRoutingSettings(settings)
	if err != nil {
		return def, row.UpdatedAt, nil
	}
	return normalized, row.UpdatedAt, nil
}

func SaveRoutingSettings(db *sql.DB, settings generator.RoutingSettings) (generator.RoutingSettings, string, error) {
	normalized, err := NormalizeRoutingSettings(settings)
	if err != nil {
		return generator.RoutingSettings{}, "", err
	}
	domainsJSON, _ := json.Marshal(normalized.BypassDomains)
	cidrsJSON, _ := json.Marshal(normalized.BypassCIDRs)
	updatedAt := util.NowRFC3339()
	err = repo.UpsertRoutingSettings(db, repo.RoutingSettingsRow{
		BypassPrivateEnabled: boolToInt(normalized.BypassPrivateEnabled),
		BypassDomainsJSON:    string(domainsJSON),
		BypassCIDRsJSON:      string(cidrsJSON),
		UpdatedAt:            updatedAt,
	})
	if err != nil {
		return generator.RoutingSettings{}, "", err
	}
	return normalized, updatedAt, nil
}

func NormalizeRoutingSettings(settings generator.RoutingSettings) (generator.RoutingSettings, error) {
	domains := normalizeStringList(settings.BypassDomains)
	cidrs := normalizeStringList(settings.BypassCIDRs)
	for _, cidr := range cidrs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return generator.RoutingSettings{}, errorx.New(errorx.REQInvalidField, "invalid CIDR in bypass_cidrs").WithDetails(map[string]any{
				"value": cidr,
			})
		}
	}
	return generator.RoutingSettings{
		BypassPrivateEnabled: settings.BypassPrivateEnabled,
		BypassDomains:        domains,
		BypassCIDRs:          cidrs,
	}, nil
}

func decodeStringArray(raw string, fallback []string) []string {
	var out []string
	if strings.TrimSpace(raw) == "" {
		return append([]string(nil), fallback...)
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return append([]string(nil), fallback...)
	}
	return out
}

func normalizeStringList(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		v := strings.TrimSpace(item)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
