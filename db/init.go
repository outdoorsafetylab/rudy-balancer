package db

import (
	"context"

	"service/config"
	"service/log"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

var client *firestore.Client

func Init() error {
	cfg := config.Get()
	options := make([]option.ClientOption, 0)
	credential := cfg.GetString("firestore.credential")
	if credential != "" {
		log.Warnf("Connecting to firestore with service account file: %s", credential)
		options = append(options, option.WithCredentialsFile(credential))
	}
	var err error
	client, err = firestore.NewClient(context.Background(), cfg.GetString("firestore.project_id"), options...)
	if err != nil {
		log.Errorf("Failed to create firestore client: %s", err.Error())
		return err
	}
	return nil
}

func Deinit() {
	if client != nil {
		client.Close()
	}
}

func Client() *firestore.Client {
	return client
}
