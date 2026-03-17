package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/notuselessdev/grove-street/internal/config"
	"github.com/notuselessdev/grove-street/internal/hooks"
	"github.com/notuselessdev/grove-street/internal/player"
	"github.com/notuselessdev/grove-street/internal/updater"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "hook":
		cmdHook()
	case "setup":
		cmdSetup()
	case "play":
		cmdPlay()
	case "list":
		cmdList()
	case "update":
		cmdUpdate()
	case "uninstall":
		cmdUninstall()
	case "version", "--version", "-v":
		fmt.Println("grove-street v" + version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "grove-street: unknown command %q\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// cmdHook reads a JSON event from stdin, classifies it, and plays a sound.
func cmdHook() {
	cfg := config.Load()
	if !cfg.Enabled {
		return
	}

	// Parse flags
	source := ""
	eventType := ""
	for i := 2; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--source" {
			source = os.Args[i+1]
		}
		if os.Args[i] == "--event" {
			eventType = os.Args[i+1]
		}
	}
	// Also check if --event is the last arg (no value after it would be caught above)
	if len(os.Args) >= 4 && os.Args[len(os.Args)-2] == "--event" {
		eventType = os.Args[len(os.Args)-1]
	}

	raw, _ := io.ReadAll(os.Stdin)

	var event hooks.Event
	if len(raw) > 0 {
		if source == "" || source == "claude" {
			json.Unmarshal(raw, &event)
		} else {
			event = normalizeEvent(source, raw)
		}
	}

	// --event flag overrides whatever was (or wasn't) in the JSON
	if eventType != "" {
		event.Type = eventType
	}

	category := hooks.Classify(event)
	if category == "" {
		return
	}

	path := player.Pick(category, cfg)
	if path == "" {
		return
	}
	player.Play(path, cfg.Volume)
	notify(filepath.Base(path), cfg)
}

// normalizeEvent converts IDE-specific JSON payloads into a hooks.Event.
// This is a minimal inline implementation; the full version lives in internal/ides.
func normalizeEvent(source string, raw []byte) hooks.Event {
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return hooks.Event{}
	}

	str := func(key string) string {
		if v, ok := data[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	switch source {
	case "cursor":
		// Cursor events: "stop", "beforeShellExecution", "beforeMCPExecution"
		return hooks.Event{Type: mapCursorEvent(str("event")), StopReason: str("stop_reason")}
	case "windsurf":
		return hooks.Event{Type: mapWindsurfEvent(str("event")), StopReason: str("stop_reason")}
	case "copilot":
		return hooks.Event{Type: mapCopilotEvent(str("event")), StopReason: str("stop_reason"), Message: str("message")}
	case "kiro":
		return hooks.Event{Type: mapKiroEvent(str("event")), StopReason: str("stop_reason")}
	default:
		// Unknown source — try parsing as Claude Code format
		var event hooks.Event
		json.Unmarshal(raw, &event)
		return event
	}
}

func mapCursorEvent(event string) string {
	switch event {
	case "stop":
		return "Stop"
	case "beforeShellExecution", "beforeMCPExecution":
		return "PermissionRequest"
	default:
		return event
	}
}

func mapWindsurfEvent(event string) string {
	switch event {
	case "post_cascade_response":
		return "Stop"
	case "pre_user_prompt":
		return "Notification"
	default:
		return event
	}
}

func mapCopilotEvent(event string) string {
	switch event {
	case "sessionStart":
		return "SessionStart"
	case "postToolUse":
		return "Stop"
	case "errorOccurred":
		return "Notification"
	default:
		return event
	}
}

func mapKiroEvent(event string) string {
	switch event {
	case "agentSpawn":
		return "SessionStart"
	case "stop":
		return "Stop"
	case "userPromptSubmit":
		return "Notification"
	default:
		return event
	}
}

// cmdSetup registers hooks, creates sound directories, and writes default config.
func cmdSetup() {
	// Parse --ide flag
	targetIDE := ""
	for i := 2; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--ide" {
			targetIDE = os.Args[i+1]
		}
	}

	// Ensure sound directories exist
	categories := []string{"session_start", "task_complete", "task_error", "input_required", "resource_limit", "user_spam"}
	for _, cat := range categories {
		os.MkdirAll(filepath.Join(config.SoundsDir(), cat), 0755)
	}

	// Install assets if missing
	installIcon()
	installOverlayScript()

	// Write default config if missing
	if _, err := os.Stat(config.ConfigPath()); os.IsNotExist(err) {
		config.Save(config.DefaultConfig())
		fmt.Println("[CJ] Default config written to", config.ConfigPath())
	}

	// Find binary path — prefer the symlink (e.g., /opt/homebrew/bin/grove-street)
	// over the resolved Cellar path, so hooks survive brew upgrades.
	binPath, err := os.Executable()
	if err != nil {
		binPath = "grove-street"
	}
	// Don't resolve symlinks — keep the stable /opt/homebrew/bin/ path

	// Register hooks for IDEs
	type ideInfo struct {
		name      string
		configDir string
		register  func(string) error
	}

	allIDEs := []ideInfo{
		{"Claude Code", claudeConfigDir(), func(bin string) error { return registerClaudeHooks(bin) }},
		{"Cursor", cursorConfigDir(), func(bin string) error { return registerCursorHooks(bin) }},
		{"Windsurf", windsurfConfigDir(), func(bin string) error { return registerWindsurfHooks(bin) }},
		{"GitHub Copilot", copilotConfigDir(), func(bin string) error { return registerCopilotHooks(bin) }},
		{"Kiro", kiroConfigDir(), func(bin string) error { return registerKiroHooks(bin) }},
	}

	for _, ide := range allIDEs {
		if targetIDE != "" && !strings.EqualFold(ide.name, targetIDE) {
			continue
		}
		if targetIDE == "" && !dirExists(ide.configDir) {
			continue
		}
		if err := ide.register(binPath); err != nil {
			fmt.Fprintf(os.Stderr, "[CJ] Failed to register hooks for %s: %v\n", ide.name, err)
		} else {
			fmt.Printf("[CJ] Hooks registered for %s\n", ide.name)
		}
	}

	fmt.Println()
	fmt.Println("[CJ] Grove Street. Home. CJ is watching your terminal now.")
}

