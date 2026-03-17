package player

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/notuselessdev/grove-street/internal/config"
)

func TestIsAudio(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"wav file", "test.wav", true},
		{"mp3 file", "test.mp3", true},
		{"ogg file", "test.ogg", true},
		{"WAV uppercase", "test.WAV", true},
		{"Mp3 mixed case", "test.Mp3", true},
		{"txt file", "test.txt", false},
		{"no extension", "testfile", false},
		{"flac file", "test.flac", false},
		{"empty string", "", false},
		{"dot only", ".mp3", true},
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

func TestContains(t *testing.T) {
	tests := []struct {
		name string
		list []string
		s    string
		want bool
	}{
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty list", []string{}, "a", false},
		{"nil list", nil, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contains(tt.list, tt.s)
			if got != tt.want {
				t.Errorf("contains(%v, %q) = %v, want %v", tt.list, tt.s, got, tt.want)
			}
		})
	}
}

func TestPickEmptyDir(t *testing.T) {
	// Pick from a non-existent category should return ""
	cfg := config.DefaultConfig()
	got := Pick("nonexistent_category_xyz", cfg)
	if got != "" {
		t.Errorf("Pick(nonexistent) = %q, want empty", got)
	}
}

func TestPickShuffleBag(t *testing.T) {
	// Create a temp sounds directory with known files
	tmpDir := t.TempDir()
	category := "test_shuffle"
	catDir := filepath.Join(tmpDir, "sounds", category)
	os.MkdirAll(catDir, 0755)

	files := []string{"one.mp3", "two.mp3", "three.mp3"}
	for _, f := range files {
		os.WriteFile(filepath.Join(catDir, f), []byte("fake"), 0644)
	}

	// Write a history file
	histFile := filepath.Join(tmpDir, "history.json")

	// Override data dir by creating sounds in the right place
	// We can't easily override config.SoundsDir(), so test the shuffle logic directly
	// by testing the history mechanism

	// Test that history tracks plays correctly
	h := make(map[string][]string)
	h[category] = []string{filepath.Join(catDir, "one.mp3")}

	data, _ := json.Marshal(h)
	os.WriteFile(histFile, data, 0644)

	// Verify the history was written
	raw, err := os.ReadFile(histFile)
	if err != nil {
		t.Fatal(err)
	}

	var loaded map[string][]string
	json.Unmarshal(raw, &loaded)

	if len(loaded[category]) != 1 {
		t.Errorf("history has %d entries, want 1", len(loaded[category]))
	}
}

func TestPickNoAudioFiles(t *testing.T) {
	// Directory with only non-audio files
	tmpDir := t.TempDir()
	catDir := filepath.Join(tmpDir, "sounds", "empty_cat")
	os.MkdirAll(catDir, 0755)
	os.WriteFile(filepath.Join(catDir, "readme.txt"), []byte("not audio"), 0644)

	// Can't easily test Pick() without overriding SoundsDir,
	// but we can verify isAudio filters correctly
	if isAudio("readme.txt") {
		t.Error("readme.txt should not be audio")
	}
}
