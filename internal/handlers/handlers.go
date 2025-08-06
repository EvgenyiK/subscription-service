package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/EvgenyiK/subscription-service/internal/models"
	"github.com/EvgenyiK/subscription-service/internal/repository"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"log"
)

const (
	dateFormatStart = "01-2006"
	dateFormatQuery = "2006-01-02"
)

type Handler struct {
	repo *repository.Repository
}

func NewHandler(repo *repository.Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ServiceName string  `json:"service_name"`
		Price       int     `json:"price"`
		UserID      string  `json:"user_id"`
		StartDate   string  `json:"start_date"` // формат "07-2025"
		EndDate     *string `json:"end_date,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}

	if input.ServiceName == "" || input.UserID == "" || input.StartDate == "" || input.Price <= 0 {
		respondWithError(w, http.StatusBadRequest, "Missing required fields")
		return
	}

	userUUID, err := parseUUID(input.UserID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user_id format")
		return
	}

	startTime, err := parseDate(dateFormatStart, input.StartDate)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid start_date format")
		return
	}

	var endTime *time.Time
	if input.EndDate != nil && *input.EndDate != "" {
		endTimeParsed, err := parseDate(dateFormatStart, *input.EndDate)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid end_date format")
			return
		}
		endTime = endTimeParsed
	} else {
		newEndTime := startTime.Add(30 * 24 * time.Hour)
		endTime = &newEndTime
	}

	sub := models.Subscription{
		ID:          uuid.New(),
		ServiceName: input.ServiceName,
		Price:       input.Price,
		UserID:      userUUID,
		StartDate:   *startTime,
		EndDate:     endTime,
	}

	if err := h.repo.Create(context.Background(), &sub); err != nil {
		log.Println("Failed to create subscription:", err)
		respondWithError(w, http.StatusInternalServerError, "Failed to create subscription")
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sub)
}

func (h *Handler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	userUUID, err := parseUUID(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user_id format")
		return
	}

	subscription, err := h.repo.GetByID(r.Context(), userUUID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Subscription not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscription)
}

func (h *Handler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	subscriptionID, err := parseUUID(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid subscription ID format")
		return
	}

	// Получаем существующую подписку
	subscription, err := h.repo.GetByID(r.Context(), subscriptionID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Subscription not found")
		return
	}

	// Парсим тело запроса для новых данных
	var updateData struct {
		ServiceName string     `json:"service_name"`
		Price       int        `json:"price"`
		UserID      uuid.UUID  `json:"user_id"`
		StartDate   time.Time  `json:"start_date"`
		EndDate     *time.Time `json:"end_date"` // nullable
	}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Обновляем поля подписки
	subscription.ServiceName = updateData.ServiceName
	subscription.Price = updateData.Price
	subscription.UserID = updateData.UserID
	subscription.StartDate = updateData.StartDate
	subscription.EndDate = updateData.EndDate

	// Обновляем в базе данных
	if err := h.repo.Update(r.Context(), subscription); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update subscription")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscription)
}

func (h *Handler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	// Получение переменной из URL
	vars := mux.Vars(r)
	idStr := vars["id"]

	// Парсинг UUID
	subscriptionID, err := parseUUID(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid subscription ID format")
		return
	}

	// Вызов метода удаления
	err = h.repo.Delete(r.Context(), subscriptionID)
	if err != nil {
		// Можно уточнить ошибку: если не найден — 404, иначе 500
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, http.StatusNotFound, "Subscription not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, "Failed to delete subscription")
		}
		return
	}

	// Успешное удаление — статус No Content
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	// Получение query-параметров
	queryParams := r.URL.Query()

	var filter models.SubscriptionFilters

	// Получение user_id из query-параметра
	if userIDStr := queryParams.Get("user_id"); userIDStr != "" {
		userIDInt, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid user_id parameter")
			return
		}
		filter.UserID = &userIDInt
	}

	// Можно добавить другие фильтры по необходимости

	// Вызов метода репозитория
	subscriptions, err := h.repo.List(r.Context(), filter)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to fetch subscriptions")
		return
	}

	// Отправка ответа
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(subscriptions); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to encode response")
		return
	}
}

func (h *Handler) GetTotalCost(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	query := req.URL.Query()

	var (
		userIDStr    = query.Get("user_id")
		serviceName  = query.Get("service_name")
		startDateStr = query.Get("start_date")
		endDateStr   = query.Get("end_date")
	)

	var (
		userID    *int64
		startDate *time.Time
		endDate   *time.Time
		err       error
	)

	if userIDStr != "" {
		idVal, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid user_id")
			return
		}
		userID = &idVal
	}

	if startDateStr != "" {
		t, err := time.Parse("2006-01-02", startDateStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid start_date format. Use YYYY-MM-DD")
			return
		}
		startDate = &t
	}

	if endDateStr != "" {
		t, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid end_date format. Use YYYY-MM-DD")
			return
		}
		endDate = &t
	} else {
		now := time.Now()
		endDate = &now
	}

	filters := models.SubscriptionFilters{
		UserID:      userID,
		ServiceName: serviceName,
		StartDate:   startDate,
		EndDate:     endDate,
	}

	totalCost, count, err := h.repo.GetSubscriptionsSummary(ctx, filters)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	resp := map[string]interface{}{
		"total_cost": totalCost,
		"count":      count,
		"filters": map[string]interface{}{
			"user_id":      userIDStr,
			"service_name": serviceName,
			"start_date":   startDateStr,
			"end_date":     endDateStr,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func parseDate(layout, dateStr string) (*time.Time, error) {
	t, err := time.Parse(layout, dateStr)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// парсинг userID
func parseUUID(userIDStr string) (uuid.UUID, error) {
	return uuid.Parse(userIDStr)
}

// Обработка ошибок с логированием
func respondWithError(w http.ResponseWriter, status int, message string) {
	log.Printf("Error: %s", message)
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
