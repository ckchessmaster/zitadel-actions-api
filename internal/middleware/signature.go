package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/zitadel/zitadel-go/v3/pkg/actions"
)

// SignatureValidator returns an HTTP middleware that verifies incoming requests
// using the ZITADEL Target signing key. If signingKey is empty, validation is bypassed.
func SignatureValidator(signingKey string, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no signing key configured, skip validation
			if strings.TrimSpace(signingKey) == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Read request body
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Warn("failed to read request body for signature validation", slog.Any("error", err))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unable to read request body"})
				return
			}

			// Restore body for downstream handlers
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// Check for Zitadel-Signature or X-Zitadel-Signature
			sigHeader := r.Header.Get("Zitadel-Signature")
			if sigHeader == "" {
				sigHeader = r.Header.Get("X-Zitadel-Signature")
			}

			if sigHeader == "" {
				logger.Warn("missing ZITADEL signature header", slog.String("path", r.URL.Path))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing ZITADEL-Signature header"})
				return
			}

			// Validate signature with a 5-minute tolerance for clock skew
			if err := actions.ValidatePayloadWithTolerance(bodyBytes, sigHeader, signingKey, 5*time.Minute); err != nil {
				logger.Warn("invalid ZITADEL signature",
					slog.String("path", r.URL.Path),
					slog.Any("error", err),
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid ZITADEL-Signature"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
