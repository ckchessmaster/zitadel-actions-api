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

// FlattenRolesHandler handles role-flattening webhook requests.
type FlattenRolesHandler struct {
	flattener service.Flattener
	cfg       *config.Config
	logger    *slog.Logger
}

// NewFlattenRolesHandler creates a new FlattenRolesHandler.
func NewFlattenRolesHandler(flattener service.Flattener, cfg *config.Config, logger *slog.Logger) *FlattenRolesHandler {
	return &FlattenRolesHandler{
		flattener: flattener,
		cfg:       cfg,
		logger:    logger,
	}
}

// ServeHTTP implements http.Handler.
func (h *FlattenRolesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	trimmedBody := bytes.TrimSpace(bodyBytes)
	if len(trimmedBody) > 0 {
		if err := json.Unmarshal(trimmedBody, &req); err != nil {
			h.logger.Warn("invalid JSON request body", slog.Any("error", err))
			writeJSONError(w, http.StatusBadRequest, "invalid json request body")
			return
		}
	}

	res := h.flattener.FlattenRoles(&req, opts)

	h.logger.Debug("flattened roles",
		slog.String("claim_name", opts.ClaimName),
		slog.Int("group_count", len(res.Groups)),
		slog.Any("groups", res.Groups),
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
