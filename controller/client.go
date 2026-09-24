package controller

import (
	"context"
	"net/http"
	"time"

	"service/dao"
	"service/geoip"
	"service/hash"
)

// clientID is the redirect log's Client field (#19), from which reports count
// distinct clients per quarter. The address itself is never logged.
func clientID(ctx context.Context, r *http.Request, t time.Time) (string, error) {
	ip, err := geoip.IPAddress(r)
	if err != nil {
		return "", err
	}
	salt, err := dao.ClientSalt(ctx, dao.Quarter(t))
	if err != nil {
		return "", err
	}
	return hash.Client(salt, ip.String(), r.UserAgent()), nil
}
