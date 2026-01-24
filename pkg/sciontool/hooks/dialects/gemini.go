/*
Copyright 2025 The Scion Authors.
*/

package dialects

import (
	"github.com/ptone/scion-agent/pkg/sciontool/hooks"
)

// GeminiDialect parses Gemini CLI hook events.
type GeminiDialect struct{}

// NewGeminiDialect creates a new Gemini dialect parser.
func NewGeminiDialect() *GeminiDialect {
	return &GeminiDialect{}
}

// Name returns the dialect name.
func (d *GeminiDialect) Name() string {
	return "gemini"
}

// Parse converts Gemini CLI event format to normalized Event.
//
// Gemini CLI sends events with the following format:
//
//	{
//	  "hook_event_name": "BeforeAgent" | "AfterAgent" | "BeforeTool" | etc.,
//	  "tool_name": "...",
//	  "prompt": "...",
//	  ...
//	}
func (d *GeminiDialect) Parse(data map[string]interface{}) (*hooks.Event, error) {
	rawName := getString(data, "hook_event_name")
	if rawName == "" {
		rawName = getString(data, "event")
	}

	event := &hooks.Event{
		Name:    d.normalizeEventName(rawName),
		RawName: rawName,
		Dialect: "gemini",
		Data: hooks.EventData{
			Prompt:   getString(data, "prompt"),
			ToolName: getString(data, "tool_name"),
			Message:  getString(data, "message"),
			Reason:   getString(data, "reason"),
			Source:   getString(data, "source"),
			Raw:      data,
		},
	}

	return event, nil
}

// normalizeEventName maps Gemini CLI event names to normalized names.
func (d *GeminiDialect) normalizeEventName(name string) string {
	switch name {
	case "SessionStart":
		return hooks.EventSessionStart
	case "SessionEnd":
		return hooks.EventSessionEnd
	case "BeforeAgent":
		return hooks.EventAgentStart
	case "AfterAgent":
		return hooks.EventAgentEnd
	case "BeforeTool":
		return hooks.EventToolStart
	case "AfterTool":
		return hooks.EventToolEnd
	case "BeforeModel":
		return hooks.EventModelStart
	case "AfterModel":
		return hooks.EventModelEnd
	case "Notification":
		return hooks.EventNotification
	default:
		return name
	}
}
