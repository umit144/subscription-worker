package main

import (
	"context"
	"log"

	"github.com/umit144/subscription-worker/internal/database"
	"github.com/umit144/subscription-worker/internal/repository"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := database.NewDatabase("sail:password@tcp(localhost:3306)/laravel")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.DB.Close()

	repo := repository.NewSubscriptionRepository(db)
	ctx := context.Background()

	subscriptions, err := repo.GetExpiredSubscriptions(ctx)
	if err != nil {
		log.Fatalf("Failed to get expired subscriptions: %v", err)
	}

	for _, sub := range subscriptions {
		log.Printf("Processing subscription ID: %d, Device ID: %d, Application ID: %d",
			sub.ID, sub.DeviceID, sub.ApplicationID)
	}
}
