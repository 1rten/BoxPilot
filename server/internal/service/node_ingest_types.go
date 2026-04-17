package service

import (
	"encoding/json"

	"boxpilot/server/internal/store/repo"
)

type IngestSource string

const (
	IngestSourceManualURI  IngestSource = "manual_uri"
	IngestSourceManualJSON IngestSource = "manual_json"
	IngestSourceManualForm IngestSource = "manual_form"
	IngestSourceSub        IngestSource = "subscription"
)

type IngestMode string

const (
	IngestModeAppend  IngestMode = "append"
	IngestModeReplace IngestMode = "replace"
)

type IngestNode struct {
	SourceTag     string
	PreferredTag  string
	PreferredName string
	Type          string
	Raw           json.RawMessage
}

type IngestInput struct {
	SubID                    string
	Source                   IngestSource
	Mode                     IngestMode
	TagPrefix                string
	DefaultEnabled           int
	DefaultForwardingEnabled int
	Nodes                    []IngestNode
}

type IngestResult struct {
	Source      IngestSource
	Mode        IngestMode
	SubID       string
	Created     int
	Rows        []repo.NodeRow
	SourceTagTo map[string]string
}
