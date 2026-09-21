package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ckchessmaster/zitadel-actions-api/internal/config"
	"github.com/ckchessmaster/zitadel-actions-api/internal/model"
	"github.com/ckchessmaster/zitadel-actions-api/internal/service"
)

// ActionsHandler is the single unified HTTP handler for ZITADEL Actions webhooks.
type ActionsHandler struct {
	dispatcher *service.Dispatcher
	cfg        *config.Config
	logger     *slog.Logger
}

// NewActionsHandler creates a new ActionsHandler.
func NewActionsHandler(dispatcher *service.Dispatcher, cfg *config.Config, logger *slog.Logger) *ActionsHandler {
	return &ActionsHandler{
		dispatcher: dispatcher,
		cfg:        cfg,
		logger:     logger,
	}
}

// ServeHTTP handles POST requests to the unified /actions endpoint.
func (h *ActionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	opts := model.FlattenOptions{
		ClaimName:       h.cfg.ClaimName,
		RoleFormat:      h.cfg.RoleFormat,
		Lowercase:       h.cfg.LowercaseRoles,
		ProjectIDFilter: h.cfg.FilterProjectID,
	}

	// Query parameter overrides
	q := r.URL.Query()
	if c := q.Get("claim_name"); c != "" {
		opts.ClaimName = strings.TrimSpace(c)
	}
	if f := q.Get("format"); f != "" {
		opts.RoleFormat = strings.ToLower(strings.TrimSpace(f))
	}
	if p := q.Get("project_id"); p != "" {
		opts.ProjectIDFilter = strings.TrimSpace(p)
	}
	if l := q.Get("lowercase"); l != "" {
		if b, err := strconv.ParseBool(l); err == nil {
			opts.Lowercase = b
		}
	}

	// Read and decode request body
	var req model.ActionRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Warn("failed to read request body", slog.Any("error", err))
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	trimmed := bytes.TrimSpace(bodyBytes)
	if len(trimmed) > 0 {
		if err := json.Unmarshal(trimmed, &req); err != nil {
			h.logger.Warn("invalid JSON request body", slog.Any("error", err))
			writeJSONError(w, http.StatusBadRequest, "invalid json request body")
			return
		}
	}

	res, err := h.dispatcher.Dispatch(r.Context(), &req, opts)
	if err != nil {
		h.logger.Error("action dispatch failed", slog.Any("error", err))
		writeJSONError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.logger.Info("action processed successfully",
		slog.String("function", req.Function),
		slog.Int("user_grants_count", len(req.UserGrants)),
		slog.Any("groups", res.Groups),
	)
	h.logger.Debug("action request payload",
		slog.String("request_payload", string(trimmed)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(res)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
