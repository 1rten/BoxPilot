package parser

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/url"
	"strconv"
	"strings"

	"boxpilot/server/internal/util/errorx"

	"gopkg.in/yaml.v3"
)

type OutboundItem struct {
	Tag  string
	Type string
	Raw  json.RawMessage
}

type RuleSetItem struct {
	Tag        string
	SourceType string
	Format     string
	URL        string
	Path       string
}

type RoutingRuleItem struct {
	Priority       int
	RuleOrder      int
	MatcherType    string
	MatcherValue   string
	TargetOutbound string
	SourceKind     string
}

type BusinessGroupItem struct {
	TargetOutbound string
	NodeTags       []string
}

type ParsedSubscription struct {
	Outbounds      []OutboundItem
	RuleSets       []RuleSetItem
	Rules          []RoutingRuleItem
	BusinessGroups []BusinessGroupItem
}

var filterTypes = map[string]bool{
	"direct": true, "block": true, "dns": true, "selector": true, "urltest": true,
}

// ParseSubscription auto-detects the payload format and converts supported subscriptions
// into sing-box outbounds.
func ParseSubscription(body []byte) ([]OutboundItem, error) {
	parsed, err := ParseSubscriptionBundle(body)
	if err != nil {
		return nil, err
	}
	return parsed.Outbounds, nil
}

// ParseSubscriptionBundle parses nodes and routing metadata from subscription payload.
func ParseSubscriptionBundle(body []byte) (ParsedSubscription, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return ParsedSubscription{}, errorx.New(errorx.SUBParseFailed, "empty subscription body")
	}

	if out, ok, err := parseSingboxJSON(trimmed); ok {
		return finalizeParsedBundle(out, parseSingboxRouting(trimmed), err, "singbox_json")
	}
	if out, ok, err := parseClashYAML(trimmed); ok {
		return finalizeParsedBundle(out, parseClashRouting(trimmed), err, "clash_yaml")
	}
	if out, ok, err := parseTraditionalURIList(trimmed); ok {
		return finalizeParsedBundle(out, nil, err, "traditional_uri")
	}

	if decoded, ok := decodeBase64Payload(trimmed); ok {
		if out, parsed, err := parseSingboxJSON(decoded); parsed {
			return finalizeParsedBundle(out, parseSingboxRouting(decoded), err, "singbox_base64")
		}
		if out, parsed, err := parseClashYAML(decoded); parsed {
			return finalizeParsedBundle(out, parseClashRouting(decoded), err, "clash_base64")
		}
		if out, parsed, err := parseTraditionalURIList(decoded); parsed {
			return finalizeParsedBundle(out, nil, err, "traditional_base64")
		}
	}

	return ParsedSubscription{}, errorx.New(errorx.SUBFormatUnsupported, "subscription format unsupported")
}

func finalizeParsed(out []OutboundItem, parseErr error, format string) ([]OutboundItem, error) {
	bundle, err := finalizeParsedBundle(out, nil, parseErr, format)
	if err != nil {
		return nil, err
	}
	return bundle.Outbounds, nil
}

func finalizeParsedBundle(out []OutboundItem, routeParsed *ParsedSubscription, parseErr error, format string) (ParsedSubscription, error) {
	if parseErr != nil {
		return ParsedSubscription{}, parseErr
	}
	if len(out) == 0 {
		return ParsedSubscription{}, errorx.New(errorx.SUBEmptyOutbounds, "no supported outbounds found").WithDetails(map[string]any{
			"format": format,
		})
	}
	result := ParsedSubscription{Outbounds: out}
	if routeParsed != nil {
		result.RuleSets = routeParsed.RuleSets
		result.Rules = routeParsed.Rules
		result.BusinessGroups = routeParsed.BusinessGroups
	}
	return result, nil
}

func parseSingboxJSON(payload []byte) ([]OutboundItem, bool, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(payload, &raw); err == nil {
		if len(raw) == 0 {
			return nil, true, nil
		}
		out, parseErr := parseSingboxArray(raw)
		return out, true, parseErr
	}

	var obj struct {
		Outbounds []json.RawMessage `json:"outbounds"`
	}
	if err := json.Unmarshal(payload, &obj); err != nil {
		return nil, false, nil
	}
	out, parseErr := parseSingboxArray(obj.Outbounds)
	return out, true, parseErr
}

func parseSingboxArray(arr []json.RawMessage) ([]OutboundItem, error) {
	out := make([]OutboundItem, 0, len(arr))
	for _, b := range arr {
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			continue
		}
		t, _ := m["type"].(string)
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || filterTypes[t] {
			continue
		}
		tag, _ := m["tag"].(string)
		out = append(out, OutboundItem{Tag: tag, Type: t, Raw: b})
	}
	return out, nil
}

