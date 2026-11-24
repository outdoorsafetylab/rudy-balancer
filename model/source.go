package model

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Source struct {
	Site             *Site     `json:"-" firestore:"-"`
	SiteName         string    `json:"Site" firestore:"-"`
	File             string    `json:"-" firestore:"-"`
	URL              string    `json:",omitempty" firestore:"-"`
	LastCheck        time.Time `json:"-"`
	LastCheckUnix    int64     `json:"LastCheck" firestore:"-"`
	Status           Status
	Error            error     `json:"-" firestore:"-"`
	LastModified     time.Time `json:"-"`
	LastModifiedUnix int64     `json:"LastModified" firestore:"-"`
	Size             int64
	Latency          time.Duration
	// CORS is whether a web page may download this file from this mirror
	// after the balancer's redirect (see CheckCORS). It is checked only when
	// the file is GOOD and does not affect Status.
	CORS bool `json:",omitempty"`
}

func (s *Source) Check(client *http.Client) error {
	s.LastCheck = time.Now()
	s.LastCheckUnix = s.LastCheck.Unix()
	res, err := client.Head(s.URL)
	duration := time.Since(s.LastCheck)
	if err != nil {
		s.Status = BAD
		s.Error = err
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		s.Status = BAD
		s.Error = fmt.Errorf("%d %s", res.StatusCode, http.StatusText(res.StatusCode))
		return s.Error
	}
	s.Status = GOOD
	s.Size = res.ContentLength
	s.Latency = (s.Latency + duration) / 2
	s.LastModified = lastModified(res.Header)
	s.LastModifiedUnix = s.LastModified.Unix()
	if s.LastModifiedUnix < 0 {
		s.LastModifiedUnix = 0
	}
	return nil
}

// CheckCORS reports whether a web page can download the file from this
// mirror once the balancer has redirected it there. After a cross-origin
// redirect the browser sends Origin: null, so only "*" passes. A download
// that resumes sends Range, which may make the browser send a preflight
// first: the mirror has to answer it with a 2xx that allows Range. Both are
// separate requests so the health check's own HEAD stays as it was; a mirror
// that answered an unexpected Origin with an error would otherwise be marked
// down.
func (s *Source) CheckCORS(client *http.Client) bool {
	head, err := http.NewRequest(http.MethodHead, s.URL, nil)
	if err != nil {
		return false
	}
	head.Header.Set("Origin", "null")
	res, err := client.Do(head)
	if err != nil {
		return false
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK || res.Header.Get("Access-Control-Allow-Origin") != "*" {
		return false
	}
	pre, err := http.NewRequest(http.MethodOptions, s.URL, nil)
	if err != nil {
		return false
	}
	pre.Header.Set("Origin", "null")
	pre.Header.Set("Access-Control-Request-Method", http.MethodGet)
	pre.Header.Set("Access-Control-Request-Headers", "range")
	res, err = client.Do(pre)
	if err != nil {
		return false
	}
	res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 || res.Header.Get("Access-Control-Allow-Origin") != "*" {
		return false
	}
	return allowsHeader(res.Header.Values("Access-Control-Allow-Headers"), "range")
}

// allowsHeader reports whether a preflight's Access-Control-Allow-Headers
// values list name (case-insensitively) or the wildcard.
func allowsHeader(values []string, name string) bool {
	for _, v := range values {
		for _, h := range strings.Split(v, ",") {
			h = strings.TrimSpace(h)
			if h == "*" || strings.EqualFold(h, name) {
				return true
			}
		}
	}
	return false
}

// lastModified is when the release reached this mirror, compared across
// mirrors to spot lagging ones. A mirror served from S3 reports the upload
// time as Last-Modified, hours after the release; rclone keeps the source's
// modification time in x-amz-meta-mtime, the same time the other mirrors
// (synced with rclone too) report, so that one wins when present. It is cut
// to whole seconds like Last-Modified, or a fraction would make every other
// mirror look a second behind.
func lastModified(h http.Header) time.Time {
	if v := h.Get("X-Amz-Meta-Mtime"); v != "" {
		// rclone writes decimal seconds; keep the whole part as text so a
		// fraction like .999999999 cannot round up through float64.
		if sec, err := strconv.ParseInt(strings.SplitN(v, ".", 2)[0], 10, 64); err == nil && sec > 0 {
			return time.Unix(sec, 0).UTC()
		}
	}
	t, _ := http.ParseTime(h.Get("Last-Modified"))
	return t
}
