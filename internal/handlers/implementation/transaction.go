package implementation

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/events"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/repository"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/service"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/validator"
)

type transactionHandler struct {
	svc       service.TransactionService
	idem      repository.IdempotencyRepository
	publisher events.Publisher
	idemMu    sync.Mutex
}

func newTransactionHandler(svc service.TransactionService, idem repository.IdempotencyRepository, publisher events.Publisher) *transactionHandler {
	return &transactionHandler{svc: svc, idem: idem, publisher: publisher}
}

func (h *transactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var req service.CreateTransactionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idemKey != "" {
		if err := validator.ValidateUUID(req.UserID); err != nil {
			handleServiceError(w, err)
			return
		}
		h.createTransactionIdempotent(w, r, body, &req, idemKey)
		return
	}

	h.createTransactionOnce(w, r, &req)
}

func (h *transactionHandler) createTransactionIdempotent(
	w http.ResponseWriter,
	r *http.Request,
	body []byte,
	req *service.CreateTransactionRequest,
	idemKey string,
) {
	ctx := r.Context()
	sum := sha256.Sum256(body)

	h.idemMu.Lock()
	defer h.idemMu.Unlock()

	rec, err := h.idem.Get(ctx, req.UserID, idemKey)
	if err != nil && !errors.Is(err, apperrors.ErrNotFound) {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if rec != nil {
		if !bytes.Equal(sum[:], rec.BodyHash) {
			respondError(w, http.StatusConflict, "idempotency key conflict: request body differs from the first request")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rec.HTTPStatus)
		_, _ = w.Write(rec.ResponseJSON)
		return
	}

	tx, err := h.svc.CreateTransaction(ctx, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	observability.RecordTransactionCreated(tx.Type, tx.Status)
	h.publishTransactionCreatedEvent(ctx, tx)

	payload := map[string]interface{}{
		"status": "success",
		"data":   tx,
	}
	respBytes, err := json.Marshal(payload)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := h.idem.Save(ctx, req.UserID, idemKey, sum[:], http.StatusCreated, respBytes); err != nil {
		log.Printf("idempotency save failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

func (h *transactionHandler) createTransactionOnce(w http.ResponseWriter, r *http.Request, req *service.CreateTransactionRequest) {
	ctx := r.Context()
	tx, err := h.svc.CreateTransaction(ctx, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	observability.RecordTransactionCreated(tx.Type, tx.Status)
	h.publishTransactionCreatedEvent(ctx, tx)

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"status": "success",
		"data":   tx,
	})
}

func (h *transactionHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	txID := r.PathValue("id")
	userID := r.URL.Query().Get("user_id")

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

func (h *transactionHandler) publishTransactionCreatedEvent(ctx context.Context, tx *models.Transaction) {
	if h.publisher == nil {
		return
	}

	if err := h.publisher.PublishTransactionCreated(ctx, tx); err != nil {
		log.Printf("kafka publish transaction.created failed: %v", err)
	}
}
