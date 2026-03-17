package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/notuselessdev/grove-street/internal/hooks"
)

func TestSoundToPhrase(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{"basic", "ah_shit_here_we_go_again.mp3", "Ah shit here we go again"},
		{"wav extension", "grove_street_home.wav", "Grove street home"},
		{"ogg extension", "piece_of_cake.ogg", "Piece of cake"},
		{"single word", "easy.mp3", "Easy"},
		{"no extension", "test_file", "Test file"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := soundToPhrase(tt.filename)
			if got != tt.want {
				t.Errorf("soundToPhrase(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsAudio(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"wav", "test.wav", true},
		{"mp3", "test.mp3", true},
		{"ogg", "test.ogg", true},
		{"WAV upper", "test.WAV", true},
		{"txt", "test.txt", false},
		{"no ext", "testfile", false},
		{"flac", "test.flac", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAudio(tt.filename)
			if got != tt.want {
				t.Errorf("isAudio(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestNormalizeEvent(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		payload  string
		wantType string
	}{
		{
			"cursor stop",
			"cursor",
			`{"event": "stop", "stop_reason": "end_turn"}`,
			"Stop",
		},
		{
			"windsurf cascade",
			"windsurf",
			`{"event": "post_cascade_response"}`,
			"Stop",
		},
		{
			"copilot session",
			"copilot",
			`{"event": "sessionStart"}`,
			"SessionStart",
		},
		{
			"kiro agent spawn",
			"kiro",
			`{"event": "agentSpawn"}`,
			"SessionStart",
		},
		{
			"unknown source falls back",
			"unknown",
			`{"type": "Stop", "stop_reason": "end_turn"}`,
			"Stop",
		},
		{
			"invalid json",
			"cursor",
			`not json`,
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := normalizeEvent(tt.source, []byte(tt.payload))
			if event.Type != tt.wantType {
				t.Errorf("normalizeEvent(%q, ...).Type = %q, want %q", tt.source, event.Type, tt.wantType)
			}
		})
	}
}

func TestMapCursorEvent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"stop", "Stop"},
		{"beforeShellExecution", "PermissionRequest"},
		{"beforeMCPExecution", "PermissionRequest"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapCursorEvent(tt.input)
			if got != tt.want {
				t.Errorf("mapCursorEvent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapWindsurfEvent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"post_cascade_response", "Stop"},
		{"pre_user_prompt", "Notification"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapWindsurfEvent(tt.input)
			if got != tt.want {
				t.Errorf("mapWindsurfEvent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapCopilotEvent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"sessionStart", "SessionStart"},
		{"postToolUse", "Stop"},
		{"errorOccurred", "Notification"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapCopilotEvent(tt.input)
			if got != tt.want {
				t.Errorf("mapCopilotEvent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMapKiroEvent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"agentSpawn", "SessionStart"},
		{"stop", "Stop"},
		{"userPromptSubmit", "Notification"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapKiroEvent(tt.input)
			if got != tt.want {
				t.Errorf("mapKiroEvent(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDetectParentApp(t *testing.T) {
	// Default should return some bundle ID
	got, _ := detectParentApp()
	if got == "" {
		t.Error("detectParentApp() returned empty string")
	}
}

func TestContainsGroveStreet(t *testing.T) {
	tests := []struct {
		name string
		hook interface{}
		want bool
	}{
		{
			"flat command match",
			map[string]interface{}{"command": "/usr/bin/grove-street hook"},
			true,
		},
		{
			"nested hooks match",
			map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{"command": "grove-street hook --event Stop"},
				},
			},
			true,
		},
		{
			"no match",
			map[string]interface{}{"command": "some-other-tool"},
			false,
		},
		{
			"not a map",
			"string value",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsGroveStreet(tt.hook)
			if got != tt.want {
				t.Errorf("containsGroveStreet(%v) = %v, want %v", tt.hook, got, tt.want)
			}
		})
	}
}

func TestClaimNotificationSlot(t *testing.T) {
	tmpDir := t.TempDir()
	slotDir := filepath.Join(tmpDir, ".notification-slots")
	os.MkdirAll(slotDir, 0755)

	// First claim should get slot 0
	idx, file := claimNotificationSlot()
	if idx != 0 {
		t.Errorf("first claim got slot %d, want 0", idx)
	}
	if file == "" {
		t.Error("first claim returned empty file")
	}

	// Clean up
	os.Remove(file)
}

func TestAcquireCooldown(t *testing.T) {
	// Remove any existing cooldown file so the test starts clean
	cooldownFile := filepath.Join(os.Getenv("HOME"), ".grove-street", ".last-notification")
	os.Remove(cooldownFile)

	// First call should succeed
	if !acquireCooldown() {
		t.Error("first acquireCooldown() should return true")
	}

	// Immediate second call should be blocked (within 2s window)
	if acquireCooldown() {
		t.Error("immediate second acquireCooldown() should return false")
	}

	// Clean up
	os.Remove(cooldownFile)
}

func TestIsProcessAlive(t *testing.T) {
	// Our own PID should be alive
	myPid := fmt.Sprintf("%d", os.Getpid())
	if !isProcessAlive(myPid) {
		t.Error("our own PID should be alive")
	}

	// A very high PID should not be alive
	if isProcessAlive("9999999") {
		t.Error("PID 9999999 should not be alive")
	}

	// Invalid PID string
	if isProcessAlive("notapid") {
		t.Error("invalid PID should not be alive")
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	if !dirExists(tmpDir) {
		t.Errorf("dirExists(%q) = false, want true", tmpDir)
	}
	if dirExists(filepath.Join(tmpDir, "nonexistent")) {
		t.Error("dirExists for nonexistent dir should be false")
	}
}

func TestEndToEndClassification(t *testing.T) {
	// Test that IDE-specific events properly map through normalizeEvent -> Classify
	tests := []struct {
		name     string
		source   string
		payload  string
		wantCat  string
	}{
		{
			"cursor stop end_turn -> task_complete",
			"cursor",
			`{"event": "stop", "stop_reason": "end_turn"}`,
			"task_complete",
		},
		{
			"copilot session start -> session_start",
			"copilot",
			`{"event": "sessionStart"}`,
			"session_start",
		},
		{
			"kiro stop -> task_complete",
			"kiro",
			`{"event": "stop", "stop_reason": "end_turn"}`,
			"task_complete",
		},
		{
			"windsurf response -> task_complete",
			"windsurf",
			`{"event": "post_cascade_response", "stop_reason": "end_turn"}`,
			"task_complete",
		},
		{
			"copilot error -> task_error",
			"copilot",
			`{"event": "errorOccurred", "message": "Something failed"}`,
			"task_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := normalizeEvent(tt.source, []byte(tt.payload))
			cat := hooks.Classify(event)
			if cat != tt.wantCat {
				t.Errorf("classify(normalize(%q)) = %q, want %q", tt.source, cat, tt.wantCat)
			}
		})
	}
}

func TestRegisterJSONHooksRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Register hooks
	err := registerJSONHooksPerEvent(settingsPath, "hooks", "grove-street hook", []string{"Stop", "SessionStart"})
	if err != nil {
		t.Fatal(err)
	}

	// Read back
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}

	hooksMap, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks key missing or not a map")
	}

	for _, event := range []string{"Stop", "SessionStart"} {
		if _, ok := hooksMap[event]; !ok {
			t.Errorf("hooks[%q] missing", event)
		}
	}

	// Unregister and verify
	unregisterJSONHooks(settingsPath, "hooks")

	data, _ = os.ReadFile(settingsPath)
	var afterUnregister map[string]interface{}
	json.Unmarshal(data, &afterUnregister)
	if _, ok := afterUnregister["hooks"]; ok {
		t.Error("hooks should be removed after unregister")
	}
}
