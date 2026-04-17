package generator

import (
	"boxpilot/server/internal/util/errorx"
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	DefaultAutoTestURL      = "https://www.gstatic.com/generate_204"
	DefaultAutoTestInterval = "30m"
	DefaultGeoSiteCNURL     = "https://ghfast.top/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/cn.srs"
	DefaultGeoIPCNURL       = "https://ghfast.top/https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/cn.srs"
)

type ProxyInbound struct {
	Type          string
	ListenAddress string
	Port          int
	Enabled       bool
	AuthMode      string
	Username      string
	Password      string
}

type RoutingSettings struct {
	BypassPrivateEnabled bool
	BypassDomains        []string
	BypassCIDRs          []string
	ListenerReadyMaxMs   int
}

type NodeOutbound struct {
	Tag     string
	RawJSON string
}

type RouteRuleSetRef struct {
	Tag        string
	SourceType string
	Format     string
	URL        string
	Path       string
}

type RouteRule struct {
	Priority       int
	RuleOrder      int
	MatcherType    string
	MatcherValue   string
	TargetOutbound string
}

type RoutingExtras struct {
	RuleSets          []RouteRuleSetRef
	Rules             []RouteRule
	GroupSelections   map[string]string
	BusinessNodePools map[string][]string
	AutoTestURL       string
	AutoTestInterval  string
}

func DefaultRoutingSettings() RoutingSettings {
	return RoutingSettings{
		BypassPrivateEnabled: true,
		BypassDomains:        []string{"localhost", "local"},
		BypassCIDRs: []string{
			"127.0.0.0/8",
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"169.254.0.0/16",
			"::1/128",
			"fc00::/7",
			"fe80::/10",
		},
		ListenerReadyMaxMs: 0,
	}
}

func BuildConfig(httpProxy ProxyInbound, socksProxy ProxyInbound, routing RoutingSettings, nodeOutboundJSONs []string) ([]byte, error) {
	nodes := make([]NodeOutbound, 0, len(nodeOutboundJSONs))
	for _, raw := range nodeOutboundJSONs {
		nodes = append(nodes, NodeOutbound{RawJSON: raw})
	}
	return BuildConfigWithRuntime(httpProxy, socksProxy, routing, nodes, RoutingExtras{})
}

func BuildConfigWithNodes(httpProxy ProxyInbound, socksProxy ProxyInbound, routing RoutingSettings, nodes []NodeOutbound) ([]byte, error) {
	return BuildConfigWithRuntime(httpProxy, socksProxy, routing, nodes, RoutingExtras{})
}

