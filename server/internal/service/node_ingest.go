package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"boxpilot/server/internal/parser"
	"boxpilot/server/internal/store/repo"
	"boxpilot/server/internal/util"
	"boxpilot/server/internal/util/errorx"
)

func IngestOutbounds(db *sql.DB, input IngestInput) (*IngestResult, *errorx.AppError) {
	if strings.TrimSpace(input.SubID) == "" {
		return nil, errorx.New(errorx.REQMissingField, "sub_id required")
	}
	if input.Mode != IngestModeAppend && input.Mode != IngestModeReplace {
		return nil, errorx.New(errorx.REQInvalidField, "ingest mode must be append/replace")
	}
	if len(input.Nodes) == 0 {
		return nil, errorx.New(errorx.NODEInvalidOutbound, "no ingest nodes")
	}
	if input.DefaultEnabled == 0 {
		input.DefaultEnabled = 1
	}
	if input.DefaultForwardingEnabled == 0 {
		input.DefaultForwardingEnabled = 1
	}

	existingTags, err := listAllNodeTags(db)
	if err != nil {
		return nil, errorx.New(errorx.DBError, "list node tags").WithDetails(map[string]any{"err": err.Error()})
	}

	used := make(map[string]struct{}, len(existingTags))
	for _, tag := range existingTags {
		used[tag] = struct{}{}
	}

	now := util.NowRFC3339()
	rows := make([]repo.NodeRow, 0, len(input.Nodes))
	sourceMap := make(map[string]string, len(input.Nodes))

	for idx, node := range input.Nodes {
		rawType := strings.ToLower(strings.TrimSpace(node.Type))
		if rawType == "" {
			return nil, errorx.New(errorx.NODEInvalidOutbound, "node type is empty")
		}
		if !json.Valid(node.Raw) {
			return nil, errorx.New(errorx.NODEInvalidOutbound, "node outbound is not valid json")
		}
		tag := resolveIngestTag(node, idx, input.TagPrefix, used)
		if tag == "" {
			return nil, errorx.New(errorx.NODEInvalidOutbound, "node tag is empty")
		}
		if _, exists := used[tag]; exists && input.Mode == IngestModeAppend {
			return nil, errorx.New(errorx.NODETagConflict, "node tag already exists").WithDetails(map[string]any{"tag": tag})
		}
		used[tag] = struct{}{}

		name := strings.TrimSpace(node.PreferredName)
		if name == "" {
			name = tag
		}
		outJSON, mErr := mergeOutboundJSONTag(node.Raw, tag)
		if mErr != nil {
			return nil, mErr
		}
		rows = append(rows, repo.NodeRow{
			ID:                util.NewID(),
			SubID:             input.SubID,
			Tag:               tag,
			Name:              name,
			Type:              rawType,
			Enabled:           input.DefaultEnabled,
			ForwardingEnabled: input.DefaultForwardingEnabled,
			OutboundJSON:      outJSON,
			CreatedAt:         now,
		})
		if src := strings.TrimSpace(node.SourceTag); src != "" {
			sourceMap[src] = tag
		}
	}

	switch input.Mode {
	case IngestModeAppend:
		for _, row := range rows {
			if err := repo.CreateNode(db, row); err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "unique") {
					return nil, errorx.New(errorx.NODETagConflict, "node tag already exists").WithDetails(map[string]any{"tag": row.Tag})
				}
				return nil, errorx.New(errorx.DBError, "create node").WithDetails(map[string]any{"err": err.Error()})
			}
		}
	case IngestModeReplace:
		if err := repo.ReplaceNodesForSubscription(db, input.SubID, rows); err != nil {
			return nil, errorx.New(errorx.SUBReplaceNodesFailed, "replace nodes").WithDetails(map[string]any{
				"sub_id": input.SubID,
				"err":    err.Error(),
			})
		}
	}

	return &IngestResult{
		Source:      input.Source,
		Mode:        input.Mode,
		SubID:       input.SubID,
		Created:     len(rows),
		Rows:        rows,
		SourceTagTo: sourceMap,
	}, nil
}

func BuildIngestNodesFromOutbounds(outbounds []parser.OutboundItem, tagPrefix string) []IngestNode {
	nodes := make([]IngestNode, 0, len(outbounds))
	for _, item := range outbounds {
		preferredTag := strings.TrimSpace(item.Tag)
		if preferredTag == "" {
			var payload map[string]any
			if err := json.Unmarshal(item.Raw, &payload); err == nil {
				if rawTag, ok := payload["tag"].(string); ok {
					preferredTag = strings.TrimSpace(rawTag)
				}
			}
		}
		nodes = append(nodes, IngestNode{
			SourceTag:     strings.TrimSpace(item.Tag),
			PreferredTag:  preferredTag,
			PreferredName: preferredTag,
			Type:          strings.ToLower(strings.TrimSpace(item.Type)),
			Raw:           item.Raw,
		})
	}
	return nodes
}

func resolveIngestTag(node IngestNode, idx int, tagPrefix string, used map[string]struct{}) string {
	tag := strings.TrimSpace(node.PreferredTag)
	if tag == "" {
		base := strings.TrimSpace(tagPrefix)
		if base == "" {
			base = fmt.Sprintf("manual-%s", strings.TrimSpace(node.Type))
			if base == "manual-" {
				base = "manual-node"
			}
		}
		tag = resolveUniqueIngestTag(base, used, idx)
	}
	return tag
}

func resolveUniqueIngestTag(base string, used map[string]struct{}, idx int) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "node"
	}
	if _, exists := used[base]; !exists {
		return base
	}
	i := 2
	if idx > 0 {
		i = idx + 1
	}
	for {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, exists := used[candidate]; !exists {
			return candidate
		}
		i++
	}
}

func listAllNodeTags(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT tag FROM nodes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := make([]string, 0, 128)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// mergeOutboundJSONTag sets the outbound "tag" field to match the resolved DB tag.
// Selector/route logic keys off DB tag; sing-box uses the JSON tag, so they must agree.
func mergeOutboundJSONTag(raw []byte, tag string) (string, *errorx.AppError) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return "", errorx.New(errorx.NODEInvalidOutbound, "merge outbound tag: tag is empty")
	}
	if !json.Valid(raw) {
		return "", errorx.New(errorx.NODEInvalidOutbound, "merge outbound tag: invalid json")
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", errorx.New(errorx.NODEInvalidOutbound, "merge outbound tag: unmarshal failed")
	}
	payload["tag"] = tag
	out, err := json.Marshal(payload)
	if err != nil {
		return "", errorx.New(errorx.NODEInvalidOutbound, "merge outbound tag: marshal failed")
	}
	return string(out), nil
}
