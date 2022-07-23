package model

import (
	"errors"
	"net/http"
	"net/url"
	"time"
)

type Source struct {
	Site             *Site    `firestore:"-"`
	URL              *url.URL `json:"-" firestore:"-"`
	Status           Status
	LastModified     time.Time `json:"-"`
	LastModifiedUnix int64     `json:"LastModified" firestore:"-"`
	Size             int64
	Latency          int64
}

func (s *Source) Check(client *http.Client) error {
	start := time.Now()
	res, err := client.Head(s.URL.String())
	duration := time.Since(start)
	if err != nil {
		s.Status = BAD
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		s.Status = BAD
		return errors.New(res.Status)
	}
	s.Status = GOOD
	s.Size = res.ContentLength
	s.Latency = int64(duration)
	s.LastModified, _ = http.ParseTime(res.Header.Get("Last-Modified"))
	s.LastModifiedUnix = s.LastModified.Unix()
	if s.LastModifiedUnix < 0 {
		s.LastModifiedUnix = 0
	}
	return nil
}