// cmdPlay plays a random sound from the given category.
func cmdPlay() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: grove-street play <category>")
		fmt.Fprintln(os.Stderr, "Categories: session_start, task_complete, task_error, input_required, resource_limit, user_spam")
		os.Exit(1)
	}

	category := os.Args[2]
	cfg := config.Load()

	path := player.Pick(category, cfg)
	if path == "" {
		fmt.Fprintf(os.Stderr, "No sounds found for category %q in %s\n", category, filepath.Join(config.SoundsDir(), category))
		os.Exit(1)
	}

	fmt.Printf("Playing: %s\n", filepath.Base(path))
	player.Play(path, cfg.Volume)
}

// cmdList lists all sounds organized by category.
func cmdList() {
	soundsDir := config.SoundsDir()
	categories := []string{"session_start", "task_complete", "task_error", "input_required", "resource_limit", "user_spam"}

	total := 0
	for _, cat := range categories {
		dir := filepath.Join(soundsDir, cat)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		var files []string
		for _, e := range entries {
			if !e.IsDir() && isAudio(e.Name()) {
				files = append(files, e.Name())
			}
		}

		if len(files) == 0 {
			continue
		}

		fmt.Printf("\n%s (%d):\n", cat, len(files))
		for _, f := range files {
			fmt.Printf("  %s\n", f)
		}
		total += len(files)
	}

	if total == 0 {
		fmt.Printf("No sounds found in %s\n", soundsDir)
		fmt.Println("Add .wav/.mp3/.ogg files to category subdirectories.")
	} else {
		fmt.Printf("\n%d sounds total\n", total)
	}
}