func parseSingboxRouting(payload []byte) *ParsedSubscription {
	var doc struct {
		Outbounds []map[string]any `json:"outbounds"`
		Route     struct {
			RuleSet []json.RawMessage `json:"rule_set"`
			Rules   []json.RawMessage `json:"rules"`
		} `json:"route"`
	}
	if err := json.Unmarshal(payload, &doc); err != nil {
		return nil
	}

	result := &ParsedSubscription{
		RuleSets:       make([]RuleSetItem, 0, len(doc.Route.RuleSet)),
		Rules:          make([]RoutingRuleItem, 0, len(doc.Route.Rules)),
		BusinessGroups: []BusinessGroupItem{},
	}
	targetOrder := make([]string, 0, len(doc.Route.Rules))
	targetSeen := map[string]struct{}{}

	for idx, raw := range doc.Route.Rules {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		target := strings.TrimSpace(toString(m["outbound"]))
		if !isBusinessTargetTag(target) {
			continue
		}
		if _, ok := targetSeen[target]; !ok {
			targetSeen[target] = struct{}{}
			targetOrder = append(targetOrder, target)
		}
		result.Rules = append(result.Rules, explodeRuleMatchers(
			m, target, "singbox", 200, idx,
		)...)
	}
	nodeSet, groupRefs := parseSingboxGroupRefs(doc.Outbounds)
	result.BusinessGroups = buildBusinessGroupsForTargets(targetOrder, nodeSet, groupRefs, nil)

	usedRuleSetTags := map[string]struct{}{}
	for _, r := range result.Rules {
		if r.MatcherType == "rule_set" {
			usedRuleSetTags[r.MatcherValue] = struct{}{}
		}
	}

	for _, raw := range doc.Route.RuleSet {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		tag := strings.TrimSpace(toString(m["tag"]))
		if tag == "" {
			continue
		}
		if len(usedRuleSetTags) > 0 {
			if _, ok := usedRuleSetTags[tag]; !ok {
				continue
			}
		}
		sourceType := strings.ToLower(strings.TrimSpace(toString(m["type"])))
		if sourceType == "" {
			sourceType = "remote"
		}
		format := strings.ToLower(strings.TrimSpace(toString(m["format"])))
		if format == "" {
			format = "binary"
		}
		item := RuleSetItem{
			Tag:        tag,
			SourceType: sourceType,
			Format:     format,
			URL:        strings.TrimSpace(toString(m["url"])),
			Path:       strings.TrimSpace(toString(m["path"])),
		}
		if item.URL == "" && item.Path == "" {
			continue
		}
		result.RuleSets = append(result.RuleSets, item)
	}

	return result
}

func parseClashRouting(payload []byte) *ParsedSubscription {
	var doc struct {
		Proxies     []map[string]any `yaml:"proxies"`
		ProxyGroups []struct {
			Name    string   `yaml:"name"`
			Type    string   `yaml:"type"`
			Proxies []string `yaml:"proxies"`
		} `yaml:"proxy-groups"`
		Rules         []string                  `yaml:"rules"`
		RuleProviders map[string]map[string]any `yaml:"rule-providers"`
	}
	if err := yaml.Unmarshal(payload, &doc); err != nil {
		return nil
	}

	result := &ParsedSubscription{
		RuleSets:       []RuleSetItem{},
		Rules:          []RoutingRuleItem{},
		BusinessGroups: []BusinessGroupItem{},
	}
	usedProviders := map[string]struct{}{}
	targetOrder := make([]string, 0, len(doc.Rules))
	targetSeen := map[string]struct{}{}

	for idx, line := range doc.Rules {
		parts := splitClashRuleLine(line)
		if len(parts) < 3 {
			continue
		}
		ruleType := strings.ToUpper(parts[0])
		matcherValue := strings.TrimSpace(parts[1])
		target := strings.TrimSpace(parts[2])
		if !isBusinessTargetTag(target) {
			continue
		}
		if _, ok := targetSeen[target]; !ok {
			targetSeen[target] = struct{}{}
			targetOrder = append(targetOrder, target)
		}

		var matcherType string
		switch ruleType {
		case "DOMAIN":
			matcherType = "domain"
		case "DOMAIN-SUFFIX":
			matcherType = "domain_suffix"
		case "DOMAIN-KEYWORD":
			matcherType = "domain_keyword"
		case "IP-CIDR", "IP-CIDR6":
			matcherType = "ip_cidr"
		case "RULE-SET", "RULESET":
			matcherType = "rule_set"
			usedProviders[matcherValue] = struct{}{}
		default:
			continue
		}

		matcherValue = normalizeMatcherValue(matcherType, matcherValue)
		if matcherValue == "" {
			continue
		}

		result.Rules = append(result.Rules, RoutingRuleItem{
			Priority:       200,
			RuleOrder:      idx,
			MatcherType:    matcherType,
			MatcherValue:   matcherValue,
			TargetOutbound: target,
			SourceKind:     "clash",
		})
	}
	nodeSet, groupRefs, alias := parseClashGroupRefs(doc.Proxies, doc.ProxyGroups)
	result.BusinessGroups = buildBusinessGroupsForTargets(targetOrder, nodeSet, groupRefs, alias)

	for tag, provider := range doc.RuleProviders {
		if len(usedProviders) > 0 {
			if _, ok := usedProviders[tag]; !ok {
				continue
			}
		}
		urlValue := strings.TrimSpace(toString(provider["url"]))
		pathValue := strings.TrimSpace(toString(provider["path"]))
		if urlValue == "" && pathValue == "" {
			continue
		}
		sourceType := "remote"
		if pathValue != "" && urlValue == "" {
			sourceType = "local"
		}
		format := strings.ToLower(strings.TrimSpace(toString(provider["format"])))
		if format == "" {
			format = "source"
		}
		result.RuleSets = append(result.RuleSets, RuleSetItem{
			Tag:        tag,
			SourceType: sourceType,
			Format:     format,
			URL:        urlValue,
			Path:       pathValue,
		})
	}
	return result
}

