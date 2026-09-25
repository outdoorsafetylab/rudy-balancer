package model

import (
	"fmt"
	"net/http"
	"strconv"
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

// lastModified is when the release reached this mirror, compared across
// mirrors to spot lagging ones. A mirror served from S3 reports the upload
// time as Last-Modified, hours after the release; rclone keeps the source's
// modification time in x-amz-meta-mtime, the same time the other mirrors
// (synced with rclone too) report, so that one wins when present. It is cut
// to whole seconds like Last-Modified, or a fraction would make every other
// mirror look a second behind.
func lastModified(h http.Header) time.Time {
	if v := h.Get("X-Amz-Meta-Mtime"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return time.Unix(int64(f), 0).UTC()
		}
	}
	t, _ := http.ParseTime(h.Get("Last-Modified"))
	return t
}
