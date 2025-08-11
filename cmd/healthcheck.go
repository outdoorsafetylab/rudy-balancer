/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"service/config"
	"service/db"
	"service/healthcheck"
	"service/log"

	"github.com/spf13/cobra"
)

var healthcheckOptions = &struct {
	config  string
	portals bool
	mirrors bool
}{
	config:  "local",
	portals: false,
	mirrors: false,
}

// healthcheckCmd represents the healthcheck command
var healthcheckCmd = &cobra.Command{
	Use:   "healthcheck",
	Short: "Run the healthcheck",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := config.Init(healthcheckOptions.config)
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
		cfg := config.Get()
		ctx := context.Background()

		// Determine which health checks to run
		// If no specific flags are set, run both
		runPortals := !healthcheckOptions.portals && !healthcheckOptions.mirrors || healthcheckOptions.portals
		runMirrors := !healthcheckOptions.portals && !healthcheckOptions.mirrors || healthcheckOptions.mirrors

		// Part 1: Portal Sites Health Check
		if runPortals {
			log.Debugf("Starting portal sites health check")
			err = healthcheck.CheckPortalSites(cfg)
			if err != nil {
				log.Errorf("Portal sites health check failed: %s", err.Error())
				return err
			}
		} else {
			log.Debugf("Skipping portal sites health check")
		}

		// Part 2: Mirror Sites Health Check
		if runMirrors {
			log.Debugf("Starting mirror sites health check")
			err = healthcheck.CheckMirrorSites(cfg, ctx)
			if err != nil {
				log.Errorf("Mirror sites health check failed: %s", err.Error())
				return err
			}
		} else {
			log.Debugf("Skipping mirror sites health check")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(healthcheckCmd)
	healthcheckCmd.Flags().StringVarP(&healthcheckOptions.config, "config", "c", healthcheckOptions.config, "Config name")
	healthcheckCmd.Flags().BoolVar(&healthcheckOptions.portals, "portals", false, "Run only portal sites health check")
	healthcheckCmd.Flags().BoolVar(&healthcheckOptions.mirrors, "mirrors", false, "Run only mirror sites health check")
}
