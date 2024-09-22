package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/andreas-hs/tc-go-app/internal/config"
	"github.com/andreas-hs/tc-go-app/internal/dependencies"
	"github.com/andreas-hs/tc-go-app/internal/infrastructure/database"
	"github.com/andreas-hs/tc-go-app/internal/infrastructure/rabbitmq"
	"github.com/andreas-hs/tc-go-app/internal/logging"
	"github.com/andreas-hs/tc-go-app/internal/services"
	"github.com/sirupsen/logrus"
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

	var db database.Database = &database.PostgresDatabase{}
	_, err = db.Connect(cfg.DbDSN)
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

		if err := dataProcessor.Stop(); err != nil {
			logging.LogError(logger, "Error stopping data processor", err)
		}
	}()

	if err := dataProcessor.Start(ctx); err != nil {
		logging.LogFatal(logger, "Data processing error", err)
	}
}
