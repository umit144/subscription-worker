package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/umit144/subscription-worker/internal/database"
	"github.com/umit144/subscription-worker/internal/repositories"
	services "github.com/umit144/subscription-worker/internal/services"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	db := initDb()
	defer db.DB.Close()

	redis := initRedis()

	defer redis.Close()

	subscriptionService := services.NewSubscriptionService(
		repositories.NewApplicationCredentialsRepository(db),
		repositories.NewDeviceRepository(db),
		repositories.NewSubscriptionRepository(db),
		redis,
		100,
	)

	err = subscriptionService.ProcessExpiredSubscriptions()
	if err != nil {
		panic(err)
	}
}

func initDb() *database.Database {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE"))

	db, err := database.NewDatabase(dsn)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func initRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
}
