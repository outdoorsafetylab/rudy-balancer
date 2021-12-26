package server

import (
	"net/http"
	"os"
	"os/signal"

	"service/config"

	"github.com/crosstalkio/httpd"
	"github.com/crosstalkio/log"
)

type server struct {
	log.Sugar
	signal  chan os.Signal
	httpErr chan error
}

func New(s log.Sugar) *server {
	server := &server{
		Sugar:   s,
		signal:  make(chan os.Signal, 1),
		httpErr: make(chan error, 1),
	}
	return server
}

func (s *server) Run(root http.FileSystem) error {
	cfg := config.Get()
	r := NewRouter(s, root)
	go func() {
		s.httpErr <- httpd.BindHTTP(s, cfg.GetInt("port"), r, nil)
	}()
	signal.Notify(s.signal, os.Interrupt)
	for {
		select {
		case err := <-s.httpErr:
			if err != nil {
				s.Errorf("HTTP error: %s", err.Error())
				return err
			}
		case <-s.signal:
			s.Infof("Interrupted")
			return nil
		}
	}
}
