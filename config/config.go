package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

var config *viper.Viper

func Init(env string) error {
	var err error
	config, err = getConfig(env)
	if err != nil {
		log.Printf("Failed to get config '%s': %s", env, err.Error())
		return err
	}
	secret, err := getConfig("secret")
	if err == nil {
		_ = config.MergeConfigMap(secret.AllSettings())
	}
	return nil
}

func Get() *viper.Viper {
	return config
}

func getConfig(name string) (*viper.Viper, error) {
	cfg := viper.New()
	cfg.AutomaticEnv()
	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	cfg.SetConfigType("yaml")
	cfg.SetConfigName(name)
	cfg.AddConfigPath("./")
	cfg.AddConfigPath("config/")
	cfg.AddConfigPath("../config/")
	err := cfg.ReadInConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
