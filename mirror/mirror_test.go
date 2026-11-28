package mirror

import (
	"os"
	"path/filepath"
	"testing"

	"service/config"
)

func TestZeroWeightSiteIsHidden(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "mirrors.yaml")
	yaml := `files:
  weekly:
  - a.map.zip
sites:
- name: up
  host: up.example.org
  weight: 2
- name: zero
  host: zero.example.org
  weight: 0
- name: unset
  host: unset.example.org
- name: hidden
  host: hidden.example.org
  weight: 3
  hidden: true
`
	if err := os.WriteFile(file, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("MIRRORS_FILE", file)
	defer os.Unsetenv("MIRRORS_FILE")
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	sites, err := Sites()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"up": false, "zero": true, "unset": true, "hidden": true}
	if len(sites) != len(want) {
		t.Fatalf("got %d sites, want %d", len(sites), len(want))
	}
	for _, s := range sites {
		if s.Hidden != want[s.Name] {
			t.Errorf("%s: hidden = %v, want %v", s.Name, s.Hidden, want[s.Name])
		}
	}
}

// A folder's probe is listed like any file, so the health check and the
// sources cover it; a folder name with a slash is refused (#63).
func TestFoldersAreProbed(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "mirrors.yaml")
	write := func(yaml string) {
		if err := os.WriteFile(file, []byte(yaml), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	os.Setenv("MIRRORS_FILE", file)
	defer os.Unsetenv("MIRRORS_FILE")
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	write(`files:
  weekly:
  - a.map.zip
folders:
  style: version.txt
sites:
- name: up
  host: up.example.org
  prefix: /rudy
  redirectScheme: https
  weight: 1
`)
	m, err := Get()
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Files["folders"]; len(got) != 1 || got[0] != "style/version.txt" {
		t.Errorf("Files[folders] = %v, want [style/version.txt]", got)
	}
	found := false
	for _, s := range m.Sites[0].Sources {
		if s.File == "style/version.txt" {
			found = true
			if s.URL != "https://up.example.org/rudy/style/version.txt" {
				t.Errorf("probe URL %q", s.URL)
			}
		}
	}
	if !found {
		t.Error("no source for the folder's probe: the health check would not check it")
	}
	for _, bad := range []string{"folders:\n  a/b: version.txt\n", "folders:\n  style: \"\"\n", "folders:\n  drops: version.txt\n"} {
		write("files:\n  weekly:\n  - a.map.zip\n  - drops/b.map.zip\n" + bad + "sites:\n- name: up\n  host: up.example.org\n  weight: 1\n")
		if _, err := Get(); err == nil {
			t.Errorf("%q: no error", bad)
		}
	}
}

// An upstream is never redirected to, whatever weight it is given, but stays
// visible so the health check publishes its status.
func TestUpstreamSite(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "mirrors.yaml")
	yaml := "files:\n  weekly:\n  - a.map.zip\nsites:\n" +
		"- name: up\n  host: up.example.org\n  weight: 2\n" +
		"- name: RudyR2\n  host: data.example.org\n  prefix: /share\n  redirectscheme: https\n  weight: 5\n  upstream: true\n"
	if err := os.WriteFile(file, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Setenv("MIRRORS_FILE", file)
	defer os.Unsetenv("MIRRORS_FILE")
	if err := config.Init("docker"); err != nil {
		t.Fatal(err)
	}
	sites, err := Sites()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sites {
		if s.Name != "RudyR2" {
			continue
		}
		if !s.Upstream || s.Weight != 0 || s.Hidden {
			t.Errorf("RudyR2: upstream %v weight %d hidden %v; want upstream, weight 0, not hidden", s.Upstream, s.Weight, s.Hidden)
		}
		if len(s.Sources) != 1 || s.Sources[0].URL != "https://data.example.org/share/a.map.zip" {
			t.Errorf("RudyR2 sources: %+v", s.Sources)
		}
		return
	}
	t.Fatal("no RudyR2 site")
}
