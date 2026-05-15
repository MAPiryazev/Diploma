package http

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	apperrors "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/errors"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/middleware"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/models"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/observability"
	transactionapp "github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/services/transactionapi/app"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/shared/platform/httpapi"
	"github.com/MAPiryazev/Wildberries_L1/tree/main/L3/L3.6/internal/validator"
)

const maxJSONBodyBytes = 1 << 20
const defaultTransactionListLimit = 100
const maxTransactionListLimit = 500

type IdempotencyStore interface {
	Get(ctx context.Context, userID, idempotencyKey string) (*IdempotencyRecord, error)
	Save(ctx context.Context, userID, idempotencyKey string, bodyHash []byte, httpStatus int, responseJSON []byte) error
}

type AuditStore interface {
	Create(ctx context.Context, log *models.AuditLog) error
}

type IdempotencyRecord struct {
	BodyHash     []byte
	HTTPStatus   int
	ResponseJSON []byte
}

type Handler struct {
	svc    transactionapp.TransactionService
	idem   IdempotencyStore
	audit  AuditStore
	idemMu sync.Mutex
}

func NewHandler(svc transactionapp.TransactionService, idem IdempotencyStore, audit AuditStore) *Handler {
	return &Handler{svc: svc, idem: idem, audit: audit}
}

func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpapi.AuthenticatedUserID(w, r)
	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		httpapi.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var req transactionapp.CreateTransactionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpapi.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID != "" && req.UserID != userID {
		httpapi.RespondError(w, http.StatusForbidden, "user_id does not match authenticated principal")
		return
	}
	req.UserID = userID

	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idemKey != "" {
		if err := validator.ValidateUUID(req.UserID); err != nil {
			httpapi.HandleServiceError(w, err)
			return
		}
		h.createTransactionIdempotent(w, r, body, &req, idemKey)
		return
	}

	h.createTransactionOnce(w, r, &req)
}

func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	txID := r.PathValue("id")
	userID, ok := httpapi.AuthenticatedUserID(w, r)
	if !ok {
		return
	}
	if txID == "" {
		httpapi.RespondError(w, http.StatusBadRequest, "id is required")
		return
	}
	if !httpapi.QueryUserMatchesPrincipal(w, r, userID) {
		return
	}

	tx, err := h.svc.GetTransaction(r.Context(), txID, userID)
	if err != nil {
		httpapi.HandleServiceError(w, err)
		return
	}
	httpapi.RespondSuccess(w, http.StatusOK, tx)
}

func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpapi.AuthenticatedUserID(w, r)
	if !ok {
		return
	}
	if !httpapi.QueryUserMatchesPrincipal(w, r, userID) {
		return
	}

	limit, offset, ok := parseTransactionListPage(w, r)
	if !ok {
		return
	}

	txs, err := h.svc.ListTransactions(r.Context(), userID, limit, offset)
	if err != nil {
		httpapi.HandleServiceError(w, err)
		return
	}

	httpapi.RespondJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data":   txs,
		"count":  len(txs),
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := httpapi.AuthenticatedUserID(w, r)
	if !ok {
		return
	}

	txID := r.PathValue("id")
	if txID == "" {
		httpapi.RespondError(w, http.StatusBadRequest, "id is required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	var req transactionapp.UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpapi.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID != "" && req.UserID != userID {
		httpapi.RespondError(w, http.StatusForbidden, "user_id does not match authenticated principal")
		return
	}

	req.ID = txID
	req.UserID = userID
	ctx := r.Context()
	if err := h.svc.UpdateTransaction(ctx, &req); err != nil {
		h.recordAudit(ctx, "transaction.update", &txID, "failure")
		httpapi.HandleServiceError(w, err)
		return
	}
	h.recordAudit(ctx, "transaction.update", &txID, "success")

	httpapi.RespondJSON(w, http.StatusOK, map[string]any{
		"status": "success",
	})
}

