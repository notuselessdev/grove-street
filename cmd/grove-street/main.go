package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

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

	if !acquireCooldown() {
		return
	}

	path := player.Pick(category, cfg)
	if path == "" {
		warnNoSounds(category, cfg)
		return
	}
	player.Play(path, cfg.Volume)
	notify(filepath.Base(path), category, cfg)
	maybeAutoUpdate(cfg)
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
	installNotifyBinary()

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

	// Detect Homebrew sandbox: post_install runs without HOME write access.
	// Skip hook registration and tell the user to run setup manually.
	if os.Getenv("HOMEBREW_BREW_FILE") != "" || os.Getenv("HOMEBREW_PREFIX") != "" {
		fmt.Println("[CJ] Running inside Homebrew — skipping hook registration.")
		fmt.Println("[CJ] Run 'grove-street setup' in your terminal to register hooks.")
		fmt.Println()
		fmt.Println("[CJ] Grove Street. Home. CJ is watching your terminal now.")
		return
	}

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

	anyRegistered := false
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
			anyRegistered = true
		}
	}

	if !anyRegistered && targetIDE == "" {
		fmt.Fprintln(os.Stderr, "[CJ] No IDE config directories found. Run 'grove-street setup' after opening your IDE at least once.")
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
	notify(filepath.Base(path), category, cfg)
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
// Pass --silent to suppress all output (used by auto-update).
func cmdUpdate() {
	silent := len(os.Args) > 2 && os.Args[2] == "--silent"

	if !silent {
		fmt.Println("[CJ] Checking for updates...")
	}

	newVersion, err := updater.Check(version)
	if err != nil {
		if !silent {
			fmt.Fprintf(os.Stderr, "[CJ] %v\n", err)
			os.Exit(1)
		}
		return
	}

	if newVersion == "" {
		if !silent {
			fmt.Println("[CJ] Already up to date (v" + version + ")")
		}
		return
	}

	if !silent {
		fmt.Printf("[CJ] New version available: v%s (current: v%s)\n", newVersion, version)
		fmt.Println("[CJ] Updating...")
	}

	if err := updater.Apply(newVersion); err != nil {
		if !silent {
			fmt.Fprintf(os.Stderr, "[CJ] Update failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if !silent {
		fmt.Println("[CJ] Updated to v" + newVersion)
	}
}

// maybeAutoUpdate spawns a background update check at most once per day.
func maybeAutoUpdate(cfg config.Config) {
	if !cfg.AutoUpdate {
		return
	}

	cooldownFile := filepath.Join(config.DataDir(), ".last-update-check")
	if data, err := os.ReadFile(cooldownFile); err == nil {
		if ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			if time.Since(time.UnixMilli(ts)) < 24*time.Hour {
				return
			}
		}
	}
	os.WriteFile(cooldownFile, []byte(strconv.FormatInt(time.Now().UnixMilli(), 10)), 0644)

	exe, err := os.Executable()
	if err != nil {
		return
	}
	cmd := exec.Command(exe, "update", "--silent")
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Start()
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

// warnNoSounds fires a one-time notification telling the user their sound
// directory for the given category is empty. Shown at most once per day via
// a dedicated cooldown file so it doesn't spam on every hook.
func warnNoSounds(category string, cfg config.Config) {
	cooldownFile := filepath.Join(config.DataDir(), ".warn-no-sounds")
	if data, err := os.ReadFile(cooldownFile); err == nil {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data))); err == nil {
			if time.Since(t) < 24*time.Hour {
				return
			}
		}
	}
	os.WriteFile(cooldownFile, []byte(time.Now().Format(time.RFC3339)), 0644)

	dir := filepath.Join(config.SoundsDir(), category)
	msg := fmt.Sprintf("No sounds in %s — add .mp3/.wav files to %s", category, dir)
	fmt.Fprintln(os.Stderr, "[CJ]", msg)
	notify("⚠️  "+msg, "", cfg)
}

// acquireCooldown prevents duplicate notifications fired in quick succession.
// Returns true if enough time has passed since the last notification (2s).
// Uses a timestamp file at ~/.grove-street/.last-notification.
func acquireCooldown() bool {
	cooldownFile := filepath.Join(config.DataDir(), ".last-notification")
	const minInterval = 2 * time.Second

	if data, err := os.ReadFile(cooldownFile); err == nil {
		if ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64); err == nil {
			last := time.UnixMilli(ts)
			if time.Since(last) < minInterval {
				return false
			}
		}
	}

	os.WriteFile(cooldownFile, []byte(strconv.FormatInt(time.Now().UnixMilli(), 10)), 0644)
	return true
}

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

