package downstream

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The registry in the repository is what pull requests change, so it must
// always load.
func TestRegistryLoads(t *testing.T) {
	list, err := Load("../config/downstreams.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("no downstreams in config/downstreams.yaml")
	}
}

func TestLoadRejects(t *testing.T) {
	cases := []struct {
		name, yaml, want string
	}{
		{"unknown key", "downstreams:\n- id: a\n  name: A\n  url: https://a.example\n  useragent:\n  - AppA/\n", "useragent"},
		{"bad id", entry("App A", "AppA/", ""), "id"},
		{"duplicate id", entry("a", "AppA/", "") + entryItem("a", "AppB/", ""), "duplicate id"},
		{"no name", "downstreams:\n- id: a\n  url: https://a.example\n  useragents:\n  - AppA/\n", "name is required"},
		{"http url", "downstreams:\n- id: a\n  name: A\n  url: http://a.example\n  useragents:\n  - AppA/\n", "https"},
		{"nothing to match", "downstreams:\n- id: a\n  name: A\n  url: https://a.example\n", "needs useragents"},
		{"generic agent", entry("a", "okhttp/", ""), "library or platform default"},
		{"generic agent with suffix", entry("a", "Mozilla/5.0 (Linux; Android 14)", ""), "library or platform default"},
		{"prefix of a generic agent", entry("a", "Dalv", ""), "library or platform default"},
		{"short agent", entry("a", "ab/", ""), "shorter"},
		{"spaces", entry("a", " AppA/", ""), "spaces"},
		{"overlapping agents", entry("a", "Hiker", "") + entryItem("b", "HikerPro/", ""), "overlaps"},
		{"referer with scheme", entry("a", "", "https://a.example"), "host name"},
		{"referer with path", entry("a", "", "a.example/map"), "host name"},
		{"upper-case referer", entry("a", "", "A.example"), "host name"},
		{"duplicate referer", entry("a", "", "a.example") + entryItem("b", "", "a.example"), "already registered"},
	}
	for _, c := range cases {
		_, err := Load(write(t, c.yaml))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %v, want one containing %q", c.name, err, c.want)
		}
	}
}

func TestLoadAccepts(t *testing.T) {
	yaml := entry("a", "AppA/", "") + entryItem("b", "", "www.b.example") + entryItem("c", "AppC/", "c.example")
	list, err := Load(write(t, yaml))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[2].Referers[0] != "c.example" || list[0].UserAgents[0] != "AppA/" {
		t.Errorf("loaded %+v", list)
	}
}

func entry(id, ua, referer string) string {
	return "downstreams:\n" + entryItem(id, ua, referer)
}

func entryItem(id, ua, referer string) string {
	s := "- id: " + id + "\n  name: N\n  url: https://n.example\n"
	if ua != "" {
		s += "  useragents:\n  - \"" + ua + "\"\n"
	}
	if referer != "" {
		s += "  referers:\n  - " + referer + "\n"
	}
	return s
}

func write(t *testing.T, content string) string {
	path := filepath.Join(t.TempDir(), "downstreams.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