func parseSingboxGroupRefs(outbounds []map[string]any) (map[string]struct{}, map[string][]string) {
	nodeSet := map[string]struct{}{}
	groupRefs := map[string][]string{}
	for _, outbound := range outbounds {
		tag := strings.TrimSpace(toString(outbound["tag"]))
		if tag == "" {
			continue
		}
		typ := strings.ToLower(strings.TrimSpace(toString(outbound["type"])))
		if !filterTypes[typ] {
			nodeSet[tag] = struct{}{}
			continue
		}
		switch typ {
		case "selector", "urltest", "fallback", "load_balance", "load-balance":
			members := explodeMatcherValues(outbound["outbounds"])
			if len(members) > 0 {
				groupRefs[tag] = members
			}
		}
	}
	return nodeSet, groupRefs
}

func parseClashGroupRefs(
	proxies []map[string]any,
	proxyGroups []struct {
		Name    string   `yaml:"name"`
		Type    string   `yaml:"type"`
		Proxies []string `yaml:"proxies"`
	},
) (map[string]struct{}, map[string][]string, map[string]string) {
	nodeSet := map[string]struct{}{}
	groupRefs := map[string][]string{}
	alias := map[string]string{}
	for _, p := range proxies {
		item, err := clashProxyToOutbound(p)
		if err != nil || item == nil {
			continue
		}
		tag := strings.TrimSpace(item.Tag)
		if tag == "" {
			continue
		}
		nodeSet[tag] = struct{}{}
		if _, ok := alias[strings.ToLower(tag)]; !ok {
			alias[strings.ToLower(tag)] = tag
		}
	}
	for _, g := range proxyGroups {
		name := strings.TrimSpace(g.Name)
		if name == "" {
			continue
		}
		members := make([]string, 0, len(g.Proxies))
		for _, raw := range g.Proxies {
			member := strings.TrimSpace(raw)
			if member != "" {
				members = append(members, member)
			}
		}
		groupRefs[name] = members
		if _, ok := alias[strings.ToLower(name)]; !ok {
			alias[strings.ToLower(name)] = name
		}
	}
	return nodeSet, groupRefs, alias
}

func buildBusinessGroupsForTargets(
	targets []string,
	nodeSet map[string]struct{},
	groupRefs map[string][]string,
	alias map[string]string,
) []BusinessGroupItem {
	if len(targets) == 0 {
		return nil
	}
	out := make([]BusinessGroupItem, 0, len(targets))
	for _, rawTarget := range targets {
		target := strings.TrimSpace(rawTarget)
		if target == "" {
			continue
		}
		members := resolveBusinessNodeTags(target, nodeSet, groupRefs, alias)
		if len(members) == 0 {
			continue
		}
		out = append(out, BusinessGroupItem{
			TargetOutbound: target,
			NodeTags:       members,
		})
	}
	return out
}

func resolveBusinessNodeTags(
	target string,
	nodeSet map[string]struct{},
	groupRefs map[string][]string,
	alias map[string]string,
) []string {
	appendNode := func(name string, seen map[string]struct{}, out *[]string) {
		if _, done := seen[name]; done {
			return
		}
		seen[name] = struct{}{}
		*out = append(*out, name)
	}
	isConcreteOutbound := func(name string) bool {
		if name == "" {
			return false
		}
		if _, ok := nodeSet[name]; ok {
			return true
		}
		return name == "direct" || name == "block"
	}
	normalizeSpecial := func(raw string) string {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "direct", "bypass":
			return "direct"
		case "reject", "block", "blackhole":
			return "block"
		default:
			return ""
		}
	}
	canonical := func(raw string) string {
		if special := normalizeSpecial(raw); special != "" {
			return special
		}
		name := strings.TrimSpace(raw)
		if name == "" || alias == nil {
			return name
		}
		if mapped, ok := alias[strings.ToLower(name)]; ok {
			return mapped
		}
		return name
	}
	target = canonical(target)
	if target == "" {
		return nil
	}
	visitedGroup := map[string]bool{}
	seenNode := map[string]struct{}{}
	out := []string{}
	var collectSpecialConcrete func(name string, visited map[string]bool)
	collectSpecialConcrete = func(name string, visited map[string]bool) {
		current := canonical(name)
		if current == "" {
			return
		}
		if current == "direct" || current == "block" {
			appendNode(current, seenNode, &out)
			return
		}
		if _, ok := nodeSet[current]; ok {
			return
		}
		refs, ok := groupRefs[current]
		if !ok {
			return
		}
		if visited[current] {
			return
		}
		visited[current] = true
		for _, child := range refs {
			collectSpecialConcrete(child, visited)
		}
		visited[current] = false
	}
	var walk func(name string)
	walk = func(name string) {
		current := canonical(name)
		if current == "" {
			return
		}
		if isConcreteOutbound(current) {
			appendNode(current, seenNode, &out)
			return
		}
		refs, ok := groupRefs[current]
		if !ok {
			return
		}
		if visitedGroup[current] {
			return
		}
		visitedGroup[current] = true

		// If a business group has explicit concrete members, prefer them directly.
		// This avoids pulling huge generic pools via helper groups like "manual/proxy".
		directConcrete := make([]string, 0, len(refs))
		for _, child := range refs {
			c := canonical(child)
			if !isConcreteOutbound(c) {
				continue
			}
			directConcrete = append(directConcrete, c)
		}
		if len(directConcrete) > 0 {
			for _, child := range directConcrete {
				appendNode(child, seenNode, &out)
			}
			// Keep direct/block reachable via helper groups even when explicit nodes exist.
			for _, child := range refs {
				collectSpecialConcrete(child, map[string]bool{})
			}
			visitedGroup[current] = false
			return
		}
		for _, child := range refs {
			walk(child)
		}
		visitedGroup[current] = false
	}
	walk(target)
	return out
}