// installNotifyBinary installs the platform-specific notification helper.
// macOS: compiles grove-notify.swift to a binary.
// Linux: copies grove-notify.py script.
// Windows: copies grove-notify.ps1 script.
func installNotifyBinary() {
	switch runtime.GOOS {
	case "darwin":
		installDarwinNotify()
	case "linux":
		installScript("grove-notify.py")
	case "windows":
		installScript("grove-notify.ps1")
	}
}

func installDarwinNotify() {
	dest := filepath.Join(config.DataDir(), "grove-notify")
	if _, err := os.Stat(dest); err == nil {
		return
	}

	binPath, err := os.Executable()
	if err != nil {
		return
	}
	binPath, _ = filepath.EvalSymlinks(binPath)
	binDir := filepath.Dir(binPath)

	// Look for pre-compiled binary first
	binaryCandidates := []string{
		filepath.Join(binDir, "grove-notify"),
		filepath.Join(binDir, "..", "share", "grove-street", "grove-notify"),
		filepath.Join(binDir, "..", "scripts", "grove-notify"),
	}

	for _, src := range binaryCandidates {
		if info, err := os.Stat(src); err == nil && !info.IsDir() {
			if data, err := os.ReadFile(src); err == nil {
				os.MkdirAll(filepath.Dir(dest), 0755)
				os.WriteFile(dest, data, 0755)
				exec.Command("codesign", "--sign", "-", "--force", dest).Run()
				return
			}
		}
	}

	// Fall back to compiling from source
	swiftCandidates := []string{
		filepath.Join(binDir, "grove-notify.swift"),
		filepath.Join(binDir, "..", "share", "grove-street", "grove-notify.swift"),
		filepath.Join(binDir, "..", "scripts", "grove-notify.swift"),
	}

	for _, src := range swiftCandidates {
		if _, err := os.Stat(src); err == nil {
			os.MkdirAll(filepath.Dir(dest), 0755)
			cmd := exec.Command("swiftc", "-O", "-o", dest, src, "-framework", "Cocoa")
			if err := cmd.Run(); err == nil {
				exec.Command("codesign", "--sign", "-", "--force", dest).Run()
				fmt.Println("[CJ] Compiled notification overlay")
				return
			}
		}
	}
}

// installScript copies a notification script to the data directory.
func installScript(name string) {
	dest := filepath.Join(config.DataDir(), name)
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
		filepath.Join(binDir, name),
		filepath.Join(binDir, "..", "share", "grove-street", name),
		filepath.Join(binDir, "..", "scripts", name),
	}

	for _, src := range candidates {
		if data, err := os.ReadFile(src); err == nil {
			os.MkdirAll(filepath.Dir(dest), 0755)
			os.WriteFile(dest, data, 0755)
			return
		}
	}
}

// categoryLabel converts a category key to a human-readable label.
func categoryLabel(category string) string {
	labels := map[string]string{
		"task_complete":  "Task Complete",
		"task_error":     "Task Error",
		"input_required": "Input Required",
		"resource_limit": "Resource Limit",
		"session_start":  "Session Start",
		"user_spam":      "Chill Out",
	}
	if l, ok := labels[category]; ok {
		return l
	}
	return ""
}

func notify(soundFile string, category string, cfg config.Config) {
	if !cfg.Notifications {
		return
	}

	// Voice line phrase from the sound filename
	phrase := soundToPhrase(soundFile)

	iconPath := config.IconPath()

	// Detect which app and exact window to focus on click
	bundleID, appPID := detectParentApp()

	// Project name from current working directory
	projectName := "grove-street"
	if wd, err := os.Getwd(); err == nil {
		projectName = filepath.Base(wd)
	}

	position := cfg.NotificationPosition
	if position == "" {
		position = "top-right"
	}
	duration := cfg.NotificationDuration
	if duration <= 0 {
		duration = 7
	}
	durationStr := fmt.Sprintf("%.1f", duration)

	// Claim a notification slot for stacking
	slotDir := filepath.Join(config.DataDir(), ".notification-slots")
	slotIndex, slotFile := claimNotificationSlot()

	notifyArgs := []string{
		"Carl Johnson", phrase, iconPath, durationStr, bundleID, projectName, position,
		fmt.Sprintf("%d", slotIndex), slotDir, categoryLabel(category),
		fmt.Sprintf("%d", appPID),
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		notifyBin := findNotifyBinary("grove-notify")
		if notifyBin == "" {
			return
		}
		cmd = exec.Command(notifyBin, notifyArgs...)
	case "linux":
		script := findNotifyScript("grove-notify.py")
		if script == "" {
			return
		}
		cmd = exec.Command("python3", append([]string{script}, notifyArgs...)...)
	case "windows":
		script := findNotifyScript("grove-notify.ps1")
		if script == "" {
			return
		}
		cmd = exec.Command("powershell", append([]string{
			"-NoProfile", "-ExecutionPolicy", "Bypass", "-File", script,
		}, notifyArgs...)...)
	default:
		return
	}

	cmd.Start()

	// Write the PID into the lock file so future invocations
	// can check if the notification is still alive
	if slotFile != "" && cmd.Process != nil {
		os.WriteFile(slotFile, []byte(fmt.Sprintf("%d", cmd.Process.Pid)), 0644)
	}
}

