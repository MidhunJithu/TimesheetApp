package firestore

import (
	"context"
	"log"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

type FireStore struct {
	App    *firebase.App
	Client *firestore.Client
}

func InitDb() *FireStore {
	// Use a service account
	ctx := context.Background()
	sa := option.WithCredentialsFile("firebasekey.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	// defer client.Close()
	return &FireStore{
		App:    app,
		Client: client,
	}
}