func explodeRuleMatchers(rule map[string]any, target, sourceKind string, priority, ruleOrder int) []RoutingRuleItem {
	type matcherDef struct {
		key         string
		matcherType string
	}
	matchers := []matcherDef{
		{key: "domain", matcherType: "domain"},
		{key: "domain_suffix", matcherType: "domain_suffix"},
		{key: "domain_keyword", matcherType: "domain_keyword"},
		{key: "ip_cidr", matcherType: "ip_cidr"},
		{key: "rule_set", matcherType: "rule_set"},
	}
	out := make([]RoutingRuleItem, 0, len(matchers))
	for _, m := range matchers {
		values := explodeMatcherValues(rule[m.key])
		for _, value := range values {
			normalized := normalizeMatcherValue(m.matcherType, value)
			if normalized == "" {
				continue
			}
			out = append(out, RoutingRuleItem{
				Priority:       priority,
				RuleOrder:      ruleOrder,
				MatcherType:    m.matcherType,
				MatcherValue:   normalized,
				TargetOutbound: target,
				SourceKind:     sourceKind,
			})
		}
	}
	return out
}

func explodeMatcherValues(v any) []string {
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil
		}
		return []string{s}
	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			if s := strings.TrimSpace(toString(item)); s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func splitClashRuleLine(line string) []string {
	raw := strings.TrimSpace(line)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

func normalizeMatcherValue(matcherType, raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	switch matcherType {
	case "domain", "domain_suffix", "domain_keyword":
		return strings.TrimSuffix(strings.ToLower(value), ".")
	case "ip_cidr":
		if _, n, err := net.ParseCIDR(value); err == nil && n != nil {
			return n.String()
		}
		return value
	default:
		return value
	}
}

func isBusinessTargetTag(tag string) bool {
	raw := strings.TrimSpace(tag)
	if raw == "" {
		return false
	}
	lower := strings.ToLower(raw)
	switch lower {
	case "direct", "block", "reject", "dns", "manual", "proxy", "proxy-auto":
		return false
	}
	if strings.Contains(lower, "直连") ||
		strings.Contains(lower, "manual") ||
		strings.Contains(lower, "手动切换") ||
		strings.Contains(lower, "auto_selector") ||
		strings.Contains(lower, "auto-selector") ||
		strings.Contains(lower, "自动选择") ||
		strings.Contains(lower, "漏网之鱼") ||
		strings.Contains(lower, "节点选择") {
		return false
	}
	return true
}

func parseClashYAML(payload []byte) ([]OutboundItem, bool, error) {
	var doc struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(payload, &doc); err != nil {
		return nil, false, nil
	}

	out := make([]OutboundItem, 0, len(doc.Proxies))
	for _, proxy := range doc.Proxies {
		item, err := clashProxyToOutbound(proxy)
		if err != nil || item == nil {
			continue
		}
		out = append(out, *item)
	}
	return out, true, nil
}

func clashProxyToOutbound(proxy map[string]any) (*OutboundItem, error) {
	typ := strings.ToLower(toString(proxy["type"]))
	tag := toString(proxy["name"])
	server := toString(proxy["server"])
	port := toInt(proxy["port"])

	if typ == "" || server == "" || port <= 0 {
		return nil, nil
	}

	switch typ {
	case "ss", "shadowsocks":
		method := toString(proxy["cipher"])
		if method == "" {
			method = toString(proxy["method"])
		}
		password := toString(proxy["password"])
		if method == "" || password == "" {
			return nil, nil
		}
		out := map[string]any{
			"type":        "shadowsocks",
			"tag":         tag,
			"server":      server,
			"server_port": port,
			"method":      method,
			"password":    password,
		}
		return mapToItem(out), nil
	case "vmess":
		uuid := toString(proxy["uuid"])
		if uuid == "" {
			return nil, nil
		}
		out := map[string]any{
			"type":        "vmess",
			"tag":         tag,
			"server":      server,
			"server_port": port,
			"uuid":        uuid,
			"security":    orDefault(toString(proxy["cipher"]), "auto"),
		}
		if aid, ok := toOptionalInt(proxy["alterId"]); ok {
			out["alter_id"] = aid
		}
		attachTransport(out, proxy)
		attachTLS(out, proxy)
		return mapToItem(out), nil
	case "vless":
		uuid := toString(proxy["uuid"])
		if uuid == "" {
			return nil, nil
		}
		out := map[string]any{
			"type":        "vless",
			"tag":         tag,
			"server":      server,
			"server_port": port,
			"uuid":        uuid,
		}
		if flow := toString(proxy["flow"]); flow != "" {
			out["flow"] = flow
		}
		attachTransport(out, proxy)
		attachTLS(out, proxy)
		return mapToItem(out), nil
	case "trojan":
		password := toString(proxy["password"])
		if password == "" {
			return nil, nil
		}
		out := map[string]any{
			"type":        "trojan",
			"tag":         tag,
			"server":      server,
			"server_port": port,
			"password":    password,
		}
		attachTransport(out, proxy)
		attachTLS(out, proxy)
		return mapToItem(out), nil
	case "hysteria2":
		password := toString(proxy["password"])
		if password == "" {
			return nil, nil
		}
		out := map[string]any{
			"type":        "hysteria2",
			"tag":         tag,
			"server":      server,
			"server_port": port,
			"password":    password,
		}
		if up, ok := toOptionalInt(proxy["up"]); ok && up > 0 {
			out["up_mbps"] = up
		}
		if down, ok := toOptionalInt(proxy["down"]); ok && down > 0 {
			out["down_mbps"] = down
		}
		// Clash field is 'obfs', sing-box field inside hysteria2 is 'obfs'
		if obfs := toString(proxy["obfs"]); obfs != "" {
			obfsPwd := toString(proxy["obfs-password"])
			if obfsPwd == "" {
				obfsPwd = toString(proxy["obfs_password"])
			}
			out["obfs"] = map[string]any{
				"type":     obfs,
				"password": obfsPwd,
			}
		}
		attachTLS(out, proxy)
		ensureTLSEnabled(out)
		return mapToItem(out), nil
	case "http", "https":
		out := map[string]any{
			"type":        "http",
			"tag":         tag,
			"server":      server,
			"server_port": port,
		}
		if username := toString(proxy["username"]); username != "" {
			out["username"] = username
		}
		if password := toString(proxy["password"]); password != "" {
			out["password"] = password
		}
		attachTLS(out, proxy)
		if typ == "https" {
			tls := map[string]any{"enabled": true}
			if existing, ok := out["tls"].(map[string]any); ok {
				for k, v := range existing {
					tls[k] = v
				}
			}
			out["tls"] = tls
		}
		return mapToItem(out), nil
	case "socks", "socks5":
		out := map[string]any{
			"type":        "socks",
			"tag":         tag,
			"server":      server,
			"server_port": port,
		}
		if username := toString(proxy["username"]); username != "" {
			out["username"] = username
		}
		if password := toString(proxy["password"]); password != "" {
			out["password"] = password
		}
		return mapToItem(out), nil
	default:
		return nil, nil
	}
}

func parseTraditionalURIList(payload []byte) ([]OutboundItem, bool, error) {
	text := strings.TrimSpace(string(payload))
	if text == "" {
		return nil, true, nil
	}

	lines := splitSubscriptionLines(text)
	if len(lines) == 0 {
		return nil, false, nil
	}

	out := make([]OutboundItem, 0, len(lines))
	recognized := 0
	for _, line := range lines {
		item, ok := parseTraditionalURI(line)
		if !ok {
			continue
		}
		recognized++
		if item != nil {
			out = append(out, *item)
		}
	}
	if recognized == 0 {
		return nil, false, nil
	}
	return out, true, nil
}

func splitSubscriptionLines(text string) []string {
	replaced := strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(text)
	parts := strings.Split(replaced, "\n")
	lines := make([]string, 0, len(parts))
	for _, p := range parts {
		line := strings.TrimSpace(p)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) > 0 {
		return lines
	}

	// Some providers concatenate links with spaces.
	for _, p := range strings.Fields(text) {
		if strings.Contains(p, "://") {
			lines = append(lines, p)
		}
	}
	return lines
}

func parseTraditionalURI(line string) (*OutboundItem, bool) {
	lower := strings.ToLower(line)
	switch {
	case strings.HasPrefix(lower, "vmess://"):
		item, ok := parseVMessURI(line)
		return item, ok
	case strings.HasPrefix(lower, "vless://"):
		item, ok := parseVLESSURI(line)
		return item, ok
	case strings.HasPrefix(lower, "trojan://"):
		item, ok := parseTrojanURI(line)
		return item, ok
	case strings.HasPrefix(lower, "ss://"):
		item, ok := parseShadowsocksURI(line)
		return item, ok
	case strings.HasPrefix(lower, "hysteria2://") || strings.HasPrefix(lower, "hy2://"):
		item, ok := parseHysteria2URI(line)
		return item, ok
	case strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://"):
		item, ok := parseHTTPURI(line)
		return item, ok
	case strings.HasPrefix(lower, "socks://") || strings.HasPrefix(lower, "socks5://"):
		item, ok := parseSOCKSURI(line)
		return item, ok
	default:
		return nil, false
	}
}

func parseVMessURI(link string) (*OutboundItem, bool) {
	enc := strings.TrimSpace(strings.TrimPrefix(link, "vmess://"))
	decoded, ok := decodeBase64String(enc)
	if !ok {
		return nil, true
	}
	var m map[string]any
	if err := json.Unmarshal(decoded, &m); err != nil {
		return nil, true
	}

	server := toString(m["add"])
	port, _ := strconv.Atoi(toString(m["port"]))
	uuid := toString(m["id"])
	if server == "" || port <= 0 || uuid == "" {
		return nil, true
	}
	out := map[string]any{
		"type":        "vmess",
		"tag":         toString(m["ps"]),
		"server":      server,
		"server_port": port,
		"uuid":        uuid,
		"security":    orDefault(toString(m["scy"]), "auto"),
	}
	if aid, err := strconv.Atoi(toString(m["aid"])); err == nil {
		out["alter_id"] = aid
	}

	netType := strings.ToLower(toString(m["net"]))
	if netType == "ws" {
		headers := map[string]any{}
		if host := toString(m["host"]); host != "" {
			headers["Host"] = host
		}
		transport := map[string]any{
			"type": "ws",
			"path": orDefault(toString(m["path"]), "/"),
		}
		if len(headers) > 0 {
			transport["headers"] = headers
		}
		out["transport"] = transport
	}

	tlsEnabled := strings.EqualFold(toString(m["tls"]), "tls")
	if tlsEnabled {
		tls := map[string]any{"enabled": true}
		if sni := toString(m["sni"]); sni != "" {
			tls["server_name"] = sni
		}
		out["tls"] = tls
	}

	return mapToItem(out), true
}

func parseVLESSURI(link string) (*OutboundItem, bool) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, true
	}
	uuid := ""
	if u.User != nil {
		uuid = u.User.Username()
	}
	server := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if err != nil || uuid == "" || server == "" || port <= 0 {
		return nil, true
	}
	out := map[string]any{
		"type":        "vless",
		"tag":         fragmentTag(u.Fragment),
		"server":      server,
		"server_port": port,
		"uuid":        uuid,
	}
	if flow := u.Query().Get("flow"); flow != "" {
		out["flow"] = flow
	}
	attachTransportFromQuery(out, u.Query())
	attachTLSFromQuery(out, u.Query())
	return mapToItem(out), true
}

