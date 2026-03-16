package hooks

import "strings"

// Event represents a Claude Code hook event payload.
type Event struct {
	Type         string `json:"type"`
	Event        string `json:"event"`
	SessionID    string `json:"session_id"`
	StopReason   string `json:"stop_reason"`
	Message      string `json:"message"`
	Notification string `json:"notification"`
}

// Classify maps a Claude Code hook event to a sound category.
func Classify(e Event) string {
	hookType := e.Type
	if hookType == "" {
		hookType = e.Event
	}

	switch hookType {
	case "SessionStart":
		return "session_start"

	case "Stop", "SubagentStop":
		reason := strings.ToLower(e.StopReason)
		switch {
		case reason == "end_turn" || reason == "stop_sequence":
			return "task_complete"
		case reason == "tool_use":
			return "input_required"
		case strings.Contains(reason, "error") || strings.Contains(reason, "fail"):
			return "task_error"
		default:
			return "task_complete"
		}

	case "Notification":
		msg := strings.ToLower(e.Message + e.Notification)
		switch {
		case strings.Contains(msg, "error") || strings.Contains(msg, "fail"):
			return "task_error"
		case strings.Contains(msg, "permission") || strings.Contains(msg, "approve"):
			return "input_required"
		case strings.Contains(msg, "compact") || strings.Contains(msg, "context"):
			return "resource_limit"
		default:
			return "input_required"
		}

	case "PreCompact":
		return "resource_limit"

	case "PermissionRequest":
		return "input_required"

	default:
		return ""
	}
}
