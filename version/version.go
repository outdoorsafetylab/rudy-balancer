package version

import (
	"strconv"
	"time"

	"service/model"
)

var (
	BuildTime string
	GitHash   string
	GitTag    string
)

func Get() *model.Version {
	t, _ := strconv.ParseInt(BuildTime, 10, 64)
	if t <= 0 {
		t = time.Now().Unix()
	}
	return &model.Version{
		Time:   time.Unix(t, 0),
		Commit: GitHash,
		Tag:    GitTag,
	}
}