func BuildConfigWithRuntime(httpProxy ProxyInbound, socksProxy ProxyInbound, routing RoutingSettings, nodes []NodeOutbound, extras RoutingExtras) ([]byte, error) {
	inbounds := []map[string]any{}
	if httpProxy.Enabled {
		inbounds = append(inbounds, buildInbound("http", "http-in", httpProxy))
	}
	if socksProxy.Enabled {
		inbounds = append(inbounds, buildInbound("socks", "socks-in", socksProxy))
	}
	outbounds := []any{
		map[string]any{"type": "direct", "tag": "direct"},
		map[string]any{"type": "block", "tag": "block"},
	}
	var tags []string
	for _, node := range nodes {
		raw := strings.TrimSpace(node.RawJSON)
		if !json.Valid([]byte(raw)) {
			continue
		}
		outbounds = append(outbounds, json.RawMessage(raw))
		tag := strings.TrimSpace(node.Tag)
		if tag == "" {
			tag = parseTagFromOutbound(raw)
		}
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	manualMembers := make([]string, 0, len(tags)+1)
	manualMembers = append(manualMembers, "direct")
	manualSeen := map[string]struct{}{
		"direct": {},
	}
	for _, raw := range tags {
		tag := strings.TrimSpace(raw)
		if tag == "" || tag == "direct" || tag == "block" {
			continue
		}
		if _, ok := manualSeen[tag]; ok {
			continue
		}
		manualSeen[tag] = struct{}{}
		manualMembers = append(manualMembers, tag)
	}
	manualDefault := "direct"
	if len(manualMembers) > 1 {
		manualDefault = manualMembers[1]
	}

	autoURL := strings.TrimSpace(extras.AutoTestURL)
	if autoURL == "" {
		autoURL = DefaultAutoTestURL
	}
	autoInterval := strings.TrimSpace(extras.AutoTestInterval)
	if autoInterval == "" {
		autoInterval = DefaultAutoTestInterval
	}

	manualSelectorOutbounds := append([]string{}, manualMembers...)
	manualAutoMembers := filterExistingNodeTags(tags, manualMembers)
	if len(manualAutoMembers) > 0 {
		used := map[string]struct{}{
			"direct": {},
			"block":  {},
			"manual": {},
			"dns":    {},
		}
		for _, tag := range tags {
			used[tag] = struct{}{}
		}
		manualAutoTag := resolveUniqueTag("manual-auto", used)
		outbounds = append(outbounds, map[string]any{
			"type":      "urltest",
			"tag":       manualAutoTag,
			"outbounds": manualAutoMembers,
			"url":       autoURL,
			"interval":  autoInterval,
			"tolerance": 120,
		})
		manualSelectorOutbounds = append([]string{manualAutoTag}, manualSelectorOutbounds...)
	}

	if selected, ok := extras.GroupSelections["manual"]; ok && containsString(manualSelectorOutbounds, selected) {
		manualDefault = selected
	}
	outbounds = append(outbounds, map[string]any{
		"type":      "selector",
		"tag":       "manual",
		"outbounds": manualSelectorOutbounds,
		"default":   manualDefault,
	})
	route := map[string]any{
		"final": "manual",
	}
	routeRuleSets := make([]map[string]any, 0, 2+len(extras.RuleSets))
	routeRules := make([]map[string]any, 0, 4+len(extras.Rules))
	if routing.BypassPrivateEnabled {
		if len(routing.BypassDomains) > 0 {
			routeRules = append(routeRules, map[string]any{
				"domain_suffix": routing.BypassDomains,
				"outbound":      "direct",
			})
		}
		if len(routing.BypassCIDRs) > 0 {
			routeRules = append(routeRules, map[string]any{
				"ip_cidr":  routing.BypassCIDRs,
				"outbound": "direct",
			})
		}
		routeRuleSets = append(routeRuleSets,
			map[string]any{
				"tag":            "geosite-cn",
				"type":           "remote",
				"format":         "binary",
				"url":            DefaultGeoSiteCNURL,
				"download_detour": "direct",
			},
			map[string]any{
				"tag":            "geoip-cn",
				"type":           "remote",
				"format":         "binary",
				"url":            DefaultGeoIPCNURL,
				"download_detour": "direct",
			},
		)
		routeRules = append(routeRules, map[string]any{
			"rule_set": []string{"geosite-cn", "geoip-cn"},
			"outbound": "direct",
		})
	}
	routeRuleSets = append(routeRuleSets, buildRouteRuleSets(extras.RuleSets)...)

	targetMap := buildBusinessGroups(
		&outbounds,
		tags,
		extras.Rules,
		extras.GroupSelections,
		extras.BusinessNodePools,
		extras.AutoTestURL,
		extras.AutoTestInterval,
	)
	availableRuleSets := make(map[string]struct{}, len(routeRuleSets))
	for _, rs := range routeRuleSets {
		if tag, ok := rs["tag"].(string); ok && strings.TrimSpace(tag) != "" {
			availableRuleSets[strings.TrimSpace(tag)] = struct{}{}
		}
	}
	for _, r := range extras.Rules {
		targetTag, ok := targetMap[r.TargetOutbound]
		if !ok {
			continue
		}
		item := map[string]any{
			"outbound": targetTag,
		}
		switch r.MatcherType {
		case "domain":
			item["domain"] = []string{r.MatcherValue}
		case "domain_suffix":
			item["domain_suffix"] = []string{r.MatcherValue}
		case "domain_keyword":
			item["domain_keyword"] = []string{r.MatcherValue}
		case "ip_cidr":
			item["ip_cidr"] = []string{r.MatcherValue}
		case "rule_set":
			if _, ok := availableRuleSets[r.MatcherValue]; !ok {
				continue
			}
			item["rule_set"] = r.MatcherValue
		default:
			continue
		}
		routeRules = append(routeRules, item)
	}
	if len(routeRuleSets) > 0 {
		route["rule_set"] = routeRuleSets
	}
	if len(routeRules) > 0 {
		route["rules"] = routeRules
	}

	cfg := map[string]any{
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route":     route,
	}
	applyClashAPI(cfg)
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, errorx.New(errorx.CFGJSONInvalid, "marshal config")
	}
	return b, nil
}

