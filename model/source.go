package model

import (
	"fmt"
	"net/http"
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
	s.LastModified, _ = http.ParseTime(res.Header.Get("Last-Modified"))
	s.LastModifiedUnix = s.LastModified.Unix()
	if s.LastModifiedUnix < 0 {
		s.LastModifiedUnix = 0
	}
	return nil
}
