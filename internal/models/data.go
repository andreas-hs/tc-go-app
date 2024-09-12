package models

import (
	"time"
)

type DestinationData struct {
	ID          uint `gorm:"primaryKey"`
	Name        string
	Description string
	CreatedAt   time.Time
}
