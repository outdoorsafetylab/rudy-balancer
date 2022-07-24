package main

import (
	"flag"
	"os"

	"service/firestore"
	"service/server"

	"service/config"

	log "github.com/sirupsen/logrus"
)

var arguments = &struct {
	environment string
	webroot     string
}{
	environment: "local",
	webroot:     "webroot",
}

func main() {
	flag.StringVar(&arguments.environment, "e", arguments.environment, "Environemnt")
	flag.StringVar(&arguments.webroot, "w", arguments.webroot, "Web root")
	flag.Parse()
	if err := config.Init(arguments.environment); err != nil {
		os.Exit(1)
	}
	log.SetLevel(log.TraceLevel)
	err := firestore.Init()
	if err != nil {
		os.Exit(-1)
	}
	defer firestore.Deinit()
	server := server.New()
	err = server.Run(arguments.webroot)
	if err == nil {
		os.Exit(0)
	} else {
		log.Errorf("Failed to run: %s", err.Error())
		os.Exit(-1)
	}
}
