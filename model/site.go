package model

import (
	"fmt"
	"net/url"
	"strings"
)

// Site is one mirror. MonthlyQuota caps the AWS account's CloudFront bytes
// this UTC month (cloudfront.Meter) while the site stays in rotation; 0 means
// no cap.
type Site struct {
	Name         string
	StatusPage   string `json:"-" firestore:"-"`
	Firestore    string `json:"-" firestore:"-"`
	MonthlyQuota int64  `json:"-" firestore:"-"`
	Hidden       bool   `json:",omitempty"`
	// Upstream marks where the mirrors sync from (Rudy Map's R2 copy): it is
	// health-checked and its status published, so a release stuck upstream
	// shows there, but no download is ever redirected to it.
	Upstream       bool      `json:",omitempty"`
	RedirectScheme string    `json:"-"`
	Host           string    `json:"-"`
	Prefix         string    `json:"-"`
	ProxyScheme    string    `json:"-"`
	Landing        string    `json:"-"`
	Sources        []*Source `json:"-"`
	Weight         int       `json:"-"`
}

func (s *Site) GetRedirectURL(uri string) string {
	return fmt.Sprintf("%s://%s%s", s.RedirectScheme, s.Host, s.path(uri))
}

func (s *Site) GetProxyURL(uri string) string {
	return fmt.Sprintf("%s://%s%s", s.ProxyScheme, s.Host, s.path(uri))
}

// path is the prefix and the file's path, escaped for a URL: a file name with a
// space (a glyph range under "Noto Sans Regular") would otherwise go out raw in
// a Location header, which http.Redirect escapes only beyond ASCII (#57).
func (s *Site) path(uri string) string {
	if uri != "" && !strings.HasPrefix(uri, "/") {
		uri = "/" + uri
	}
	return (&url.URL{Path: s.Prefix + uri}).EscapedPath()
}
