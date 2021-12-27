package model

type Status int

const (
	GOOD = iota
	BAD
	DEAD
)

type Health struct {
	Site   *Site
	Status Status
}