func parseTrojanURI(link string) (*OutboundItem, bool) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, true
	}
	password := ""
	if u.User != nil {
		password = u.User.Username()
	}
	server := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if err != nil || password == "" || server == "" || port <= 0 {
		return nil, true
	}
	out := map[string]any{
		"type":        "trojan",
		"tag":         fragmentTag(u.Fragment),
		"server":      server,
		"server_port": port,
		"password":    password,
	}
	attachTransportFromQuery(out, u.Query())
	attachTLSFromQuery(out, u.Query())
	ensureTLSEnabled(out)
	return mapToItem(out), true
}

func parseHysteria2URI(link string) (*OutboundItem, bool) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, true
	}
	password := ""
	if u.User != nil {
		password = u.User.Username()
	}
	server := u.Hostname()
	portStr := u.Port()
	if portStr == "" {
		portStr = "443"
	}
	port, _ := strconv.Atoi(portStr)
	if server == "" || port <= 0 {
		return nil, true
	}
	out := map[string]any{
		"type":        "hysteria2",
		"tag":         fragmentTag(u.Fragment),
		"server":      server,
		"server_port": port,
	}
	if password != "" {
		out["password"] = password
	}

	q := u.Query()
	if up := q.Get("up"); up != "" {
		if v, err := strconv.Atoi(up); err == nil {
			out["up_mbps"] = v
		}
	}
	if down := q.Get("down"); down != "" {
		if v, err := strconv.Atoi(down); err == nil {
			out["down_mbps"] = v
		}
	}
	if obfs := q.Get("obfs"); obfs != "" {
		out["obfs"] = map[string]any{
			"type":     obfs,
			"password": q.Get("obfs-password"),
		}
	}

	attachTLSFromQuery(out, q)
	ensureTLSEnabled(out)
	return mapToItem(out), true
}