func buildRouteRuleSets(extras []RouteRuleSetRef) []map[string]any {
	if len(extras) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(extras))
	seen := map[string]struct{}{}
	for _, rs := range extras {
		tag := strings.TrimSpace(rs.Tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		item := map[string]any{
			"tag":    tag,
			"type":   defaultString(strings.TrimSpace(rs.SourceType), "remote"),
			"format": defaultString(strings.TrimSpace(rs.Format), "binary"),
		}
		if u := strings.TrimSpace(rs.URL); u != "" {
			item["url"] = u
			item["download_detour"] = "direct"
		}
		if p := strings.TrimSpace(rs.Path); p != "" {
			item["path"] = p
		}
		if _, hasURL := item["url"]; !hasURL {
			if _, hasPath := item["path"]; !hasPath {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func buildBusinessGroups(
	outbounds *[]any,
	nodeTags []string,
	rules []RouteRule,
	selections map[string]string,
	businessNodePools map[string][]string,
	autoTestURL string,
	autoTestInterval string,
) map[string]string {
	targets := map[string]struct{}{}
	for _, r := range rules {
		target := strings.TrimSpace(r.TargetOutbound)
		if target == "" {
			continue
		}
		targets[target] = struct{}{}
	}
	result := map[string]string{}
	if len(targets) == 0 {
		return result
	}
	used := map[string]struct{}{
		"direct": {},
		"block":  {},
		"manual": {},
		"dns":    {},
	}
	for _, tag := range nodeTags {
		used[tag] = struct{}{}
	}
	targetList := make([]string, 0, len(targets))
	for target := range targets {
		targetList = append(targetList, target)
	}
	sort.Strings(targetList)
	autoURL := strings.TrimSpace(autoTestURL)
	if autoURL == "" {
		autoURL = DefaultAutoTestURL
	}
	autoInterval := strings.TrimSpace(autoTestInterval)
	if autoInterval == "" {
		autoInterval = DefaultAutoTestInterval
	}
	for _, target := range targetList {
		selectorTag := resolveUniqueTag("biz-"+slugTag(target), used)
		used[selectorTag] = struct{}{}
		autoTag := resolveUniqueTag(selectorTag+"-auto", used)
		used[autoTag] = struct{}{}
		result[target] = selectorTag
		poolMembers := businessNodePools[target]
		autoMembers := filterExistingNodeTags(nodeTags, poolMembers)
		manualMembers := filterExistingGroupMembers(nodeTags, poolMembers)
		selectorOutbounds := make([]string, 0, 2+len(manualMembers))
		selectorOutbounds = append(selectorOutbounds, "manual")
		if len(autoMembers) > 0 && len(manualMembers) > 0 {
			*outbounds = append(*outbounds, map[string]any{
				"type":      "urltest",
				"tag":       autoTag,
				"outbounds": autoMembers,
				"url":       autoURL,
				"interval":  autoInterval,
				"tolerance": 120,
			})
			selectorOutbounds = append([]string{autoTag}, selectorOutbounds...)
		}
		seenMember := map[string]struct{}{}
		for _, existing := range selectorOutbounds {
			seenMember[existing] = struct{}{}
		}
		for _, member := range manualMembers {
			tag := strings.TrimSpace(member)
			if tag == "" {
				continue
			}
			if _, ok := seenMember[tag]; ok {
				continue
			}
			seenMember[tag] = struct{}{}
			selectorOutbounds = append(selectorOutbounds, tag)
		}
		selectedDefault := "manual"
		if selected, ok := selections[selectorTag]; ok && containsString(selectorOutbounds, selected) {
			selectedDefault = selected
		}
		*outbounds = append(*outbounds, map[string]any{
			"type":      "selector",
			"tag":       selectorTag,
			"outbounds": selectorOutbounds,
			"default":   selectedDefault,
		})
	}
	return result
}

func filterExistingNodeTags(availableNodeTags, preferredNodeTags []string) []string {
	if len(availableNodeTags) == 0 || len(preferredNodeTags) == 0 {
		return nil
	}
	availableSet := map[string]struct{}{}
	for _, tag := range availableNodeTags {
		availableSet[tag] = struct{}{}
	}
	out := make([]string, 0, len(preferredNodeTags))
	seen := map[string]struct{}{}
	for _, raw := range preferredNodeTags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if _, ok := availableSet[tag]; !ok {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

func filterExistingGroupMembers(availableNodeTags, preferredNodeTags []string) []string {
	if len(preferredNodeTags) == 0 {
		return nil
	}
	availableSet := map[string]struct{}{}
	for _, tag := range availableNodeTags {
		availableSet[tag] = struct{}{}
	}
	out := make([]string, 0, len(preferredNodeTags))
	seen := map[string]struct{}{}
	for _, raw := range preferredNodeTags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		if tag != "direct" && tag != "block" {
			if _, ok := availableSet[tag]; !ok {
				continue
			}
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

func containsString(items []string, candidate string) bool {
	for _, v := range items {
		if v == candidate {
			return true
		}
	}
	return false
}

func slugTag(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "group"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		// Keep alphanumeric, Unicode letters (e.g., Chinese), and basic Emoji/Symbols
		isAZ := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		is09 := r >= '0' && r <= '9'
		isUnicodeLetter := r >= 0x80 && (unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSymbol(r))

		if isAZ || is09 || isUnicodeLetter {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "group"
	}
	return out
}

func resolveUniqueTag(base string, used map[string]struct{}) string {
	tag := strings.TrimSpace(base)
	if tag == "" {
		tag = "biz"
	}
	if _, exists := used[tag]; !exists {
		return tag
	}
	i := 2
	for {
		candidate := tag + "-" + strconv.Itoa(i)
		if _, exists := used[candidate]; !exists {
			return candidate
		}
		i++
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func parseTagFromOutbound(raw string) string {
	var m map[string]any
	if json.Unmarshal([]byte(raw), &m) != nil {
		return ""
	}
	if tag, ok := m["tag"].(string); ok {
		return strings.TrimSpace(tag)
	}
	return ""
}

func applyClashAPI(cfg map[string]any) {
	controller := strings.TrimSpace(os.Getenv("SINGBOX_CLASH_API_ADDR"))
	if controller == "" {
		controller = "127.0.0.1:9090"
	}
	if controller == "off" {
		return
	}

	clashAPI := map[string]any{
		"external_controller": controller,
	}
	if secret := strings.TrimSpace(os.Getenv("SINGBOX_CLASH_API_SECRET")); secret != "" {
		clashAPI["secret"] = secret
	}

	cfg["experimental"] = map[string]any{
		"clash_api": clashAPI,
	}
}

func buildInbound(inType, tag string, p ProxyInbound) map[string]any {
	inb := map[string]any{
		"type":        inType,
		"tag":         tag,
		"listen":      p.ListenAddress,
		"listen_port": p.Port,
	}
	if p.AuthMode == "basic" && p.Username != "" && p.Password != "" {
		inb["users"] = []map[string]any{
			{"username": p.Username, "password": p.Password},
		}
	}
	return inb
}
