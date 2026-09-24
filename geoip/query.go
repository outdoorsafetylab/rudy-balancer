package geoip

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"service/config"
	"strings"
	"time"

	"github.com/oschwald/geoip2-golang"
)

var (
	client = &http.Client{
		Timeout: 3 * time.Second,
	}
)

// IPAddress is the client's address as the redirect handlers see it. Cloud
// Run appends the address it received the request from to X-Forwarded-For,
// so only the last entry is trustworthy; anything before it is whatever the
// client chose to send. Errors never carry the address.
func IPAddress(req *http.Request) (net.IP, error) {
	var host string
	var err error
	fwd := req.Header.Get("X-Forwarded-For")
	if fwd != "" {
		splits := strings.Split(fwd, ",")
		host = strings.TrimSpace(splits[len(splits)-1])
	} else {
		addr := req.RemoteAddr
		host, _, err = net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("invalid client IP address")
	}
	return ip, nil
}

func Country(req *http.Request) (*geoip2.Country, error) {
	endpoint := config.Get().GetString("geoip.endpoint")
	if endpoint == "" {
		return nil, fmt.Errorf("no config for 'geoip.endpoint'")
	}
	ip, err := IPAddress(req)
	if err != nil {
		return nil, err
	}
	q := make(url.Values)
	q.Set("ip", ip.String())
	res, err := client.Get(fmt.Sprintf("%s/country?%s", endpoint, q.Encode()))
	if err != nil {
		// url.Error repeats the request URL, which carries the client's
		// address; keep only what went wrong.
		var ue *url.Error
		if errors.As(err, &ue) {
			return nil, fmt.Errorf("geoip request failed: %w", ue.Err)
		}
		return nil, fmt.Errorf("geoip request failed")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("%s", res.Status)
	}
	c := &country{}
	err = json.NewDecoder(res.Body).Decode(c)
	if err != nil {
		return nil, err
	}
	if c.Country.Country.GeoNameID == 0 {
		return nil, fmt.Errorf("country is unknown")
	}
	return c.Country, nil
}

type country struct {
	IP      string `json:"IP"`
	Updated string `json:"Updated,omitempty"`
	*geoip2.Country
}
