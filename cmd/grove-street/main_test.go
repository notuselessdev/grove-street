package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/notuselessdev/grove-street/internal/config"
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

func TestCategoryLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"task_complete", "Task Complete"},
		{"task_error", "Task Error"},
		{"input_required", "Input Required"},
		{"resource_limit", "Resource Limit"},
		{"session_start", "Session Start"},
		{"user_spam", "Chill Out"},
		{"unknown", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := categoryLabel(tt.input)
			if got != tt.want {
				t.Errorf("categoryLabel(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildNotifyArgs(t *testing.T) {
	args := buildNotifyArgs(
		"Ah shit here we go again",
		"/home/user/.grove-street/icon.png",
		"my-project",
		"top-right",
		"/home/user/.grove-street/.notification-slots",
		2,
		7.0,
		"task_complete",
	)

	// Arg count must match what all three notify scripts expect
	if len(args) != 9 {
		t.Fatalf("buildNotifyArgs returned %d args, want 9", len(args))
	}

	checks := []struct {
		idx  int
		name string
		want string
	}{
		{0, "sender", "Carl Johnson"},
		{1, "phrase", "Ah shit here we go again"},
		{2, "iconPath", "/home/user/.grove-street/icon.png"},
		{3, "duration", "7.0"},
		{4, "projectName", "my-project"},
		{5, "position", "top-right"},
		{6, "slotIndex", "2"},
		{7, "slotDir", "/home/user/.grove-street/.notification-slots"},
		{8, "categoryLabel", "Task Complete"},
	}

	for _, c := range checks {
		if args[c.idx] != c.want {
			t.Errorf("args[%d] (%s) = %q, want %q", c.idx, c.name, args[c.idx], c.want)
		}
	}
}

func TestBuildNotifyArgsDefaults(t *testing.T) {
	// Empty category should produce empty label
	args := buildNotifyArgs("phrase", "", "proj", "top-right", "", 0, 4.0, "")
	if args[8] != "" {
		t.Errorf("args[8] (categoryLabel) = %q, want empty for unknown category", args[8])
	}
}

func TestStopResume(t *testing.T) {
	// Use a temp dir to isolate config
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", tmpDir)
	}

	// Ensure config dir exists
	os.MkdirAll(filepath.Join(tmpDir, ".grove-street"), 0755)

	// Start with default config (enabled=true)
	cfg := config.DefaultConfig()
	config.Save(cfg)

	// Stop should disable
	cfg = config.Load()
	cfg.Enabled = false
	config.Save(cfg)

	cfg = config.Load()
	if cfg.Enabled {
		t.Error("after stop, config.Enabled should be false")
	}

	// Resume should re-enable
	cfg.Enabled = true
	config.Save(cfg)

	cfg = config.Load()
	if !cfg.Enabled {
		t.Error("after resume, config.Enabled should be true")
	}
}

func TestCheckJSONHooksPerEvent(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// No file → should report missing
	issues, err := checkJSONHooksPerEvent(settingsPath, "hooks", "grove-street hook", []string{"Stop"})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) == 0 {
		t.Error("expected issues for missing file")
	}

	// Register hooks, then check — should be clean
	registerJSONHooksPerEvent(settingsPath, "hooks", "grove-street hook", []string{"Stop", "SessionStart"})

	issues, err = checkJSONHooksPerEvent(settingsPath, "hooks", "grove-street hook", []string{"Stop", "SessionStart"})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}

	// Check with extra expected event → should report missing
	issues, err = checkJSONHooksPerEvent(settingsPath, "hooks", "grove-street hook", []string{"Stop", "SessionStart", "Notification"})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue for missing Notification, got %d: %v", len(issues), issues)
	}
}

func TestCheckJSONHooks(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "hooks.json")

	// Register hooks
	registerJSONHooks(settingsPath, "hooks", "grove-street hook --source cursor", []string{"stop", "beforeShellExecution"})

	// All present → no issues
	issues, err := checkJSONHooks(settingsPath, "hooks", "grove-street hook --source cursor", []string{"stop", "beforeShellExecution"})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}

	// Wrong binary path → should flag issues
	issues, err = checkJSONHooks(settingsPath, "hooks", "/new/path/grove-street hook --source cursor", []string{"stop", "beforeShellExecution"})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 2 {
		t.Errorf("expected 2 wrong-path issues, got %d: %v", len(issues), issues)
	}
}

