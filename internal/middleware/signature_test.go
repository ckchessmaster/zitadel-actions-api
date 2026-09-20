package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zitadel/zitadel-go/v3/pkg/actions"
)

func TestSignatureValidator_BypassWhenNoKey(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := SignatureValidator("", logger)

	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/actions/flatten-roles", bytes.NewBufferString(`{"test":"data"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if !called {
		t.Errorf("downstream handler was not called")
	}
}

func TestSignatureValidator_MissingSignature(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mw := SignatureValidator("secret-key", logger)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/actions/flatten-roles", bytes.NewBufferString(`{"test":"data"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 Unauthorized", rec.Code)
	}
}

func TestSignatureValidator_ValidSignature(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	signingKey := "my-target-secret-key"
	mw := SignatureValidator(signingKey, logger)

	payload := []byte(`{"user_grants":[{"projectId":"p1","roles":["admin"]}]}`)

	sig := actions.ComputeSignatureHeader(time.Now(), payload, signingKey)

	called := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// Verify body is still readable
		b, _ := io.ReadAll(r.Body)
		if string(b) != string(payload) {
			t.Errorf("body modified: got %s, want %s", string(b), string(payload))
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/actions/flatten-roles", bytes.NewBuffer(payload))
	req.Header.Set("Zitadel-Signature", sig)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if !called {
		t.Errorf("downstream handler was not called")
	}
}

func TestSignatureValidator_InvalidSignature(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	signingKey := "my-target-secret-key"
	mw := SignatureValidator(signingKey, logger)

	payload := []byte(`{"user_grants":[{"projectId":"p1","roles":["admin"]}]}`)
	sig := actions.ComputeSignatureHeader(time.Now(), payload, "wrong-key")

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/actions/flatten-roles", bytes.NewBuffer(payload))
	req.Header.Set("Zitadel-Signature", sig)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 Unauthorized", rec.Code)
	}
}
