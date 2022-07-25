package model

type Status int

const (
	GOOD = iota
	BAD
	UNKNWON
)

func (s Status) String() string {
	switch s {
	case GOOD:
		return "GOOD"
	case BAD:
		return "BAD"
	}
	return "UNKNWON"
}

type Health struct {
	Site   *Site
	Status Status
}
