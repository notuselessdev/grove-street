package hooks

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		name     string
		event    Event
		expected string
	}{
		// Session start
		{"session start", Event{Type: "SessionStart"}, "session_start"},

		// Stop events
		{"stop end_turn", Event{Type: "Stop", StopReason: "end_turn"}, "task_complete"},
		{"stop stop_sequence", Event{Type: "Stop", StopReason: "stop_sequence"}, "task_complete"},
		{"stop tool_use", Event{Type: "Stop", StopReason: "tool_use"}, "input_required"},
		{"stop error", Event{Type: "Stop", StopReason: "error"}, "task_error"},
		{"stop failure", Event{Type: "Stop", StopReason: "some_failure"}, "task_error"},
		{"stop unknown defaults to complete", Event{Type: "Stop", StopReason: "something"}, "task_complete"},

		// SubagentStop mirrors Stop
		{"subagent stop end_turn", Event{Type: "SubagentStop", StopReason: "end_turn"}, "task_complete"},
		{"subagent stop error", Event{Type: "SubagentStop", StopReason: "error"}, "task_error"},

		// Notifications
		{"notification error", Event{Type: "Notification", Message: "Something failed"}, "task_error"},
		{"notification permission", Event{Type: "Notification", Message: "Needs permission"}, "input_required"},
		{"notification compact", Event{Type: "Notification", Message: "context limit"}, "resource_limit"},
		{"notification default", Event{Type: "Notification", Message: "something else"}, "input_required"},

		// PreCompact
		{"precompact", Event{Type: "PreCompact"}, "resource_limit"},

		// PermissionRequest
		{"permission request", Event{Type: "PermissionRequest"}, "input_required"},

		// Event field fallback
		{"event field fallback", Event{Event: "Stop", StopReason: "end_turn"}, "task_complete"},

		// Unknown
		{"unknown type", Event{Type: "Unknown"}, ""},
		{"empty event", Event{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.event)
			if got != tt.expected {
				t.Errorf("Classify(%+v) = %q, want %q", tt.event, got, tt.expected)
			}
		})
	}
}
