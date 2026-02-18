package implementation

import (
	"encoding/json"
	"net/http"

	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/service"
)

type transactionHandler struct {
	svc service.TransactionService
}

func newTransactionHandler(svc service.TransactionService) *transactionHandler {
	return &transactionHandler{svc: svc}
}

func (h *transactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req service.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ctx := r.Context()
	tx, err := h.svc.CreateTransaction(ctx, &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"status": "success",
		"data":   tx,
	})
}

func (h *transactionHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	txID := r.PathValue("id")
	userID := r.PathValue("user_id")

	if txID == "" || userID == "" {
		respondError(w, http.StatusBadRequest, "id and user_id are required")
		return
	}

	ctx := r.Context()
	tx, err := h.svc.GetTransaction(ctx, txID, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   tx,
	})
}

func (h *transactionHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	ctx := r.Context()
	txs, err := h.svc.ListTransactions(ctx, userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   txs,
		"count":  len(txs),
	})
}

func (h *transactionHandler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	txID := r.PathValue("id")
	if txID == "" {
		respondError(w, http.StatusBadRequest, "id is required")
		return
	}

	var req service.UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.ID = txID
	ctx := r.Context()
	if err := h.svc.UpdateTransaction(ctx, &req); err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
	})
}

func (h *transactionHandler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	txID := r.PathValue("id")
	userID := r.URL.Query().Get("user_id")

	if txID == "" || userID == "" {
		respondError(w, http.StatusBadRequest, "id and user_id are required")
		return
	}

	ctx := r.Context()
	if err := h.svc.DeleteTransaction(ctx, txID, userID); err != nil {
		handleServiceError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status": "success",
	})
}
