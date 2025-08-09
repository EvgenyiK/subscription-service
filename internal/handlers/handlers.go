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
	"time"

	"github.com/google/uuid"
	"log"
)

const (
	dateFormatStart = "01-2006"
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

	userUUID, err := parseUUID(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid subscription ID format")
		return
	}

	// Получаем существующую подписку
	subscription, err := h.repo.GetByID(r.Context(), userUUID)
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
	userUUID, err := parseUUID(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid subscription ID format")
		return
	}

	// Вызов метода удаления
	err = h.repo.Delete(r.Context(), userUUID)
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
	// Просто получаем все подписки без фильтров
	subscriptions, err := h.repo.GetAllSubscriptions(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error fetching subscriptions: "+err.Error())
		return
	}

	// Отправляем результат в формате JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subscriptions)
}

// GetTotalCost подсчитывает сумму подписок за выбранный период
func (h *Handler) GetTotalCost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	dateStr := vars["date"] // например, "2023-10-15"

	// Парсим дату
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid date format")
		return
	}

	// Получаем фильтры из query-параметров
	userIDStr := r.URL.Query().Get("user_id")
	serviceName := r.URL.Query().Get("service_name")

	var userUUID uuid.UUID
	var filterByUser bool

	if userIDStr != "" {
		userUUID, err = parseUUID(userIDStr)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid subscription ID format")
			return
		}
		filterByUser = true
	}

	// Вызов вашей функции подсчета
	totalCost, err := h.repo.GetTotalSubscriptionCost(r.Context(), date, filterByUser, userUUID, serviceName)
	if err != nil {
		http.Error(w, "Error calculating total cost", http.StatusInternalServerError)
		return
	}

	// Отправляем ответ в JSON
	resp := map[string]interface{}{
		"date":  dateStr,
		"total": totalCost,
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