func TestCheckKiroHooks(t *testing.T) {
	tmpDir := t.TempDir()
	agentsDir := filepath.Join(tmpDir, ".kiro", "agents")
	os.MkdirAll(agentsDir, 0755)

	// No file → should report missing
	// Temporarily override kiroConfigDir by writing directly
	agentPath := filepath.Join(agentsDir, "grove-street.json")

	// Write valid config
	kiroConfig := map[string]interface{}{
		"name": "grove-street",
		"hooks": map[string]interface{}{
			"agentSpawn":       []interface{}{map[string]interface{}{"command": "grove-street hook --source kiro"}},
			"stop":             []interface{}{map[string]interface{}{"command": "grove-street hook --source kiro"}},
			"userPromptSubmit": []interface{}{map[string]interface{}{"command": "grove-street hook --source kiro"}},
		},
	}
	data, _ := json.MarshalIndent(kiroConfig, "", "  ")
	os.WriteFile(agentPath, data, 0644)

	// Check with correct binary → should be clean
	// We need to call the underlying logic directly since checkKiroHooks uses kiroConfigDir()
	issues := checkKiroAgentFile(agentPath, "grove-street hook --source kiro", []string{"agentSpawn", "stop", "userPromptSubmit"})
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}

	// Check with wrong binary → should flag issues
	issues = checkKiroAgentFile(agentPath, "/new/path/grove-street hook --source kiro", []string{"agentSpawn", "stop", "userPromptSubmit"})
	if len(issues) != 3 {
		t.Errorf("expected 3 wrong-path issues, got %d: %v", len(issues), issues)
	}
}

func TestHookEntryHasCmd(t *testing.T) {
	tests := []struct {
		name string
		hook interface{}
		cmd  string
		want bool
	}{
		{
			"nested match",
			map[string]interface{}{
				"matcher": "",
				"hooks": []interface{}{
					map[string]interface{}{"type": "command", "command": "grove-street hook --event Stop"},
				},
			},
			"grove-street hook --event Stop",
			true,
		},
		{
			"nested mismatch",
			map[string]interface{}{
				"hooks": []interface{}{
					map[string]interface{}{"command": "/old/path/grove-street hook --event Stop"},
				},
			},
			"grove-street hook --event Stop",
			false,
		},
		{
			"flat match",
			map[string]interface{}{"command": "grove-street hook"},
			"grove-street hook",
			true,
		},
		{
			"not a map",
			"string",
			"grove-street hook",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hookEntryHasCmd(tt.hook, tt.cmd)
			if got != tt.want {
				t.Errorf("hookEntryHasCmd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegisterJSONHooksPerEventFlat(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	err := registerJSONHooksPerEventFlat(settingsPath, "hooks", "grove-street hook", []string{"Stop", "SessionStart"})
	if err != nil {
		t.Fatal(err)
	}

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
		arr, ok := hooksMap[event].([]interface{})
		if !ok || len(arr) == 0 {
			t.Fatalf("hooks[%q] missing or empty", event)
		}
		entry, ok := arr[0].(map[string]interface{})
		if !ok {
			t.Fatalf("hooks[%q][0] is not a map", event)
		}
		// Flat format should have "command" directly, not nested "hooks"
		if _, hasNested := entry["hooks"]; hasNested {
			t.Errorf("hooks[%q] should use flat format, but has nested 'hooks' key", event)
		}
		cmd, ok := entry["command"].(string)
		if !ok || cmd == "" {
			t.Errorf("hooks[%q] missing flat 'command' field", event)
		}
	}

	// Unregister and verify cleanup works for flat format too
	unregisterJSONHooks(settingsPath, "hooks")
	data, _ = os.ReadFile(settingsPath)
	var afterUnregister map[string]interface{}
	json.Unmarshal(data, &afterUnregister)
	if _, ok := afterUnregister["hooks"]; ok {
		t.Error("hooks should be removed after unregister")
	}
}

func TestHookArrayContainsCmd(t *testing.T) {
	arr := []interface{}{
		map[string]interface{}{"command": "grove-street hook --source kiro"},
	}

	if !hookArrayContainsCmd(arr, "grove-street hook --source kiro") {
		t.Error("should find matching command")
	}
	if hookArrayContainsCmd(arr, "/other/grove-street hook --source kiro") {
		t.Error("should not match different path")
	}
}
