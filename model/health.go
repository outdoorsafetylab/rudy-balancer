package model

type Status int

const (
	UNKNWON = iota
	GOOD
	BAD
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
