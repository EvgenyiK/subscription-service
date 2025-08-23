package main

import (
	"context"
	_ "github.com/EvgenyiK/subscription-service/cmd/docs"
	"github.com/EvgenyiK/subscription-service/internal/handlers"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	serverAddr := ":" + cfg.ServerPort
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	go func() {
		log.Printf("Server starting on port %s...", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v", serverAddr, err)
		}
	}()

	// Создаем канал для ловли системных сигналов
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Ожидаем сигнала
	sig := <-sigs
	log.Printf("Получен сигнал %s. Начинаем graceful shutdown...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Ошибка при graceful shutdown: %v", err)
	} else {
		log.Println("Сервер успешно остановлен")
	}

	log.Println("Выход из программы")
}
