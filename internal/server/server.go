package server

import (
	"github.com/EvgenyiK/subscription-service/internal/handlers"
	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/gorilla/mux"
)

func NewRouter(h *handlers.Handler) *mux.Router {
	r := mux.NewRouter()

	// Группировка маршрутов по пути "/subscriptions"
	subsRouter := r.PathPrefix("/subscriptions").Subrouter()

	// Маршруты для просмотра и подсчета
	subsRouter.HandleFunc("/view/list", h.ListSubscriptions).Methods("GET")
	subsRouter.HandleFunc("/view/total/{date}", h.GetTotalCost).Methods("GET")

	// CRUD операции для подписок
	subsRouter.HandleFunc("", h.CreateSubscription).Methods("POST")
	subsRouter.HandleFunc("/{id:[0-9a-fA-F-]{36}}", h.GetSubscription).Methods("GET")
	subsRouter.HandleFunc("/{id:[0-9a-fA-F-]{36}}", h.UpdateSubscription).Methods("PUT")
	subsRouter.HandleFunc("/{id:[0-9a-fA-F-]{36}}", h.DeleteSubscription).Methods("DELETE")

	subsRouter.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	return r
}
