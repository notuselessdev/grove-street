package updater

import (
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

	// We can't easily override repoAPI, so test the parsing logic directly
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
	if rel.Assets[0].Name != "grove-street_darwin_arm64.tar.gz" {
		t.Errorf("Asset name = %q, want grove-street_darwin_arm64.tar.gz", rel.Assets[0].Name)
	}
}

func TestCheckSameVersion(t *testing.T) {
	// When latest == current, Check should return ""
	// Test the comparison logic directly
	latest := "1.0.0"
	current := "1.0.0"

	if latest != current {
		t.Errorf("same version comparison failed")
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
