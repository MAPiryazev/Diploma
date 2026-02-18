package handlers

import "net/http"

type TransactionHandler interface {
	CreateTransaction(w http.ResponseWriter, r *http.Request)
	GetTransaction(w http.ResponseWriter, r *http.Request)
	ListTransactions(w http.ResponseWriter, r *http.Request)
	UpdateTransaction(w http.ResponseWriter, r *http.Request)
	DeleteTransaction(w http.ResponseWriter, r *http.Request)
}

type AnalyticsHandler interface {
	GetAnalytics(w http.ResponseWriter, r *http.Request)
}

type HealthHandler interface {
	Health(w http.ResponseWriter, r *http.Request)
}

type Handlers interface {
	Transaction() TransactionHandler
	Analytics() AnalyticsHandler
	Health() HealthHandler
}
