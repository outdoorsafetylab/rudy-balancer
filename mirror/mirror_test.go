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