func parseShadowsocksURI(link string) (*OutboundItem, bool) {
	u, err := url.Parse(link)
	if err == nil && u.Host != "" {
		method, password, ok := decodeSSUser(u.User)
		if !ok {
			return nil, true
		}
		port, err := strconv.Atoi(u.Port())
		if err != nil || port <= 0 {
			return nil, true
		}
		out := map[string]any{
			"type":        "shadowsocks",
			"tag":         fragmentTag(u.Fragment),
			"server":      u.Hostname(),
			"server_port": port,
			"method":      method,
			"password":    password,
		}
		return mapToItem(out), true
	}

	// Legacy form: ss://BASE64(method:password@host:port)#tag
	raw := strings.TrimSpace(strings.TrimPrefix(link, "ss://"))
	tag := ""
	if idx := strings.Index(raw, "#"); idx >= 0 {
		tag = fragmentTag(raw[idx+1:])
		raw = raw[:idx]
	}
	decoded, ok := decodeBase64String(raw)
	if !ok {
		return nil, true
	}
	plain := string(decoded)
	at := strings.LastIndex(plain, "@")
	if at <= 0 || at >= len(plain)-1 {
		return nil, true
	}
	cred := plain[:at]
	hostPort := plain[at+1:]
	host, portRaw, err := net.SplitHostPort(hostPort)
	if err != nil {
		return nil, true
	}
	method, password, ok := strings.Cut(cred, ":")
	if !ok || method == "" {
		return nil, true
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil || port <= 0 {
		return nil, true
	}
	out := map[string]any{
		"type":        "shadowsocks",
		"tag":         tag,
		"server":      host,
		"server_port": port,
		"method":      method,
		"password":    password,
	}
	return mapToItem(out), true
}

func parseHTTPURI(link string) (*OutboundItem, bool) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, true
	}
	server := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if err != nil || server == "" || port <= 0 {
		return nil, true
	}
	out := map[string]any{
		"type":        "http",
		"tag":         fragmentTag(u.Fragment),
		"server":      server,
		"server_port": port,
	}
	if u.User != nil {
		if username := u.User.Username(); username != "" {
			out["username"] = username
		}
		if password, ok := u.User.Password(); ok && password != "" {
			out["password"] = password
		}
	}
	if strings.EqualFold(u.Scheme, "https") {
		out["tls"] = map[string]any{"enabled": true}
	}
	return mapToItem(out), true
}

