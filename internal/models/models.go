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

// CreateSubscriptionInput представляет входные данные для создания подписки.
// swagger:model
type CreateSubscriptionInput struct {
	ServiceName string    `json:"service_name" example:"Netflix"`
	Price       int       `json:"price" example:"10"`
	UserID      uuid.UUID `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	StartDate   string    `json:"start_date" example:"07-2025"` // формат "01-2006"
	EndDate     *string   `json:"end_date,omitempty" example:"08-2025"`
}

// UpdateSubscriptionInput представляет данные для обновления подписки.
// swagger:model
type UpdateSubscriptionInput struct {
	ServiceName string     `json:"service_name" example:"Netflix"`
	Price       int        `json:"price" example:"15"`
	UserID      uuid.UUID  `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	StartDate   time.Time  `json:"start_date" example:"2025-07-01"` // формат ISO8601
	EndDate     *time.Time `json:"end_date,omitempty" example:"2025-08-01"`
}
