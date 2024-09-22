package models

import (
	"time"
)

type DataItem struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	Description string
	CreatedAt   time.Time
}

type SourceData struct {
	DataItem
}

type DestinationData struct {
	DataItem
}

type ProcessedData struct {
	ID          uint      `gorm:"primaryKey"`
	SourceID    uint      `gorm:"unique"`
	ProcessedAt time.Time `gorm:"autoUpdateTime"`
}
