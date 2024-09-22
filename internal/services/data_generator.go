package services

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/andreas-hs/tc-go-app/internal/dependencies"
	"github.com/andreas-hs/tc-go-app/internal/infrastructure/database"
	"github.com/andreas-hs/tc-go-app/internal/logging"
	"github.com/andreas-hs/tc-go-app/internal/models"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/streadway/amqp"
	"time"
)

type DataGenerator struct {
	rabbitCh *amqp.Channel
	db       database.Database
}

func NewDataGenerator(deps *dependencies.Dependencies) *DataGenerator {
	return &DataGenerator{
		rabbitCh: deps.RabbitCh,
		db:       deps.DB,
	}
}

func TriggerDataGeneration(ctx context.Context, deps *dependencies.Dependencies, count int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		generator := NewDataGenerator(deps)
		if err := generator.generateAndSend(count); err != nil {
			logging.LogError(deps.Logger, "Failed to generate and send data", err)
			return err
		}
	}
	return nil
}

func (g *DataGenerator) generateAndSend(numRecords int) error {
	var sourceDataBatch []models.SourceData

	// Generate data
	for i := 0; i < numRecords; i++ {
		data := models.SourceData{
			DataItem: models.DataItem{
				Name:        gofakeit.Name(),
				Description: gofakeit.Sentence(5),
				CreatedAt:   time.Now(),
			},
		}
		sourceDataBatch = append(sourceDataBatch, data)
	}

	// Attempt to write to DB with retries
	maxDBRetries := 3
	var lastError error
	for attempt := 1; attempt <= maxDBRetries; attempt++ {
		orm, err := g.db.GetConnection()
		if err != nil {
			lastError = fmt.Errorf("failed to get database connection: %w", err)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}

		if err := orm.Create(&sourceDataBatch).Error; err != nil {
			lastError = fmt.Errorf("failed to save source data, attempt %d: %w", attempt, err)
			time.Sleep(time.Duration(attempt) * time.Second)
		} else {
			lastError = nil
			break
		}
	}

	if lastError != nil {
		fmt.Printf("All attempts to save data failed: %v\n", lastError)
		return lastError
	}

	// Send data to RabbitMQ
	for _, data := range sourceDataBatch {
		body, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal source data: %w", err)
		}
		if err := g.rabbitCh.Publish("", queueName, false, false, amqp.Publishing{Body: body}); err != nil {
			return fmt.Errorf("failed to publish message to RabbitMQ: %w", err)
		}
	}

	return nil
}
