package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckNewVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := release{
			TagName: "v1.2.0",
			Assets:  []asset{{Name: "grove-street_darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/dl"}},
		}
		json.NewEncoder(w).Encode(rel)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		t.Fatal(err)
	}

	if rel.TagName != "v1.2.0" {
		t.Errorf("TagName = %q, want v1.2.0", rel.TagName)
	}
	if len(rel.Assets) != 1 {
		t.Fatalf("Assets len = %d, want 1", len(rel.Assets))
	}
}

func TestVersionParsing(t *testing.T) {
	tests := []struct {
		name    string
		tagName string
		want    string
	}{
		{"with v prefix", "v1.2.3", "1.2.3"},
		{"without prefix", "1.2.3", "1.2.3"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tagName
			if len(got) > 0 && got[0] == 'v' {
				got = got[1:]
			}
			if got != tt.want {
				t.Errorf("version parse(%q) = %q, want %q", tt.tagName, got, tt.want)
			}
		})
	}
}

func TestIsBrewInstall(t *testing.T) {
	tests := []struct {
		exe  string
		want bool
	}{
		{"/opt/homebrew/bin/grove-street", true},
		{"/opt/homebrew/Cellar/grove-street/0.5.2/bin/grove-street", true},
		{"/usr/local/Cellar/grove-street/0.5.2/bin/grove-street", true},
		{"/home/linuxbrew/.linuxbrew/bin/grove-street", true},
		{"/usr/local/bin/grove-street", false},
		{"/Users/mayron/go/bin/grove-street", false},
		{"./grove-street", false},
	}
	for _, tt := range tests {
		t.Run(tt.exe, func(t *testing.T) {
			if got := isBrewInstall(tt.exe); got != tt.want {
				t.Errorf("isBrewInstall(%q) = %v, want %v", tt.exe, got, tt.want)
			}
		})
	}
}

func TestFindAssetExactMatch(t *testing.T) {
	assets := []asset{
		{Name: "grove-street_darwin_arm64.tar.gz", BrowserDownloadURL: "https://x/arm64"},
		{Name: "grove-street_darwin_amd64.tar.gz", BrowserDownloadURL: "https://x/amd64"},
		{Name: "sounds.tar.gz", BrowserDownloadURL: "https://x/sounds"},
	}

	if got := findAsset(assets, "grove-street_darwin_arm64.tar.gz"); got != "https://x/arm64" {
		t.Errorf("got %q, want arm64 url", got)
	}
	if got := findAsset(assets, "grove-street_linux_arm64.tar.gz"); got != "" {
		t.Errorf("expected miss, got %q", got)
	}
}

// TestFindAssetRejectsSubstring guards against the v0.5.2 bug where
// strings.Contains("grove-street_darwin_arm64") matched the tarball name and
// caused the gzip to be written as the executable.
func TestFindAssetRejectsSubstring(t *testing.T) {
	assets := []asset{
		{Name: "grove-street_darwin_arm64.tar.gz", BrowserDownloadURL: "https://x/tar"},
	}
	if got := findAsset(assets, "grove-street_darwin_arm64"); got != "" {
		t.Errorf("substring match leaked through: got %q", got)
	}
}

func TestExtractBinaryFromTarGz(t *testing.T) {
	want := []byte{0xCF, 0xFA, 0xED, 0xFE, 'b', 'i', 'n', 'a', 'r', 'y'}
	archive := makeTarGz(t, map[string][]byte{
		"grove-street":      want,
		"icon.png":          []byte("png"),
		"grove-notify.py":   []byte("python"),
		"grove-notify.ps1":  []byte("powershell"),
	})

	got, err := extractBinary(archive, "grove-street", false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("extracted bytes mismatch: got %x, want %x", got, want)
	}
}

func TestExtractBinaryMissing(t *testing.T) {
	archive := makeTarGz(t, map[string][]byte{"icon.png": []byte("png")})
	if _, err := extractBinary(archive, "grove-street", false); err == nil {
		t.Error("expected error when binary missing from archive")
	}
}

func TestValidateExecutable(t *testing.T) {
	tests := []struct {
		name    string
		goos    string
		data    []byte
		wantErr bool
	}{
		{"darwin mach-o 64 le", "darwin", []byte{0xCF, 0xFA, 0xED, 0xFE, 0x00}, false},
		{"darwin mach-o 64 be", "darwin", []byte{0xFE, 0xED, 0xFA, 0xCF, 0x00}, false},
		{"darwin universal", "darwin", []byte{0xCA, 0xFE, 0xBA, 0xBE, 0x00}, false},
		{"darwin gzip rejected", "darwin", []byte{0x1F, 0x8B, 0x08, 0x00, 0x00}, true},
		{"darwin html rejected", "darwin", []byte{'<', '!', 'D', 'O', 'C'}, true},
		{"linux elf", "linux", []byte{0x7F, 'E', 'L', 'F', 0x00}, false},
		{"linux gzip rejected", "linux", []byte{0x1F, 0x8B, 0x08, 0x00, 0x00}, true},
		{"windows pe", "windows", []byte{'M', 'Z', 0x00, 0x00, 0x00}, false},
		{"windows wrong rejected", "windows", []byte{0x7F, 'E', 'L', 'F', 0x00}, true},
		{"too small", "darwin", []byte{0x00}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExecutable(tt.data, tt.goos)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateExecutable() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReleaseJSONParsing(t *testing.T) {
	raw := `{
		"tag_name": "v0.4.0",
		"assets": [
			{"name": "grove-street_darwin_arm64.tar.gz", "browser_download_url": "https://example.com/1"},
			{"name": "grove-street_linux_amd64.tar.gz", "browser_download_url": "https://example.com/2"}
		]
	}`

	var rel release
	if err := json.Unmarshal([]byte(raw), &rel); err != nil {
		t.Fatal(err)
	}

	if rel.TagName != "v0.4.0" {
		t.Errorf("TagName = %q, want v0.4.0", rel.TagName)
	}
	if len(rel.Assets) != 2 {
		t.Fatalf("Assets len = %d, want 2", len(rel.Assets))
	}
}

func makeTarGz(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0755, Size: int64(len(body))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
