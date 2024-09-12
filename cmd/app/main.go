package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"https://github.com/andreas-hs/nd-go-app/internal/config"
	"https://github.com/andreas-hs/nd-go-app/internal/dependencies"
	"https://github.com/andreas-hs/nd-go-app/internal/infrastructure/database"
	"https://github.com/andreas-hs/nd-go-app/internal/infrastructure/rabbitmq"
	"https://github.com/andreas-hs/nd-go-app/internal/logging"
	"https://github.com/andreas-hs/nd-go-app/internal/services"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := logrus.New()
	ctx = context.WithValue(ctx, "logger", logger)

	cfg, err := config.GetConfig()
	if err != nil {
		logging.LogFatal(logger, "Configuration loading error", err)
	}

	db, err := database.SetupDatabase(cfg.MySQLDSN)
	if err != nil {
		logging.LogFatal(logger, "Database connection error", err)
	}

	rabbitChannel, rabbitConn, err := rabbitmq.SetupRabbitMQ(cfg.RabbitMQURL)
	if err != nil {
		logging.LogFatal(logger, "RabbitMQ connection error", err)
	}

	deps := &dependencies.Dependencies{
		Logger:     logger,
		DB:         db,
		RabbitConn: rabbitConn,
		RabbitCh:   rabbitChannel,
	}

	dataProcessor := services.NewDataProcessor(deps)

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logging.LogInfo(logger, "Received shutdown signal, closing application...")
		cancel()

		err := dataProcessor.Stop()
		if err != nil {
			logging.LogError(logger, "Error stopping data processor", err)
		}
	}()

	if err := dataProcessor.Start(ctx); err != nil {
		logging.LogFatal(logger, "Data processing error", err)
	}
}
