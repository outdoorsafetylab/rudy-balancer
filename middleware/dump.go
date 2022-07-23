package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"strings"

	log "github.com/sirupsen/logrus"
)

type responseDumper struct {
	w http.ResponseWriter
	b bytes.Buffer
	s int
}

func (d *responseDumper) Header() http.Header {
	return d.w.Header()
}

func (d *responseDumper) Write(data []byte) (int, error) {
	d.b.Write(data)
	return d.w.Write(data)
}

func (d *responseDumper) WriteHeader(statusCode int) {
	if d.s == 0 {
		d.w.WriteHeader(statusCode)
		d.s = statusCode
	} else {
		log.Warningf("Attempt to write header again: %d", statusCode)
		debug.PrintStack()
	}
}

func Dump(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("Handling: %s %s", r.Method, r.RequestURI)
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read request body: %s", err.Error()), 500)
			return
		}
		if data != nil {
			r.Body = ioutil.NopCloser(bytes.NewBuffer(data))
		}
		dumper := &responseDumper{w: w}
		handler.ServeHTTP(dumper, r)
		err = dump(r, data, dumper)
		if err != nil {
			log.Errorf("Failed to dump: %s", err.Error())
		}
	})
}

func dump(r *http.Request, data []byte, d *responseDumper) error {
	type requestDump struct {
		RequestDump
		Body interface{} `json:"body,omitempty"`
	}
	type responseDump struct {
		ResponseDump
		Body interface{} `json:"body"`
	}
	out := &struct {
		Request  *requestDump  `json:"request"`
		Response *responseDump `json:"response"`
	}{
		Request: &requestDump{
			RequestDump: RequestDump{
				Method: r.Method,
				Uri:    r.RequestURI,
				Proto:  r.Proto,
			},
		},
		Response: &responseDump{
			ResponseDump: ResponseDump{
				Code: int32(d.s),
			},
		},
	}
	out.Request.Headers = dumpHeaders(r.Header)
	if len(data) > 0 {
		ctype := r.Header.Get("Content-Type")
		if strings.HasPrefix(ctype, "application/json") {
			out.Request.Body = json.RawMessage(data)
		} else if strings.HasPrefix(ctype, "text/") {
			out.Request.Body = string(data)
		}
	}
	if out.Response.Code == 0 {
		out.Response.Code = 200
	}
	out.Response.Headers = dumpHeaders(d.Header())
	data = d.b.Bytes()
	if len(data) > 0 {
		ctype := d.Header().Get("Content-Type")
		if strings.HasPrefix(ctype, "application/json") {
			out.Response.Body = json.RawMessage(data)
		} else if strings.HasPrefix(ctype, "text/") {
			out.Request.Body = string(data)
		}
	}
	data, err := json.Marshal(out)
	if err != nil {
		log.Errorf("Failed to marshal dump: %s", err.Error())
		return err
	}
	log.Debugf("%s", data)
	return nil
}

func dumpHeaders(header http.Header) []*HeaderDump {
	headers := make([]*HeaderDump, 0)
	for k, v := range header {
		headers = append(headers, &HeaderDump{Name: k, Values: v})
	}
	return headers
}
