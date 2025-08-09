package repository

import (
	"context"
	"fmt"
	"github.com/EvgenyiK/subscription-service/internal/config"
	"github.com/EvgenyiK/subscription-service/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"strconv"
	"time"
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
	query := `SELECT id, service_name, price, user_id, start_date, end_date FROM subscriptions WHERE user_id=$1`
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
            start_date=$5,
            end_date=$6
        WHERE user_id=$4`

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
func (r *Repository) Delete(ctx context.Context, user_id uuid.UUID) error {
	cmdTag, err := r.db.Exec(ctx, `DELETE FROM subscriptions WHERE user_id=$1`, user_id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() != 1 {
		return fmt.Errorf("no rows affected")
	}
	return nil
}

func (r *Repository) GetAllSubscriptions(ctx context.Context) ([]models.Subscription, error) {
	rows, err := r.db.Query(ctx, "SELECT id, service_name, price, user_id, start_date, end_date FROM subscriptions")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.Subscription
	for rows.Next() {
		var s models.Subscription
		err := rows.Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &s.StartDate, &s.EndDate)
		if err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *Repository) GetTotalSubscriptionCost(ctx context.Context, date time.Time, filterByUser bool, userID uuid.UUID, serviceName string) (int, error) {
	query := `
		SELECT COALESCE(SUM(price), 0)
		FROM subscriptions
		WHERE start_date <= TO_DATE($1, 'YYYY-MM-DD') AND end_date >= TO_DATE($1, 'YYYY-MM-DD')`

	var args []interface{}
	args = append(args, date)

	argPos := 2 // позиция следующего аргумента

	if filterByUser {
		query += " AND user_id = $" + strconv.Itoa(argPos)
		args = append(args, userID)
		argPos++
	}

	if serviceName != "" {
		query += " AND service_name = $" + strconv.Itoa(argPos)
		args = append(args, serviceName)
	}

	var total int
	err := r.db.QueryRow(ctx, query, args...).Scan(&total)
	fmt.Println(err)
	return total, err
}
