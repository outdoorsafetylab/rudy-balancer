package model

type Status int

const (
	GOOD = iota
	BAD
)

func (s Status) String() string {
	switch s {
	case GOOD:
		return "GOOD"
	case BAD:
		return "BAD"
	}
	return ""
}

type Health struct {
	Site   *Site
	Status Status
}
