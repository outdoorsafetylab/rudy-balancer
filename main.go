package main

import (
	"flag"
	"os"

	"service/config"
	"service/db"
	"service/log"
	"service/server"
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
	err := config.Init(arguments.environment)
	if err != nil {
		os.Exit(1)
	}
	err = log.Init()
	if err != nil {
		os.Exit(-1)
	}
	err = db.Init()
	if err != nil {
		os.Exit(-1)
	}
	defer db.Deinit()
	server := server.New()
	err = server.Run(arguments.webroot)
	if err == nil {
		os.Exit(0)
	} else {
		log.Errorf("Failed to run: %s", err.Error())
		os.Exit(-1)
	}
}
