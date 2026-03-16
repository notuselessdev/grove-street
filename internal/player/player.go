package player

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/notuselessdev/grove-street/internal/config"
)

// Pick selects a random sound file from the given category.
func Pick(category string, cfg config.Config) string {
	dir := filepath.Join(config.SoundsDir(), category)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && isAudio(e.Name()) {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}

	if len(files) == 0 {
		return ""
	}

	return files[rand.Intn(len(files))]
}

// Play plays an audio file at the given volume (0.0 - 1.0).
func Play(path string, volume float64) {
	if volume <= 0 {
		return
	}

	switch runtime.GOOS {
	case "darwin":
		playMacOS(path, volume)
	case "linux":
		playLinux(path, volume)
	case "windows":
		playWindows(path, volume)
	default:
		fmt.Fprintf(os.Stderr, "grove-street: unsupported platform %s\n", runtime.GOOS)
	}
}

func playMacOS(path string, volume float64) {
	vol := fmt.Sprintf("%.0f", volume*255)
	cmd := exec.Command("afplay", "-v", vol, path)
	cmd.Start()
}

func playLinux(path string, volume float64) {
	players := []struct {
		name string
		args func() []string
	}{
		{"pw-play", func() []string {
			return []string{"--volume", fmt.Sprintf("%.2f", volume), path}
		}},
		{"paplay", func() []string {
			vol := fmt.Sprintf("%d", int(volume*65536))
			return []string{"--volume", vol, path}
		}},
		{"ffplay", func() []string {
			vol := fmt.Sprintf("%.2f", volume)
			return []string{"-nodisp", "-autoexit", "-volume", vol, "-loglevel", "quiet", path}
		}},
		{"mpv", func() []string {
			vol := fmt.Sprintf("%.0f", volume*100)
			return []string{"--no-video", "--really-quiet", "--volume", vol, path}
		}},
		{"aplay", func() []string {
			return []string{path}
		}},
	}

	for _, p := range players {
		if _, err := exec.LookPath(p.name); err == nil {
			cmd := exec.Command(p.name, p.args()...)
			cmd.Start()
			return
		}
	}

	fmt.Fprintln(os.Stderr, "grove-street: no audio player found. Install pulseaudio, pipewire, ffmpeg, mpv, or alsa-utils.")
}

func playWindows(path string, volume float64) {
	ps := fmt.Sprintf(
		`Add-Type -AssemblyName PresentationCore; `+
			`$p = New-Object System.Windows.Media.MediaPlayer; `+
			`$p.Open([uri]"%s"); `+
			`$p.Volume = %.2f; `+
			`$p.Play(); `+
			`Start-Sleep -Seconds 5`,
		path, volume,
	)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
	cmd.Start()
}

func isAudio(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".wav" || ext == ".mp3" || ext == ".ogg"
}
