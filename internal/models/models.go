package models

import (
	"github.com/google/uuid"
	"time"
)

type Subscription struct {
	ID          uuid.UUID  `json:"id"`
	ServiceName string     `json:"service_name"`
	Price       int        `json:"price"` // в рублях, целое число
	UserID      uuid.UUID  `json:"user_id"`
	StartDate   time.Time  `json:"start_date"`         // месяц и год, например 07-2025
	EndDate     *time.Time `json:"end_date,omitempty"` // опционально
}