func (h *Handler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	txID := r.PathValue("id")
	userID, ok := httpapi.AuthenticatedUserID(w, r)
	if !ok {
		return
	}
	if txID == "" {
		httpapi.RespondError(w, http.StatusBadRequest, "id is required")
		return
	}
	if !httpapi.QueryUserMatchesPrincipal(w, r, userID) {
		return
	}

	ctx := r.Context()
	if err := h.svc.DeleteTransaction(ctx, txID, userID); err != nil {
		h.recordAudit(ctx, "transaction.delete", &txID, "failure")
		httpapi.HandleServiceError(w, err)
		return
	}
	h.recordAudit(ctx, "transaction.delete", &txID, "success")

	httpapi.RespondJSON(w, http.StatusOK, map[string]any{
		"status": "success",
	})
}

func (h *Handler) createTransactionIdempotent(
	w http.ResponseWriter,
	r *http.Request,
	body []byte,
	req *transactionapp.CreateTransactionRequest,
	idemKey string,
) {
	start := time.Now()
	ctx := r.Context()
	sum := sha256.Sum256(body)

	h.idemMu.Lock()
	defer h.idemMu.Unlock()

	rec, err := h.idem.Get(ctx, req.UserID, idemKey)
	if err != nil && !errors.Is(err, apperrors.ErrNotFound) {
		httpapi.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if rec != nil {
		if !bytes.Equal(sum[:], rec.BodyHash) {
			observability.ObserveAPIUseCase("idempotency.conflict", "error", time.Since(start))
			httpapi.RespondError(w, http.StatusConflict, "idempotency key conflict: request body differs from the first request")
			return
		}
		observability.ObserveAPIUseCase("idempotency.replay", "success", time.Since(start))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rec.HTTPStatus)
		_, _ = w.Write(rec.ResponseJSON)
		return
	}

	tx, err := h.svc.CreateTransaction(ctx, req)
	if err != nil {
		h.recordAudit(ctx, "transaction.create", nil, "failure")
		httpapi.HandleServiceError(w, err)
		return
	}
	h.recordAudit(ctx, "transaction.create", &tx.ID, "success")

	observability.RecordTransactionCreated(tx.Type, tx.Status)

	payload := map[string]any{
		"status": "success",
		"data":   tx,
	}
	respBytes, err := json.Marshal(payload)
	if err != nil {
		httpapi.RespondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := h.idem.Save(ctx, req.UserID, idemKey, sum[:], http.StatusCreated, respBytes); err != nil {
		log.Printf("idempotency save failed: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

func (h *Handler) createTransactionOnce(w http.ResponseWriter, r *http.Request, req *transactionapp.CreateTransactionRequest) {
	ctx := r.Context()
	tx, err := h.svc.CreateTransaction(ctx, req)
	if err != nil {
		h.recordAudit(ctx, "transaction.create", nil, "failure")
		httpapi.HandleServiceError(w, err)
		return
	}
	h.recordAudit(ctx, "transaction.create", &tx.ID, "success")

	observability.RecordTransactionCreated(tx.Type, tx.Status)
	httpapi.RespondSuccess(w, http.StatusCreated, tx)
}

func (h *Handler) recordAudit(ctx context.Context, action string, entityID *string, result string) {
	if h.audit == nil {
		return
	}
	principal, ok := middleware.GetPrincipal(ctx)
	if !ok || principal.UserID == "" {
		return
	}
	if err := h.audit.Create(ctx, &models.AuditLog{
		ActorID:    principal.UserID,
		ActorRole:  principal.Role,
		Action:     action,
		EntityType: "transaction",
		EntityID:   entityID,
		Result:     result,
		RequestID:  middleware.GetRequestID(ctx),
	}); err != nil {
		log.Printf("audit log write failed: action=%s result=%s err=%v", action, result, err)
	}
}

func parseTransactionListPage(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	limit := defaultTransactionListLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > maxTransactionListLimit {
			httpapi.RespondError(w, http.StatusBadRequest, "limit must be between 1 and 500")
			return 0, 0, false
		}
		limit = parsed
	}

	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			httpapi.RespondError(w, http.StatusBadRequest, "offset must be non-negative")
			return 0, 0, false
		}
		offset = parsed
	}

	return limit, offset, true
}