// cmdUpdate checks for and applies updates.
func cmdUpdate() {
	fmt.Println("[CJ] Checking for updates...")

	newVersion, err := updater.Check(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[CJ] %v\n", err)
		os.Exit(1)
	}

	if newVersion == "" {
		fmt.Println("[CJ] Already up to date (v" + version + ")")
		return
	}

	fmt.Printf("[CJ] New version available: v%s (current: v%s)\n", newVersion, version)
	fmt.Println("[CJ] Updating...")

	if err := updater.Apply(newVersion); err != nil {
		fmt.Fprintf(os.Stderr, "[CJ] Update failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("[CJ] Updated to v" + newVersion)
}

// cmdUninstall removes hooks from all known IDE configs.
func cmdUninstall() {
	home, _ := os.UserHomeDir()

	// Claude Code
	unregisterJSONHooks(filepath.Join(home, ".claude", "settings.json"), "grove-street")
	fmt.Println("[CJ] Removed hooks from Claude Code")

	// Cursor
	unregisterJSONHooks(filepath.Join(home, ".cursor", "hooks.json"), "grove-street")
	fmt.Println("[CJ] Removed hooks from Cursor")

	// Windsurf
	unregisterJSONHooks(filepath.Join(home, ".codeium", "windsurf", "hooks.json"), "grove-street")
	fmt.Println("[CJ] Removed hooks from Windsurf")

	// Copilot
	unregisterJSONHooks(filepath.Join(home, ".github", "hooks", "hooks.json"), "grove-street")
	fmt.Println("[CJ] Removed hooks from GitHub Copilot")

	// Kiro
	kiroPath := filepath.Join(home, ".kiro", "agents", "grove-street.json")
	os.Remove(kiroPath)
	fmt.Println("[CJ] Removed hooks from Kiro")

	fmt.Println()
	fmt.Println("[CJ] All hooks removed. To fully uninstall, run:")
	fmt.Printf("  rm -rf %s\n", config.DataDir())
}

func printUsage() {
	fmt.Println(`Grove Street — GTA San Andreas voice notifications for AI coding agents

"Ah shit, here we go again." — CJ

Usage:
  grove-street <command> [options]

Commands:
  hook                  Handle an IDE hook event (reads JSON from stdin)
  setup [--ide <name>]  Register hooks for detected IDEs
  play <category>       Test-play a random sound from a category
  list                  List all installed sounds
  update                Check for updates
  uninstall             Remove all hooks
  version               Print version

Categories:
  session_start, task_complete, task_error,
  input_required, resource_limit, user_spam

Supported IDEs:
  Claude Code, Cursor, Windsurf, GitHub Copilot, Kiro`)
}

// --- IDE config directories ---

func claudeConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

func cursorConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cursor")
}

func windsurfConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".codeium", "windsurf")
}

func copilotConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".github")
}

func kiroConfigDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kiro")
}

// --- Hook registration ---

func registerClaudeHooks(binPath string) error {
	settingsPath := filepath.Join(claudeConfigDir(), "settings.json")
	events := []string{"SessionStart", "Stop", "Notification", "SubagentStop", "PreCompact", "PermissionRequest"}
	return registerJSONHooksPerEvent(settingsPath, "hooks", binPath+" hook", events)
}

func registerCursorHooks(binPath string) error {
	settingsPath := filepath.Join(cursorConfigDir(), "hooks.json")
	hookCmd := binPath + " hook --source cursor"
	events := []string{"stop", "beforeShellExecution", "beforeMCPExecution"}
	return registerJSONHooks(settingsPath, "hooks", hookCmd, events)
}

func registerWindsurfHooks(binPath string) error {
	settingsPath := filepath.Join(windsurfConfigDir(), "hooks.json")
	hookCmd := binPath + " hook --source windsurf"
	events := []string{"post_cascade_response", "pre_user_prompt"}
	return registerJSONHooks(settingsPath, "hooks", hookCmd, events)
}

func registerCopilotHooks(binPath string) error {
	configDir := copilotConfigDir()
	os.MkdirAll(filepath.Join(configDir, "hooks"), 0755)
	settingsPath := filepath.Join(configDir, "hooks", "hooks.json")
	hookCmd := binPath + " hook --source copilot"
	events := []string{"sessionStart", "postToolUse", "errorOccurred"}
	return registerJSONHooks(settingsPath, "hooks", hookCmd, events)
}

