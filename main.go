package main

import (
	"flag"
	"fmt"
	"os"
	"service/db"
	"service/server"

	"service/config"
	"service/log"
)

func main() {
	name := flag.String("c", "config", "config")
	flag.Usage = func() {
		fmt.Printf("Usage: %s -c <config name>\n", os.Args[0])
		os.Exit(1)
	}
	flag.Parse()
	if err := config.Init(*name); err != nil {
		os.Exit(1)
	}
	err := log.Init()
	if err != nil {
		os.Exit(-1)
	}
	s := log.GetSugar()
	err = db.Init(s)
	if err != nil {
		os.Exit(-1)
	}
	defer db.Deinit(s)
	server := server.New(s)
	err = server.Run()
	if err == nil {
		os.Exit(0)
	} else {
		os.Exit(-1)
	}
}
