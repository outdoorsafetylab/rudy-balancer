/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"service/config"
	"service/db"
	"service/log"
	"service/server"

	"github.com/spf13/cobra"
)

var serveOptions = &struct {
	config string
}{
	config: "local",
}

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the server",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := config.Init(serveOptions.config)
		if err != nil {
			return err
		}
		err = log.Init()
		if err != nil {
			return err
		}
		err = db.Init()
		if err != nil {
			return err
		}
		defer db.Deinit()
		server := server.New()
		return server.Run()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringVarP(&serveOptions.config, "config", "c", serveOptions.config, "Config name")
}