func registerKiroHooks(binPath string) error {
	agentsDir := filepath.Join(kiroConfigDir(), "agents")
	os.MkdirAll(agentsDir, 0755)

	hookCmd := binPath + " hook --source kiro"
	kiroConfig := map[string]interface{}{
		"name":        "grove-street",
		"description": "GTA San Andreas voice notifications",
		"hooks": map[string]interface{}{
			"agentSpawn":      []interface{}{map[string]interface{}{"command": hookCmd}},
			"stop":            []interface{}{map[string]interface{}{"command": hookCmd}},
			"userPromptSubmit": []interface{}{map[string]interface{}{"command": hookCmd}},
		},
	}

	data, err := json.MarshalIndent(kiroConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(agentsDir, "grove-street.json"), data, 0644)
}

// registerJSONHooksPerEvent adds grove-street hook entries with --event flag per event type.
func registerJSONHooksPerEvent(path, hooksKey, baseCmd string, events []string) error {
	os.MkdirAll(filepath.Dir(path), 0755)

	settings := make(map[string]interface{})
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &settings)
	}

	hooksMap, ok := settings[hooksKey].(map[string]interface{})
	if !ok {
		hooksMap = make(map[string]interface{})
	}

	for _, event := range events {
		hookCmd := baseCmd + " --event " + event

		hookEntry := map[string]interface{}{
			"matcher": "",
			"hooks": []interface{}{
				map[string]interface{}{
					"type":    "command",
					"command": hookCmd,
				},
			},
		}

		var existing []interface{}
		if arr, ok := hooksMap[event].([]interface{}); ok {
			for _, h := range arr {
				if containsGroveStreet(h) {
					continue
				}
				existing = append(existing, h)
			}
		}
		existing = append(existing, hookEntry)
		hooksMap[event] = existing
	}

	settings[hooksKey] = hooksMap

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// registerJSONHooks adds grove-street hook entries to a JSON config file.
// It reads existing config, adds hook entries under the given key, and writes back.
func registerJSONHooks(path, hooksKey, hookCmd string, events []string) error {
	os.MkdirAll(filepath.Dir(path), 0755)

	// Read existing config or start fresh
	settings := make(map[string]interface{})
	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &settings)
	}

	// Get or create hooks map
	hooksMap, ok := settings[hooksKey].(map[string]interface{})
	if !ok {
		hooksMap = make(map[string]interface{})
	}

	hookEntry := map[string]interface{}{
		"matcher": "",
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": hookCmd,
			},
		},
	}

	for _, event := range events {
		// Get existing hooks for this event
		var existing []interface{}
		if arr, ok := hooksMap[event].([]interface{}); ok {
			// Filter out any existing grove-street hooks
			for _, h := range arr {
				if containsGroveStreet(h) {
					continue
				}
				existing = append(existing, h)
			}
		}
		existing = append(existing, hookEntry)
		hooksMap[event] = existing
	}

	settings[hooksKey] = hooksMap

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// containsGroveStreet checks if a hook entry references grove-street in any format.
func containsGroveStreet(h interface{}) bool {
	m, ok := h.(map[string]interface{})
	if !ok {
		return false
	}
	// Flat format: {"command": "...grove-street..."}
	if cmd, ok := m["command"].(string); ok && strings.Contains(cmd, "grove-street") {
		return true
	}
	// Nested format: {"hooks": [{"command": "...grove-street..."}]}
	if hooksArr, ok := m["hooks"].([]interface{}); ok {
		for _, hk := range hooksArr {
			if hm, ok := hk.(map[string]interface{}); ok {
				if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, "grove-street") {
					return true
				}
			}
		}
	}
	return false
}

// unregisterJSONHooks removes grove-street hook entries from a JSON config file.
func unregisterJSONHooks(path, _ string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return
	}

	hooksMap, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return
	}

	modified := false
	for event, val := range hooksMap {
		arr, ok := val.([]interface{})
		if !ok {
			continue
		}
		var filtered []interface{}
		for _, h := range arr {
			if containsGroveStreet(h) {
				modified = true
				continue
			}
			filtered = append(filtered, h)
		}
		if len(filtered) == 0 {
			delete(hooksMap, event)
		} else {
			hooksMap[event] = filtered
		}
	}

	if !modified {
		return
	}

	if len(hooksMap) == 0 {
		delete(settings, "hooks")
	}

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(path, out, 0644)
}

// --- Helpers ---

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// soundToPhrase converts a filename like "ah_shit_here_we_go_again.mp3" to "Ah shit here we go again"
func soundToPhrase(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	name = strings.ReplaceAll(name, "_", " ")
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}

