package db

import (
	"service/config"

	"cloud.google.com/go/firestore"
)

func Collection() *firestore.CollectionRef {
	return Client().Collection(config.Get().GetString("firestore.collection"))
}
