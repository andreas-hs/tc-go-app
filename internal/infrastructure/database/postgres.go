package database

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"time"
)

func ConnectDatabase(dsn string) (*gorm.DB, error) {
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

	return db, nil
}

func CloseDatabase(db *gorm.DB) error {
	if sqlDB, err := db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			return fmt.Errorf("error closing database connection: %w", err)
		}
	}

	return nil
}