func isAudio(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".wav" || ext == ".mp3" || ext == ".ogg"
}

// --- Notifications ---


// installIcon copies the icon from the binary's directory to the data directory.
func installIcon() {
	dest := config.IconPath()
	if _, err := os.Stat(dest); err == nil {
		return // already installed
	}

	// Look for icon next to the binary
	binPath, err := os.Executable()
	if err != nil {
		return
	}
	binPath, _ = filepath.EvalSymlinks(binPath)
	binDir := filepath.Dir(binPath)

	// Check a few possible locations (Homebrew puts it in share/grove-street/)
	candidates := []string{
		filepath.Join(binDir, "icon.png"),
		filepath.Join(binDir, "..", "share", "grove-street", "icon.png"),
		filepath.Join(binDir, "..", "assets", "icon.png"),
		filepath.Join(binDir, "..", "lib", "grove-street", "icon.png"),
	}

	for _, src := range candidates {
		if data, err := os.ReadFile(src); err == nil {
			os.MkdirAll(filepath.Dir(dest), 0755)
			os.WriteFile(dest, data, 0644)
			return
		}
	}
}

// installOverlayScript copies mac-overlay.js to the data directory.
func installOverlayScript() {
	dest := filepath.Join(config.DataDir(), "mac-overlay.js")
	if _, err := os.Stat(dest); err == nil {
		return
	}

	binPath, err := os.Executable()
	if err != nil {
		return
	}
	binPath, _ = filepath.EvalSymlinks(binPath)
	binDir := filepath.Dir(binPath)

	candidates := []string{
		filepath.Join(binDir, "mac-overlay.js"),
		filepath.Join(binDir, "..", "share", "grove-street", "mac-overlay.js"),
		filepath.Join(binDir, "..", "scripts", "mac-overlay.js"),
	}

	for _, src := range candidates {
		if data, err := os.ReadFile(src); err == nil {
			os.MkdirAll(filepath.Dir(dest), 0755)
			os.WriteFile(dest, data, 0644)
			return
		}
	}
}

func notify(soundFile string, cfg config.Config) {
	if !cfg.Notifications {
		return
	}

	if runtime.GOOS != "darwin" {
		return
	}

	// Voice line phrase from the sound filename
	phrase := soundToPhrase(soundFile)

	overlayScript := findOverlayScript()
	if overlayScript == "" {
		return
	}

	iconPath := config.IconPath()

	// Detect which app to focus on click
	bundleID := detectParentApp()

	// Project name from current working directory
	projectName := "grove-street"
	if wd, err := os.Getwd(); err == nil {
		projectName = filepath.Base(wd)
	}

	args := []string{
		"-l", "JavaScript", overlayScript,
		"Carl Johnson", phrase, iconPath, "7", bundleID, projectName,
	}

	cmd := exec.Command("osascript", args...)
	cmd.Start()
}

// findOverlayScript locates mac-overlay.js relative to the binary.
func findOverlayScript() string {
	binPath, err := os.Executable()
	if err != nil {
		return ""
	}
	binPath, _ = filepath.EvalSymlinks(binPath)
	binDir := filepath.Dir(binPath)

	candidates := []string{
		filepath.Join(binDir, "..", "share", "grove-street", "mac-overlay.js"),
		filepath.Join(binDir, "mac-overlay.js"),
		filepath.Join(binDir, "..", "scripts", "mac-overlay.js"),
	}

	// Also check data dir
	candidates = append(candidates, filepath.Join(config.DataDir(), "mac-overlay.js"))

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// detectParentApp tries to find the bundle ID of the terminal/IDE running us.
func detectParentApp() string {
	// Check common environment hints
	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		return "com.googlecode.iterm2"
	}
	if os.Getenv("TERM_PROGRAM") == "Apple_Terminal" {
		return "com.apple.Terminal"
	}
	if os.Getenv("TERM_PROGRAM") == "vscode" {
		return "com.microsoft.VSCode"
	}
	if os.Getenv("CURSOR_TRACE_ID") != "" {
		return "com.todesktop.230313mzl4w4u92"
	}
	// Default to Terminal
	return "com.apple.Terminal"
}
