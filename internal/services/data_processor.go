package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"https://github.com/andreas-hs/nd-go-app/internal/dependencies"
	"https://github.com/andreas-hs/nd-go-app/internal/infrastructure/database"
	"https://github.com/andreas-hs/nd-go-app/internal/infrastructure/rabbitmq"
	"https://github.com/andreas-hs/nd-go-app/internal/logging"
	"https://github.com/andreas-hs/nd-go-app/internal/models"
	"sync"
	"time"
)

const (
	queueName      = "source_data_queue"
	batchSize      = 100
	maxRetries     = 3
	workerPoolSize = 5
	prefetchSize   = 10 // QoS prefetch count
)

type DataProcessor struct {
	deps          *dependencies.Dependencies
	wg            sync.WaitGroup
	rabbitChannel *amqp.Channel
	rabbitConn    *amqp.Connection
	workerPool    chan struct{}
	dataBatch     []models.DestinationData
}

func NewDataProcessor(deps *dependencies.Dependencies) *DataProcessor {
	return &DataProcessor{
		deps:          deps,
		workerPool:    make(chan struct{}, workerPoolSize),
		rabbitChannel: deps.RabbitCh,
		rabbitConn:    deps.RabbitConn,
	}
}

func (dp *DataProcessor) Start(ctx context.Context) error {
	if err := dp.rabbitChannel.Qos(prefetchSize, 0, false); err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := dp.rabbitChannel.Consume(queueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	for i := 0; i < workerPoolSize; i++ {
		dp.wg.Add(1)
		go dp.worker(msgs)
	}

	<-ctx.Done()
	logging.LogInfo(dp.deps.Logger, "Context cancelled, stopping workers")
	return nil
}

func (dp *DataProcessor) worker(msgs <-chan amqp.Delivery) {
	defer dp.wg.Done()

	flushTimer := time.NewTicker(time.Minute)
	defer flushTimer.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			dp.workerPool <- struct{}{}
			go func(msg amqp.Delivery) {
				defer func() { <-dp.workerPool }()
				var data models.DestinationData
				if err := json.Unmarshal(msg.Body, &data); err != nil {
					_ = msg.Nack(false, false)
					return
				}
				dp.dataBatch = append(dp.dataBatch, data)
				if len(dp.dataBatch) >= batchSize {
					dp.saveBatch(dp.dataBatch)
					dp.dataBatch = nil
				}
				_ = msg.Ack(false)
			}(msg)
		case <-flushTimer.C:
			if len(dp.dataBatch) > 0 {
				dp.saveBatch(dp.dataBatch)
				dp.dataBatch = nil
			}
		}
	}
}

func (dp *DataProcessor) saveBatch(dataBatch []models.DestinationData) {
	var lastError error
	for retry := 0; retry < maxRetries; retry++ {
		err := dp.deps.DB.Transaction(func(tx *gorm.DB) error {
			return tx.Create(&dataBatch).Error
		})
		if err == nil {
			lastError = nil
			break
		} else {
			lastError = err
			time.Sleep(time.Duration(retry) * time.Second)
		}
	}

	if lastError != nil {
		dp.saveRecordsIndividually(dataBatch)
	}
}

func (dp *DataProcessor) saveRecordsIndividually(records []models.DestinationData) {
	var ids []uint
	for _, record := range records {
		ids = append(ids, record.ID)
	}

	var existingRecords []models.DestinationData
	if err := dp.deps.DB.Where("id IN ?", ids).Find(&existingRecords).Error; err != nil {
		logging.LogError(dp.deps.Logger, "Error fetching existing records", err)
		return
	}

	existingIDs := make(map[uint]struct{})
	for _, record := range existingRecords {
		existingIDs[record.ID] = struct{}{}
	}

	for _, record := range records {
		if _, exists := existingIDs[record.ID]; exists {
			continue
		}

		err := dp.deps.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&record).Error; err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
				return err
			}
			return nil
		})

		if err != nil {
			logging.LogError(dp.deps.Logger, "Failed to process record after duplicate error", err)
		}
	}
}

func (dp *DataProcessor) Stop() error {
	var errs []error

	// Close RabbitMQ resources first to stop receiving new messages
	if err := rabbitmq.CloseRabbitMQ(dp.rabbitChannel, dp.rabbitConn); err != nil {
		errs = append(errs, fmt.Errorf("RabbitMQ shutdown error: %w", err))
	}

	dp.wg.Wait()

	// Save any remaining batches that have not been processed
	if len(dp.dataBatch) > 0 {
		dp.saveBatch(dp.dataBatch)
		dp.dataBatch = nil
	}

	if err := database.CloseDatabase(dp.deps.DB); err != nil {
		errs = append(errs, fmt.Errorf("database shutdown error: %w", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors occurred during shutdown: %v", errs)
	}

	logging.LogInfo(dp.deps.Logger, "Graceful shutdown completed successfully")
	return nil
}

func (dp *DataProcessor) Wait() {
	dp.wg.Wait()
}
