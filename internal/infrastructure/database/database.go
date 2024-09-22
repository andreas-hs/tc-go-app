package database

import "gorm.io/gorm"

type Database interface {
	Connect(dsn string) (*gorm.DB, error)
	GetConnection() (*gorm.DB, error)
	Close() error
}
