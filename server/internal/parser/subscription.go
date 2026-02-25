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

var filterTypes = map[string]bool{
	"direct": true, "block": true, "dns": true, "selector": true, "urltest": true,
}

// ParseSubscription auto-detects the payload format and converts supported subscriptions
// into sing-box outbounds.
func ParseSubscription(body []byte) ([]OutboundItem, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, errorx.New(errorx.SUBParseFailed, "empty subscription body")
	}

	if out, ok, err := parseSingboxJSON(trimmed); ok {
		return finalizeParsed(out, err, "singbox_json")
	}
	if out, ok, err := parseClashYAML(trimmed); ok {
		return finalizeParsed(out, err, "clash_yaml")
	}
	if out, ok, err := parseTraditionalURIList(trimmed); ok {
		return finalizeParsed(out, err, "traditional_uri")
	}

	if decoded, ok := decodeBase64Payload(trimmed); ok {
		if out, parsed, err := parseSingboxJSON(decoded); parsed {
			return finalizeParsed(out, err, "singbox_base64")
		}
		if out, parsed, err := parseClashYAML(decoded); parsed {
			return finalizeParsed(out, err, "clash_base64")
		}
		if out, parsed, err := parseTraditionalURIList(decoded); parsed {
			return finalizeParsed(out, err, "traditional_base64")
		}
	}

	return nil, errorx.New(errorx.SUBFormatUnsupported, "subscription format unsupported")
}

func finalizeParsed(out []OutboundItem, parseErr error, format string) ([]OutboundItem, error) {
	if parseErr != nil {
		return nil, parseErr
	}
	if len(out) == 0 {
		return nil, errorx.New(errorx.SUBEmptyOutbounds, "no supported outbounds found").WithDetails(map[string]any{
			"format": format,
		})
	}
	return out, nil
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
	out["tls"] = tls
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
