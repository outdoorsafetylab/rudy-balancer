package model

import "time"

type Version struct {
	Time   time.Time
	Commit string
	Tag    string
}
