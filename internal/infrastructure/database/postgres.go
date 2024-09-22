package database

import (
	"fmt"
	"github.com/andreas-hs/tc-go-app/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"time"
)

type PostgresDatabase struct {
	connection *gorm.DB
}

func (p *PostgresDatabase) Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("gorm open error: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("sqlDB initialization error: %w", err)
	}

	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	// AutoMigrate for source_data and destination_data tables
	if err := db.AutoMigrate(
		&models.SourceData{},
		&models.DestinationData{},
		&models.ProcessedData{},
	); err != nil {
		return nil, fmt.Errorf("auto migrate error: %w", err)
	}

	p.connection = db

	return db, nil
}

func (p *PostgresDatabase) GetConnection() (*gorm.DB, error) {
	if p.connection == nil {
		return nil, fmt.Errorf("connection not initialized")
	}
	return p.connection, nil
}

func (p *PostgresDatabase) Close() error {
	if sqlDB, err := p.connection.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			return fmt.Errorf("error closing database connection: %w", err)
		}
	}

	return nil
}
