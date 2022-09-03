package geoip

import (
	"encoding/json"
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

func ipAddress(req *http.Request) (net.IP, error) {
	var host string
	var err error
	fwd := req.Header.Get("X-Forwarded-For")
	if fwd != "" {
		splits := strings.Split(fwd, ",")
		host = splits[0]
	} else {
		addr := req.RemoteAddr
		host, _, err = net.SplitHostPort(addr)
		if err != nil {
			host = addr
		}
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", host)
	}
	return ip, nil
}

func Country(req *http.Request) (*geoip2.Country, error) {
	endpoint := config.Get().GetString("geoip.endpoint")
	if endpoint == "" {
		return nil, fmt.Errorf("no config for 'geoip.endpoint'")
	}
	ip, err := ipAddress(req)
	if err != nil {
		return nil, err
	}
	q := make(url.Values)
	q.Set("ip", ip.String())
	res, err := client.Get(fmt.Sprintf("%s/country?%s", endpoint, q.Encode()))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("%s", res.Status)
	}
	c := &country{}
	err = json.NewDecoder(res.Body).Decode(c)
	if err != nil {
		return nil, err
	}
	if c.Country.Country.GeoNameID == 0 {
		return nil, fmt.Errorf("country is unknown: %s", ip.String())
	}
	return c.Country, nil
}

type country struct {
	IP      string `json:"IP"`
	Updated string `json:"Updated,omitempty"`
	*geoip2.Country
}
