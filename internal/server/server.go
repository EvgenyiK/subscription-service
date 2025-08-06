package server

import (
	"github.com/EvgenyiK/subscription-service/internal/handlers"

	"github.com/gorilla/mux"
)

func NewRouter(h *handlers.Handler) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/subscriptions", h.CreateSubscription).Methods("POST")
	r.HandleFunc("/subscriptions/{id}", h.GetSubscription).Methods("GET")
	r.HandleFunc("/subscriptions/{id}", h.UpdateSubscription).Methods("PUT")
	r.HandleFunc("/subscriptions/{id}", h.DeleteSubscription).Methods("DELETE")
	r.HandleFunc("/subscriptions", h.ListSubscriptions).Methods("GET") // с фильтрами

	r.HandleFunc("/subscriptions/total", h.GetTotalCost).Methods("GET") // подсчет стоимости

	return r
}
