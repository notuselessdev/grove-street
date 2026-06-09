package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const repoAPI = "https://api.github.com/repos/notuselessdev/grove-street/releases/latest"

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Check compares the current version against the latest GitHub release.
func Check(currentVersion string) (string, error) {
	resp, err := http.Get(repoAPI)
	if err != nil {
		return "", fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("failed to parse release: %w", err)
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	if latest == current || latest == "" {
		return "", nil
	}

	return latest, nil
}

// Apply downloads and replaces the current binary with the latest release.
// When the binary is managed by Homebrew, delegates to `brew upgrade`.
func Apply(newVersion string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	if isBrewInstall(exe) {
		if _, err := exec.LookPath("brew"); err != nil {
			return fmt.Errorf("installed via Homebrew but brew not in PATH: %w", err)
		}
		cmd := exec.Command("brew", "upgrade", "grove-street")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	resp, err := http.Get(repoAPI)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return fmt.Errorf("failed to parse release: %w", err)
	}

	archiveName := fmt.Sprintf("grove-street_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	binaryName := "grove-street"
	if runtime.GOOS == "windows" {
		archiveName = fmt.Sprintf("grove-street_%s_%s.zip", runtime.GOOS, runtime.GOARCH)
		binaryName = "grove-street.exe"
	}

	downloadURL := findAsset(rel.Assets, archiveName)
	if downloadURL == "" {
		return fmt.Errorf("no release asset matching %q for %s/%s", archiveName, runtime.GOOS, runtime.GOARCH)
	}

	dlResp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer dlResp.Body.Close()

	body, err := io.ReadAll(dlResp.Body)
	if err != nil {
		return err
	}

	binBytes, err := extractBinary(body, binaryName, runtime.GOOS == "windows")
	if err != nil {
		return fmt.Errorf("extract %s from %s: %w", binaryName, archiveName, err)
	}

	if err := validateExecutable(binBytes, runtime.GOOS); err != nil {
		return fmt.Errorf("downloaded binary failed validation: %w", err)
	}

	return atomicReplace(exe, binBytes)
}

// isBrewInstall reports whether the executable lives in a Homebrew-managed
// location. Self-replacing such binaries corrupts brew's symlink tracking and
// strands Cellar in a stale state.
func isBrewInstall(exe string) bool {
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		resolved = exe
	}
	for _, p := range []string{resolved, exe} {
		if strings.Contains(p, "/Cellar/") ||
			strings.HasPrefix(p, "/opt/homebrew/") ||
			strings.HasPrefix(p, "/usr/local/Cellar/") ||
			strings.HasPrefix(p, "/home/linuxbrew/") {
			return true
		}
	}
	return false
}

// findAsset returns the download URL of the asset whose Name exactly matches
// archiveName. Substring matching previously caused the tarball to be downloaded
// and written as the executable.
func findAsset(assets []asset, archiveName string) string {
	for _, a := range assets {
		if a.Name == archiveName {
			return a.BrowserDownloadURL
		}
	}
	return ""
}

// extractBinary pulls the named binary out of the release archive. The release
// workflow ships tarballs containing the binary alongside icons and notify
// scripts, so we must extract instead of writing the raw archive to disk.
func extractBinary(archive []byte, binaryName string, isZip bool) ([]byte, error) {
	if isZip {
		return nil, errors.New("zip extraction not yet supported")
	}

	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("binary %q not found in archive", binaryName)
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}
		return io.ReadAll(tr)
	}
}

// validateExecutable checks the file magic matches a native executable for the
// current platform. Stops us from ever swapping in a tarball, html error page,
// or partial download.
func validateExecutable(data []byte, goos string) error {
	if len(data) < 4 {
		return errors.New("binary too small")
	}
	switch goos {
	case "darwin":
		// Mach-O 64-bit little-endian: CF FA ED FE
		// Mach-O 64-bit big-endian:    FE ED FA CF
		// Mach-O 32-bit le/be also accepted for safety
		// Universal (fat) binary:      CA FE BA BE
		magic := [4]byte{data[0], data[1], data[2], data[3]}
		switch magic {
		case [4]byte{0xCF, 0xFA, 0xED, 0xFE},
			[4]byte{0xFE, 0xED, 0xFA, 0xCF},
			[4]byte{0xCE, 0xFA, 0xED, 0xFE},
			[4]byte{0xFE, 0xED, 0xFA, 0xCE},
			[4]byte{0xCA, 0xFE, 0xBA, 0xBE}:
			return nil
		}
		return fmt.Errorf("not a Mach-O binary (magic %x)", magic)
	case "linux":
		if data[0] == 0x7F && data[1] == 'E' && data[2] == 'L' && data[3] == 'F' {
			return nil
		}
		return fmt.Errorf("not an ELF binary (magic %x)", data[:4])
	case "windows":
		if data[0] == 'M' && data[1] == 'Z' {
			return nil
		}
		return fmt.Errorf("not a PE binary (magic %x)", data[:2])
	default:
		return nil
	}
}

// atomicReplace writes data to <exe>.new, renames the current binary to .old,
// then swaps the new one in. Rollback on failure.
func atomicReplace(exe string, data []byte) error {
	tmp := exe + ".new"
	if err := os.WriteFile(tmp, data, 0755); err != nil {
		return err
	}

	old := exe + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exe, old); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, exe); err != nil {
		os.Rename(old, exe)
		return err
	}
	os.Remove(old)
	return nil
}
