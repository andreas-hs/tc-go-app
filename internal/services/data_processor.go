package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andreas-hs/tc-go-app/internal/dependencies"
	"github.com/andreas-hs/tc-go-app/internal/infrastructure/rabbitmq"
	"github.com/andreas-hs/tc-go-app/internal/logging"
	"github.com/andreas-hs/tc-go-app/internal/models"
	"github.com/streadway/amqp"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sync"
	"time"
)

const (
	queueName         = "source_data_queue"
	batchSize         = 100
	generateBatchSize = 100
	maxRetries        = 2
	workerPoolSize    = 5
	prefetchSize      = 10 // QoS prefetch count
)

type DataProcessor struct {
	deps          *dependencies.Dependencies
	wg            sync.WaitGroup
	rabbitChannel *amqp.Channel
	rabbitConn    *amqp.Connection
	workerPool    chan struct{}
	dataBatch     []models.DestinationData
	dataBatchMu   sync.Mutex
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

				dp.dataBatchMu.Lock()
				defer dp.dataBatchMu.Unlock()
				dp.dataBatch = append(dp.dataBatch, data)
				if len(dp.dataBatch) >= batchSize {
					dp.saveBatch(dp.dataBatch)
					dp.dataBatch = nil
				}
				_ = msg.Ack(false)
			}(msg)

		case <-flushTimer.C:
			dp.dataBatchMu.Lock()
			if len(dp.dataBatch) > 0 {
				dp.saveBatch(dp.dataBatch)
				dp.dataBatch = nil
			}
			dp.dataBatchMu.Unlock()
		}
	}
}

func (dp *DataProcessor) saveBatch(dataBatch []models.DestinationData) {
	var lastError error
	orm, connErr := dp.deps.DB.GetConnection()

	if connErr != nil {
		logging.LogFatal(dp.deps.Logger, "failed to connect to database: %w", connErr)
		return
	}

	processedDataBatch := make([]models.ProcessedData, len(dataBatch))
	for i, record := range dataBatch {
		processedDataBatch[i] = models.ProcessedData{SourceID: record.ID}
	}

	for retry := 0; retry < maxRetries; retry++ {
		err := orm.Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"name", "description", "created_at"}),
			}).Create(&dataBatch).Error; err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					dp.saveRecordsIndividually(dataBatch)
					return nil
				}
				return err
			}
			// Only save processedDataBatch for successfully saved dataBatch
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "source_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"source_id"}),
			}).Create(&processedDataBatch).Error; err != nil {
				return err
			}
			return nil
		})
		if err == nil {
			return // Successfully saved
		}

		lastError = err
		time.Sleep(time.Duration(retry) * time.Second)
	}

	if lastError != nil {
		logging.LogError(dp.deps.Logger, "Failed to save batch, retrying individually", lastError)
		dp.saveRecordsIndividually(dataBatch)
	}
}

func (dp *DataProcessor) saveRecordsIndividually(records []models.DestinationData) {
	orm, connErr := dp.deps.DB.GetConnection()
	if connErr != nil {
		logging.LogFatal(dp.deps.Logger, "failed to connect to database: %w", connErr)
		return
	}

	for _, record := range records {
		err := orm.Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"name", "description", "created_at"}),
			}).Create(&record).Error; err != nil {
				return fmt.Errorf("error saving record with ID %d: %w", record.ID, err)
			}
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "source_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"source_id"}),
			}).Create(&models.ProcessedData{SourceID: record.ID}).Error
		})

		if err != nil {
			logging.LogError(dp.deps.Logger, "Error saving record", nil)
		}
	}
}

func (dp *DataProcessor) Stop() error {
	var errs []error

	// Close RabbitMQ resources first to stop receiving new messages
	if err := rabbitmq.CloseRabbitMQ(dp.rabbitChannel, dp.rabbitConn); err != nil {
		errs = append(errs, fmt.Errorf("RabbitMQ shutdown error: %w", err))
	}

	dp.wg.Wait() // Wait for all workers to finish

	// Save any remaining batches that have not been processed
	dp.dataBatchMu.Lock()
	if len(dp.dataBatch) > 0 {
		dp.saveBatch(dp.dataBatch)
		dp.dataBatch = nil
	}
	dp.dataBatchMu.Unlock() // Unlock after processing

	// Close database connection
	if err := dp.deps.DB.Close(); err != nil {
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
