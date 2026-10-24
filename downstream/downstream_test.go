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
		{"unknown key", "downstreams:\n- id: a\n  name: A\n  url: https://a.example\n  useragents:\n  - AppA/\n", "useragents"},
		{"bad id", entry("App A", "AppA/", ""), "id"},
		{"duplicate id", entry("a", "AppA/", "") + entryItem("a", "AppB/", ""), "duplicate id"},
		{"no name", "downstreams:\n- id: a\n  url: https://a.example\n  user_agents:\n  - AppA/\n", "name is required"},
		{"http url", "downstreams:\n- id: a\n  name: A\n  url: http://a.example\n  user_agents:\n  - AppA/\n", "https"},
		{"nothing to match", "downstreams:\n- id: a\n  name: A\n  url: https://a.example\n", "needs user_agents"},
		{"bad email", "downstreams:\n- id: a\n  name: A\n  url: https://a.example\n  email: not-an-address\n  user_agents:\n  - AppA/\n", "email"},
		{"email with display name", "downstreams:\n- id: a\n  name: A\n  url: https://a.example\n  email: A <a@a.example>\n  user_agents:\n  - AppA/\n", "email"},
		{"generic agent", entry("a", "okhttp/", ""), "library or platform default"},
		{"generic agent with suffix", entry("a", "Mozilla/5.0 (Linux; Android 14)", ""), "library or platform default"},
		{"prefix of a generic agent", entry("a", "Dalv", ""), "library or platform default"},
		{"short agent", entry("a", "ab/", ""), "shorter"},
		{"spaces", entry("a", " AppA/", ""), "spaces"},
		{"overlapping agents", entry("a", "Hiker", "") + entryItem("b", "HikerPro/", ""), "overlaps"},
		{"referer with scheme", entry("a", "", "https://a.example"), "host name"},
		{"referer with path", entry("a", "", "a.example/map"), "host name"},
		{"upper-case referer", entry("a", "", "A.example"), "host name"},
		{"second document", entry("a", "AppA/", "") + "---\ndownstreams:\n- id: b\n  bogus: 1\n", "one YAML document"},
		{"empty entry", "downstreams:\n-\n", "empty"},
		{"url without host", "downstreams:\n- id: a\n  name: A\n  url: https://:443/\n  user_agents:\n  - AppA/\n", "https"},
		{"control character", entry("a", "AppA/\\ninvalid", ""), "printable ASCII"},
		{"non-ASCII agent", entry("a", "登山App/", ""), "printable ASCII"},
		{"mixed-case generic agent", entry("a", "OkHttp/4", ""), "library or platform default"},
		{"same agent twice", entry("a", "AppA/", "") + entryItem("b", "AppA/", ""), "overlaps"},
		{"agent inside an earlier one", entry("a", "HikerPro/", "") + entryItem("b", "Hiker", ""), "overlaps"},
		{"single-label referer", entry("a", "", "localhost"), "host name"},
		{"numeric tld", entry("a", "", "a.example.123"), "host name"},
		{"label over 63", entry("a", "", strings.Repeat("a", 64)+".example"), "host name"},
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
	yaml := entry("a", "AppA/", "") + entryItem("b", "", "www.b.example") + entryItem("c", "AppC/", "c.example") +
		entryItem("e", "", "map.example.xn--kpry57d") +
		"- id: d\n  name: D\n  url: https://d.example\n  email: dev@d.example\n  user_agents:\n  - AppD/\n"
	list, err := Load(write(t, yaml))
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 5 || list[4].Email != "dev@d.example" || list[2].Referers[0] != "c.example" || list[0].UserAgents[0] != "AppA/" {
		t.Errorf("loaded %+v", list)
	}
}

func entry(id, ua, referer string) string {
	return "downstreams:\n" + entryItem(id, ua, referer)
}

func entryItem(id, ua, referer string) string {
	s := "- id: " + id + "\n  name: N\n  url: https://n.example\n"
	if ua != "" {
		s += "  user_agents:\n  - \"" + ua + "\"\n"
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
