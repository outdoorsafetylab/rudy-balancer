package server

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"service/config"

	log "github.com/sirupsen/logrus"
)

type server struct {
	signal  chan os.Signal
	httpErr chan error
}

func New() *server {
	server := &server{
		signal:  make(chan os.Signal, 1),
		httpErr: make(chan error, 1),
	}
	return server
}

func (s *server) Run(webroot string) error {
	cfg := config.Get()
	r, err := newRouter(webroot)
	if err != nil {
		return err
	}
	go func() {
		s.httpErr <- http.ListenAndServe(fmt.Sprintf(":%d", cfg.GetInt("port")), r)
	}()
	signal.Notify(s.signal, os.Interrupt)
	for {
		select {
		case err := <-s.httpErr:
			if err != nil {
				log.Errorf("HTTP error: %s", err.Error())
				return err
			}
		case <-s.signal:
			log.Infof("Interrupted")
			return nil
		}
	}
}