func parseSOCKSURI(link string) (*OutboundItem, bool) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, true
	}
	server := u.Hostname()
	port, err := strconv.Atoi(u.Port())
	if err != nil || server == "" || port <= 0 {
		return nil, true
	}
	out := map[string]any{
		"type":        "socks",
		"tag":         fragmentTag(u.Fragment),
		"server":      server,
		"server_port": port,
	}
	if u.User != nil {
		if username := u.User.Username(); username != "" {
			out["username"] = username
		}
		if password, ok := u.User.Password(); ok && password != "" {
			out["password"] = password
		}
	}
	return mapToItem(out), true
}

func decodeSSUser(user *url.Userinfo) (method, password string, ok bool) {
	if user == nil {
		return "", "", false
	}
	username := user.Username()
	if p, has := user.Password(); has {
		if username == "" {
			return "", "", false
		}
		return username, p, true
	}
	if decoded, ok := decodeBase64String(username); ok {
		if m, p, found := strings.Cut(string(decoded), ":"); found && m != "" {
			return m, p, true
		}
	}
	if m, p, found := strings.Cut(username, ":"); found && m != "" {
		return m, p, true
	}
	return "", "", false
}

func attachTransport(out map[string]any, proxy map[string]any) {
	netType := strings.ToLower(toString(proxy["network"]))
	switch netType {
	case "ws":
		wsOpts := mapFromAny(proxy["ws-opts"])
		if len(wsOpts) == 0 {
			wsOpts = mapFromAny(proxy["ws_opts"])
		}
		transport := map[string]any{
			"type": "ws",
			"path": "/",
		}
		if path := toString(wsOpts["path"]); path != "" {
			transport["path"] = path
		}
		headers := mapFromAny(wsOpts["headers"])
		if len(headers) > 0 {
			transport["headers"] = headers
		}
		out["transport"] = transport
	case "grpc":
		grpcOpts := mapFromAny(proxy["grpc-opts"])
		if len(grpcOpts) == 0 {
			grpcOpts = mapFromAny(proxy["grpc_opts"])
		}
		serviceName := toString(grpcOpts["grpc-service-name"])
		if serviceName == "" {
			serviceName = toString(grpcOpts["service_name"])
		}
		transport := map[string]any{"type": "grpc"}
		if serviceName != "" {
			transport["service_name"] = serviceName
		}
		out["transport"] = transport
	}
}

func attachTLS(out map[string]any, proxy map[string]any) {
	tlsEnabled := toBool(proxy["tls"])
	serverName := toString(proxy["servername"])
	if serverName == "" {
		serverName = toString(proxy["sni"])
	}
	insecure := toBool(proxy["skip-cert-verify"])
	if !tlsEnabled && serverName == "" && !insecure {
		return
	}
	tls := map[string]any{
		"enabled": tlsEnabled,
	}
	if serverName != "" {
		tls["server_name"] = serverName
	}
	if insecure {
		tls["insecure"] = true
	}
	out["tls"] = tls
}

