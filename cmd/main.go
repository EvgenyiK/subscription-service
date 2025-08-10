package main

import (
	_ "github.com/EvgenyiK/subscription-service/cmd/docs"
	"github.com/EvgenyiK/subscription-service/internal/handlers"
	"log"
	"net/http"

	"github.com/EvgenyiK/subscription-service/internal/config"
	"github.com/EvgenyiK/subscription-service/internal/repository"
	"github.com/EvgenyiK/subscription-service/internal/server"
	"github.com/joho/godotenv"
)

// @title Subscription Service API
// @version 1.0
// @description API для управления подписками.
// @host localhost:8080

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	repo, err := repository.NewRepository(cfg)
	if err != nil {
		log.Fatal(err)
	}

	h := handlers.NewHandler(repo)

	router := server.NewRouter(h)

	log.Printf("Server starting on port %s...", cfg.ServerPort)
	err = http.ListenAndServe(":"+cfg.ServerPort, router)
	if err != nil {
		log.Fatal(err)
	}
}
