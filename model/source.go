package model

import (
	"errors"
	"net/http"
	"net/url"
	"time"
)

type Source struct {
	Site             *Site     `json:",omitempty" firestore:"-"`
	File             string    `json:"-" firestore:"-"`
	URL              *url.URL  `json:"-" firestore:"-"`
	URLString        string    `json:"URL,omitempty" firestore:"-"`
	LastCheck        time.Time `json:"-"`
	LastCheckUnix    int64     `json:"LastCheck" firestore:"-"`
	Status           Status
	LastModified     time.Time `json:"-"`
	LastModifiedUnix int64     `json:"LastModified" firestore:"-"`
	Size             int64
	Latency          time.Duration
}

func (s *Source) Check(client *http.Client) error {
	s.LastCheck = time.Now()
	s.LastCheckUnix = s.LastCheck.Unix()
	res, err := client.Head(s.URL.String())
	duration := time.Since(s.LastCheck)
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
	s.Latency = (s.Latency + duration) / 2
	s.LastModified, _ = http.ParseTime(res.Header.Get("Last-Modified"))
	s.LastModifiedUnix = s.LastModified.Unix()
	if s.LastModifiedUnix < 0 {
		s.LastModifiedUnix = 0
	}
	return nil
}
