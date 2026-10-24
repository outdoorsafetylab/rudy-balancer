// Package downstream reads config/downstreams.yaml, the registry of apps and
// web sites that use Rudy Map files or tiles. Nothing in the balancer routes
// by it; it exists so that usage reports can name where traffic comes from.
// Entries arrive as pull requests from outside developers, so Load checks
// everything a reviewer could miss.
package downstream

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

type Downstream struct {
	ID         string   `yaml:"id"`
	Name       string   `yaml:"name"`
	Developer  string   `yaml:"developer"`
	URL        string   `yaml:"url"`
	Email      string   `yaml:"email"`
	UserAgents []string `yaml:"user_agents"`
	Referers   []string `yaml:"referers"`
}

type registry struct {
	Downstreams []*Downstream `yaml:"downstreams"`
}

var (
	idPattern    = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	labelPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
	// A top-level domain is letters, or punycode for an internationalised one.
	tldPattern = regexp.MustCompile(`^([a-z]{2,63}|xn--[a-z0-9-]{1,59})$`)
)

// Prefixes that libraries and platforms send by default. Many unrelated apps
// share them, so none of them can name one product.
var genericUserAgents = []string{
	"AndroidDownloadManager/",
	"Apache-HttpClient/",
	"CFNetwork/",
	"Dalvik/",
	"Dart/",
	"Go-http-client/",
	"Java/",
	"Mozilla/",
	"Python-urllib/",
	"Runner/",
	"Wget/",
	"curl/",
	"okhttp/",
	"python-requests/",
}

// The shortest User-Agent prefix accepted, so that one entry cannot claim a
// large share of everyone else's traffic.
const minUserAgentLen = 4

// Load reads and checks the registry at path. Unknown keys are errors, so a
// misspelt field fails instead of being dropped, and so is a second YAML
// document, which would otherwise be skipped unchecked.
func Load(path string) ([]*Downstream, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.SetStrict(true)
	var r registry
	if err := dec.Decode(&r); err != nil {
		return nil, err
	}
	var extra interface{}
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("only one YAML document is allowed")
	}
	if err := check(r.Downstreams); err != nil {
		return nil, err
	}
	return r.Downstreams, nil
}

func check(list []*Downstream) error {
	ids := make(map[string]bool)
	agents := make(map[string]string)
	hosts := make(map[string]string)
	for i, d := range list {
		if d == nil {
			return fmt.Errorf("entry %d is empty", i+1)
		}
		if !idPattern.MatchString(d.ID) {
			return fmt.Errorf("entry %d: id %q must be lower-case letters, digits and single dashes", i+1, d.ID)
		}
		if ids[d.ID] {
			return fmt.Errorf("%s: duplicate id", d.ID)
		}
		ids[d.ID] = true
		if strings.TrimSpace(d.Name) == "" {
			return fmt.Errorf("%s: name is required", d.ID)
		}
		if u, err := url.Parse(d.URL); err != nil || u.Scheme != "https" || u.Hostname() == "" {
			return fmt.Errorf("%s: url %q must be an https URL", d.ID, d.URL)
		}
		// A bare address only: a display name ("Name <a@b>") would not be
		// what the field says.
		if d.Email != "" {
			if a, err := mail.ParseAddress(d.Email); err != nil || a.Address != d.Email {
				return fmt.Errorf("%s: email %q must be a bare address such as dev@example.com", d.ID, d.Email)
			}
		}
		if len(d.UserAgents) == 0 && len(d.Referers) == 0 {
			return fmt.Errorf("%s: needs user_agents, referers or both", d.ID)
		}
		for _, ua := range d.UserAgents {
			if err := checkUserAgent(ua); err != nil {
				return fmt.Errorf("%s: %w", d.ID, err)
			}
			// One prefix inside another would make the longer one's traffic
			// count for both entries.
			for other, owner := range agents {
				if strings.HasPrefix(ua, other) || strings.HasPrefix(other, ua) {
					return fmt.Errorf("%s: user agent %q overlaps %q of %s", d.ID, ua, other, owner)
				}
			}
			agents[ua] = d.ID
		}
		for _, host := range d.Referers {
			if !validHost(host) {
				return fmt.Errorf("%s: referer %q must be a lower-case host name, without scheme or path", d.ID, host)
			}
			if owner, ok := hosts[host]; ok {
				return fmt.Errorf("%s: referer %q is already registered by %s", d.ID, host, owner)
			}
			hosts[host] = d.ID
		}
	}
	return nil
}

// validHost accepts a lower-case DNS host name of at least two labels.
func validHost(host string) bool {
	if len(host) > 253 {
		return false
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 || !tldPattern.MatchString(labels[len(labels)-1]) {
		return false
	}
	for _, label := range labels {
		if !labelPattern.MatchString(label) {
			return false
		}
	}
	return true
}

func checkUserAgent(ua string) error {
	// Printable ASCII only: a header value cannot carry control characters,
	// and every product token we have seen is ASCII.
	for _, c := range ua {
		if c < 0x20 || c > 0x7e {
			return fmt.Errorf("user agent %q must be printable ASCII", ua)
		}
	}
	if ua != strings.TrimSpace(ua) {
		return fmt.Errorf("user agent %q has leading or trailing spaces", ua)
	}
	if len(ua) < minUserAgentLen {
		return fmt.Errorf("user agent %q is shorter than %d characters", ua, minUserAgentLen)
	}
	for _, generic := range genericUserAgents {
		if strings.HasPrefix(strings.ToLower(generic), strings.ToLower(ua)) || strings.HasPrefix(strings.ToLower(ua), strings.ToLower(generic)) {
			return fmt.Errorf("user agent %q is a library or platform default (%s), not a product name", ua, generic)
		}
	}
	return nil
}
