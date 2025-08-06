package repository

import (
	"context"
	"fmt"
	"github.com/EvgenyiK/subscription-service/internal/config"
	"github.com/EvgenyiK/subscription-service/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
)

type Repository struct {
	db *pgxpool.Pool
}

// NewRepository создает новое подключение к базе данных
func NewRepository(cfg *config.Config) (*Repository, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	pool, err := pgxpool.Connect(context.Background(), dsn)
	if err != nil {
		return nil, err
	}

	return &Repository{db: pool}, nil
}

// Create добавляет новую подписку в базу данных
func (r *Repository) Create(ctx context.Context, sub *models.Subscription) error {
	query := `
        INSERT INTO subscriptions (id, service_name, price, user_id, start_date, end_date)
        VALUES ($1, $2, $3, $4, $5, $6)
    `

	_, err := r.db.Exec(ctx, query,
		sub.ID,
		sub.ServiceName,
		sub.Price,
		sub.UserID,
		sub.StartDate,
		sub.EndDate,
	)
	return err
}

// GetByID возвращает подписку по ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	query := `SELECT id, service_name, price, user_id, start_date, end_date FROM subscriptions WHERE id=$1`
	row := r.db.QueryRow(ctx, query, id)

	var sub models.Subscription

	err := row.Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.Price,
		&sub.UserID,
		&sub.StartDate,
		&sub.EndDate,
	)
	if err != nil {
		return nil, err
	}

	return &sub, nil
}

// Update обновляет существующую подписку
func (r *Repository) Update(ctx context.Context, sub *models.Subscription) error {
	query := `
        UPDATE subscriptions SET 
            service_name=$2,
            price=$3,
            user_id=$4,
            start_date=$5,
            end_date=$6
        WHERE id=$1`

	// Передача NULL для end_date если EndDate == nil
	cmdTag, err := r.db.Exec(ctx, query,
		// Порядок аргументов должен соответствовать порядку в запросе
		sub.ServiceName,
		sub.Price,
		sub.UserID,
		sub.StartDate,
		sub.EndDate,
	)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() != 1 {
		return fmt.Errorf("no rows affected")
	}
	return nil
}

// Delete удаляет подписку по ID
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	cmdTag, err := r.db.Exec(ctx, `DELETE FROM subscriptions WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() != 1 {
		return fmt.Errorf("no rows affected")
	}
	return nil
}

func (r *Repository) List(ctx context.Context, filter models.SubscriptionFilters) ([]models.Subscription, error) {
	query := `SELECT id, service_name, price, user_id, start_date FROM subscriptions WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id=$%d", argIdx)
		args = append(args, *filter.UserID)
		argIdx++
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.Subscription

	for rows.Next() {
		var sub models.Subscription

		if err := rows.Scan(
			&sub.ID,
			&sub.ServiceName,
			&sub.Price,
			&sub.UserID,
			&sub.StartDate,
			&sub.EndDate,
		); err != nil {
			return nil, err
		}

		// Добавляем подписку в список
		subs = append(subs, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subs, nil
}

func (r *Repository) GetSubscriptionsSummary(ctx context.Context, filters models.SubscriptionFilters) (totalCost float64, count int, err error) {
	baseQuery := `SELECT COALESCE(SUM(price), 0), COUNT(*) FROM subscriptions WHERE 1=1`
	args := []interface{}{}
	i := 1

	if filters.UserID != nil {
		baseQuery += fmt.Sprintf(" AND user_id=$%d", i)
		args = append(args, *filters.UserID)
		i++
	}

	if filters.ServiceName != "" {
		baseQuery += fmt.Sprintf(" AND service_name ILIKE $%d", i)
		args = append(args, "%"+filters.ServiceName+"%")
		i++
	}

	if filters.StartDate != nil && filters.EndDate != nil {
		baseQuery += fmt.Sprintf(` AND start_date <= $%d AND (end_date >= $%d OR end_date IS NULL)`, i, i+1)
		args = append(args, *filters.EndDate)
		args = append(args, *filters.StartDate)
	}

	err = r.db.QueryRow(ctx, baseQuery, args...).Scan(&totalCost, &count)
	if err != nil {
		log.Printf("Ошибка при выполнении запроса: %v", err)
	}
	return
}