func attachTransportFromQuery(out map[string]any, q url.Values) {
	switch strings.ToLower(q.Get("type")) {
	case "ws":
		transport := map[string]any{
			"type": "ws",
			"path": orDefault(q.Get("path"), "/"),
		}
		if host := q.Get("host"); host != "" {
			transport["headers"] = map[string]any{"Host": host}
		}
		out["transport"] = transport
	case "grpc":
		transport := map[string]any{"type": "grpc"}
		if serviceName := q.Get("serviceName"); serviceName != "" {
			transport["service_name"] = serviceName
		}
		out["transport"] = transport
	}
}

func attachTLSFromQuery(out map[string]any, q url.Values) {
	security := strings.ToLower(q.Get("security"))
	tlsEnabled := security == "tls" || security == "xtls" || security == "reality"
	insecure := strings.EqualFold(q.Get("allowInsecure"), "1") ||
		strings.EqualFold(q.Get("allowInsecure"), "true") ||
		strings.EqualFold(q.Get("insecure"), "1") ||
		strings.EqualFold(q.Get("insecure"), "true")
	serverName := q.Get("sni")
	if serverName == "" {
		serverName = q.Get("peer")
	}
	if !tlsEnabled && serverName == "" && !insecure {
		return
	}
	tls := map[string]any{
		"enabled": tlsEnabled,
	}
	if serverName != "" {
		tls["server_name"] = serverName
	}
	if insecure {
		tls["insecure"] = true
	}
	if security == "reality" {
		reality := map[string]any{"enabled": true}
		if pbk := strings.TrimSpace(q.Get("pbk")); pbk != "" {
			reality["public_key"] = pbk
		}
		if sid := strings.TrimSpace(q.Get("sid")); sid != "" {
			reality["short_id"] = sid
		}
		tls["reality"] = reality
		if fp := strings.TrimSpace(q.Get("fp")); fp != "" {
			tls["utls"] = map[string]any{
				"enabled":     true,
				"fingerprint": fp,
			}
		}
	}
	out["tls"] = tls
}

// ensureTLSEnabled guarantees the outbound has tls.enabled=true.
// Trojan protocol always requires TLS; some Clash subscriptions omit the
// explicit "tls: true" flag because it is implied for the protocol.
// Without this, sing-box panics with a nil-pointer dereference during
// TLS handshake.
func ensureTLSEnabled(out map[string]any) {
	existing, ok := out["tls"].(map[string]any)
	if ok {
		existing["enabled"] = true
		return
	}
	out["tls"] = map[string]any{"enabled": true}
}

func mapToItem(m map[string]any) *OutboundItem {
	t := strings.ToLower(toString(m["type"]))
	tag := toString(m["tag"])
	raw, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return &OutboundItem{
		Tag:  tag,
		Type: t,
		Raw:  raw,
	}
}

func decodeBase64Payload(payload []byte) ([]byte, bool) {
	raw := strings.TrimSpace(string(payload))
	if raw == "" || !looksLikeBase64(raw) {
		return nil, false
	}
	decoded, ok := decodeBase64String(raw)
	if !ok {
		return nil, false
	}
	trimmed := bytes.TrimSpace(decoded)
	if len(trimmed) == 0 {
		return nil, false
	}
	return trimmed, true
}

func decodeBase64String(raw string) ([]byte, bool) {
	clean := strings.Map(func(r rune) rune {
		switch r {
		case '\r', '\n', '\t', ' ':
			return -1
		default:
			return r
		}
	}, raw)
	if clean == "" {
		return nil, false
	}

	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	for _, enc := range encodings {
		if out, err := enc.DecodeString(clean); err == nil {
			return out, true
		}
	}
	// padding repair for standard alphabet
	if mod := len(clean) % 4; mod != 0 {
		fixed := clean + strings.Repeat("=", 4-mod)
		for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding} {
			if out, err := enc.DecodeString(fixed); err == nil {
				return out, true
			}
		}
	}
	return nil, false
}

func looksLikeBase64(s string) bool {
	if len(s) < 16 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '+', r == '/', r == '=', r == '-', r == '_', r == '\r', r == '\n':
		default:
			return false
		}
	}
	return true
}

func fragmentTag(fragment string) string {
	if fragment == "" {
		return ""
	}
	if unescaped, err := url.QueryUnescape(fragment); err == nil {
		return unescaped
	}
	return fragment
}

func mapFromAny(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func toOptionalInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(x))
		if err == nil {
			return i, true
		}
	}
	return 0, false
}

func toInt(v any) int {
	if i, ok := toOptionalInt(v); ok {
		return i
	}
	return 0
}

func toBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		x = strings.ToLower(strings.TrimSpace(x))
		return x == "1" || x == "true" || x == "yes" || x == "on"
	case int:
		return x != 0
	case int64:
		return x != 0
	case float64:
		return x != 0
	default:
		return false
	}
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatInt(int64(x), 10)
	default:
		return ""
	}
}

func orDefault(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
