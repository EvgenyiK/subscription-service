package repository

import (
	"context"
	"fmt"
	"github.com/EvgenyiK/subscription-service/internal/config"
	"github.com/EvgenyiK/subscription-service/internal/models"
	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
	"log"
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

// Create добавляет новую подписку в базу данных с помощью Squirrel
func (r *Repository) Create(ctx context.Context, sub *models.Subscription) error {
	queryBuilder := squirrel.Insert("subscriptions").
		Columns("id", "service_name", "price", "user_id", "start_date", "end_date").
		Values(sub.ID, sub.ServiceName, sub.Price, sub.UserID, sub.StartDate, sub.EndDate).
		PlaceholderFormat(squirrel.Dollar)

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Printf("Create: ошибка формирования SQL: %v", err)
		return err
	}

	log.Printf("Create: выполняется SQL: %s с аргументами: %v", sqlStr, args)

	_, err = r.db.Exec(ctx, sqlStr, args...)
	return err
}

// GetByID возвращает подписку по user_id
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	queryBuilder := squirrel.Select("id", "service_name", "price", "user_id", "start_date", "end_date").
		From("subscriptions").
		Where(squirrel.Eq{"user_id": id}).PlaceholderFormat(squirrel.Dollar)

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Printf("GetByID: ошибка формирования SQL: %v", err)
		return nil, err
	}

	log.Printf("GetByID: выполняется SQL: %s с аргументами: %v", sqlStr, args)

	var sub models.Subscription

	row := r.db.QueryRow(ctx, sqlStr, args...)
	err = row.Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.Price,
		&sub.UserID,
		&sub.StartDate,
		&sub.EndDate,
	)
	if err != nil {
		log.Printf("GetByID: ошибка при сканировании результата: %v", err)
		return nil, err
	}

	log.Printf("GetByID: успешно получена подписка с ID: %s", sub.ID)
	return &sub, nil
}

// Update обновляет существующую подписку
func (r *Repository) Update(ctx context.Context, sub *models.Subscription) error {
	queryBuilder := squirrel.Update("subscriptions").
		Set("service_name", sub.ServiceName).
		Set("price", sub.Price).
		Set("start_date", sub.StartDate).
		Set("end_date", sub.EndDate).
		Where(squirrel.Eq{"user_id": sub.UserID}).PlaceholderFormat(squirrel.Dollar)

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Printf("Update: ошибка формирования SQL: %v", err)
		return err
	}

	log.Printf("Update: выполняется SQL: %s с аргументами: %v", sqlStr, args)

	cmdTag, err := r.db.Exec(ctx, sqlStr, args...)
	if err != nil {
		log.Printf("Update: ошибка выполнения SQL: %v", err)
		return err
	}
	if cmdTag.RowsAffected() != 1 {
		log.Printf("Update: строк не обновлено (RowsAffected=%d)", cmdTag.RowsAffected())
		return fmt.Errorf("no rows affected")
	}

	log.Printf("Update: успешно обновлена подписка для user_id=%s", sub.UserID)
	return nil
}

// Delete удаляет подписку по ID
func (r *Repository) Delete(ctx context.Context, userID uuid.UUID) error {
	queryBuilder := squirrel.Delete("subscriptions").
		Where(squirrel.Eq{"user_id": userID}).PlaceholderFormat(squirrel.Dollar)

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Printf("Update: успешно обновлена подписка для user_id=%s", userID)
		return err
	}

	log.Printf("Delete: выполняется SQL: %s с аргументами: %v", sqlStr, args)

	cmdTag, err := r.db.Exec(ctx, sqlStr, args...)
	if err != nil {
		log.Printf("Delete: ошибка выполнения SQL: %v", err)
		return err
	}
	if cmdTag.RowsAffected() != 1 {
		log.Printf("Delete: строк не удалено (RowsAffected=%d)", cmdTag.RowsAffected())
		return fmt.Errorf("no rows affected")
	}

	log.Printf("Delete: успешно удалена подписка для user_id=%s", userID)
	return nil
}

// Получение всех подписок
func (r *Repository) GetAllSubscriptions(ctx context.Context) ([]models.Subscription, error) {
	queryBuilder := squirrel.Select("id", "service_name", "price", "user_id", "start_date", "end_date").
		From("subscriptions").PlaceholderFormat(squirrel.Dollar)

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Printf("GetAllSubscriptions : ошибка формирования SQL :%v", err)
		return nil, err
	}

	log.Printf("GetAllSubscriptions : выполняется SQL :%s с аргументами :%v", sqlStr, args)

	rows, err := r.db.Query(ctx, sqlStr, args...)
	if err != nil {
		log.Printf("GetAllSubscriptions : ошибка выполнения запроса :%v", err)
		return nil, err
	}
	defer rows.Close()

	var subs []models.Subscription
	for rows.Next() {
		var s models.Subscription
		err := rows.Scan(&s.ID, &s.ServiceName, &s.Price, &s.UserID, &s.StartDate, &s.EndDate)
		if err != nil {
			log.Printf("GetAllSubscriptions : ошибка сканирования строки :%v", err)
			return nil, err
		}
		subs = append(subs, s)
	}

	log.Println("GetAllSubscriptions : успешно получено количество подписок:", len(subs))
	return subs, nil
}

// Подсчет стоимости подписки по указанной дате в запросе
func (r *Repository) GetTotalSubscriptionCost(
	ctx context.Context,
	date time.Time,
	filterByUser bool,
	userID uuid.UUID,
	serviceName string,
) (int, error) {

	queryBuilder := squirrel.Select("COALESCE(SUM(price), 0)").From("subscriptions").
		Where(
			squirrel.And{
				squirrel.LtOrEq{"start_date": date},
				squirrel.GtOrEq{"end_date": date},
			},
		).PlaceholderFormat(squirrel.Dollar)

	if filterByUser {
		queryBuilder = queryBuilder.Where(squirrel.Eq{"user_id": userID})
	}

	if serviceName != "" {
		queryBuilder = queryBuilder.Where(squirrel.Eq{"service_name": serviceName})
	}

	sqlStr, args, err := queryBuilder.ToSql()
	if err != nil {
		log.Printf("GetTotalSubscriptionCost : ошибка формирования SQL :%v", err)
		return 0, err
	}

	log.Printf("GetTotalSubscriptionCost : выполняется SQL :%s с аргументами :%v", sqlStr, args)

	var total int
	err = r.db.QueryRow(ctx, sqlStr, args...).Scan(&total)
	if err != nil {
		log.Printf("GetTotalSubscriptionCost : ошибка выполнения запроса :%v", err)
		return 0, err
	}

	log.Println("GetTotalSubscriptionCost : успешно получена сумма:", total)
	return total, nil
}