// claimNotificationSlot finds the first available slot (0-9) and creates a lock file.
func claimNotificationSlot() (int, string) {
	slotDir := filepath.Join(config.DataDir(), ".notification-slots")
	os.MkdirAll(slotDir, 0755)

	for i := 0; i < 10; i++ {
		slotFile := filepath.Join(slotDir, fmt.Sprintf("%d.lock", i))
		// Try to create exclusively
		f, err := os.OpenFile(slotFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			// Lock file exists — check if the PID inside is still running
			if pidBytes, rerr := os.ReadFile(slotFile); rerr == nil && len(pidBytes) > 0 {
				if !isProcessAlive(strings.TrimSpace(string(pidBytes))) {
					// Process is dead, reclaim this slot
					os.Remove(slotFile)
					f, err = os.OpenFile(slotFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
					if err != nil {
						continue
					}
					f.Close()
					return i, slotFile
				}
			}
			continue
		}
		f.Close()
		return i, slotFile
	}
	return 0, ""
}

// isProcessAlive checks if a process with the given PID string is still running.
func isProcessAlive(pidStr string) bool {
	if runtime.GOOS == "windows" {
		return exec.Command("tasklist", "/FI", "PID eq "+pidStr).Run() == nil
	}
	return exec.Command("kill", "-0", pidStr).Run() == nil
}

// findNotifyBinary locates a compiled notification binary by name.
func findNotifyBinary(name string) string {
	return findInPaths(name)
}

// findNotifyScript locates a notification script by name.
func findNotifyScript(name string) string {
	return findInPaths(name)
}

// findInPaths searches standard locations for a file.
func findInPaths(name string) string {
	binPath, err := os.Executable()
	if err != nil {
		return ""
	}
	binPath, _ = filepath.EvalSymlinks(binPath)
	binDir := filepath.Dir(binPath)

	candidates := []string{
		filepath.Join(binDir, "..", "share", "grove-street", name),
		filepath.Join(binDir, name),
		filepath.Join(binDir, "..", "scripts", name),
		filepath.Join(config.DataDir(), name),
	}

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// detectParentApp walks the process tree to find the IDE/terminal that launched
// this hook. Returns the bundle ID and the PID of the matched process so
// grove-notify can activate the exact window (important when multiple windows
// of the same app are open).
func detectParentApp() (bundleID string, pid int32) {
	type ideMatch struct {
		substr   string
		bundleID string
	}
	ides := []ideMatch{
		{"Cursor", "com.todesktop.230313mzl4w4u92"},
		{"cursor", "com.todesktop.230313mzl4w4u92"},
		{"Code", "com.microsoft.VSCode"},
		{"Windsurf", "com.codeium.windsurf"},
		{"iTerm2", "com.googlecode.iterm2"},
		{"Warp", "dev.warp.Warp-Stable"},
		{"Terminal", "com.apple.Terminal"},
	}

	current := os.Getpid()
	for i := 0; i < 12; i++ {
		ppid, name := parentProcessInfo(current)
		if ppid <= 1 || name == "" {
			break
		}
		for _, ide := range ides {
			if strings.Contains(name, ide.substr) {
				return ide.bundleID, int32(ppid)
			}
		}
		current = ppid
	}

	// Fallback to environment variables
	switch os.Getenv("TERM_PROGRAM") {
	case "iTerm.app":
		return "com.googlecode.iterm2", 0
	case "Apple_Terminal":
		return "com.apple.Terminal", 0
	case "vscode":
		return "com.microsoft.VSCode", 0
	case "WarpTerminal":
		return "dev.warp.Warp-Stable", 0
	}
	if os.Getenv("CURSOR_TRACE_ID") != "" {
		return "com.todesktop.230313mzl4w4u92", 0
	}
	return "com.apple.Terminal", 0
}

// parentProcessInfo returns the parent PID and a searchable string combining
// the process name and full executable path for the given PID.
func parentProcessInfo(pid int) (ppid int, name string) {
	// Use command= (full path) so we can match "Warp" in /Applications/Warp.app/...
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "ppid=,command=").Output()
	if err != nil {
		return 0, ""
	}
	line := strings.TrimSpace(string(out))
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, ""
	}
	ppid, _ = strconv.Atoi(fields[0])
	// Use first token of command (the executable path) for matching
	name = fields[1]
	return
}
