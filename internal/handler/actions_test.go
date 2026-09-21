package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/ckchessmaster/zitadel-actions-api/internal/config"
	"github.com/ckchessmaster/zitadel-actions-api/internal/model"
	"github.com/ckchessmaster/zitadel-actions-api/internal/service"
)

func TestActionsHandler_PreAccessToken(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		ClaimName:      "groups",
		RoleFormat:     "bare",
		LowercaseRoles: true,
	}
	dispatcher := service.NewDispatcher(service.NewRoleFlattener())
	h := NewActionsHandler(dispatcher, cfg, logger)

	payload := `{
		"function": "preaccesstoken",
		"user_grants": [
			{"projectId": "frigate", "roles": ["ADMIN", "viewer"]}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/actions", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp model.ActionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expectedGroups := []string{"admin", "viewer"}
	if !reflect.DeepEqual(resp.Groups, expectedGroups) {
		t.Errorf("resp.Groups = %v, want %v", resp.Groups, expectedGroups)
	}

	if len(resp.AppendClaims) != 1 {
		t.Fatalf("len(resp.AppendClaims) = %d, want 1", len(resp.AppendClaims))
	}
	if resp.AppendClaims[0].Key != "groups" {
		t.Errorf("claim key = %q, want 'groups'", resp.AppendClaims[0].Key)
	}
}

func TestActionsHandler_QueryParamsOverride(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		ClaimName:      "groups",
		RoleFormat:     "bare",
		LowercaseRoles: true,
	}
	dispatcher := service.NewDispatcher(service.NewRoleFlattener())
	h := NewActionsHandler(dispatcher, cfg, logger)

	payload := `{
		"function": "preuserinfo",
		"user_grants": [
			{"projectId": "frigate", "roles": ["admin"]}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/actions?claim_name=roles&format=prefixed", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp model.ActionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expectedGroups := []string{"frigate:admin"}
	if !reflect.DeepEqual(resp.Groups, expectedGroups) {
		t.Errorf("resp.Groups = %v, want %v", resp.Groups, expectedGroups)
	}
	if resp.AppendClaims[0].Key != "roles" {
		t.Errorf("claim key = %q, want 'roles'", resp.AppendClaims[0].Key)
	}
}

func TestActionsHandler_EmptyBody(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		ClaimName: "groups",
	}
	dispatcher := service.NewDispatcher(service.NewRoleFlattener())
	h := NewActionsHandler(dispatcher, cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/actions", bytes.NewBufferString(""))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp model.ActionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Groups) != 0 {
		t.Errorf("resp.Groups = %v, want empty", resp.Groups)
	}
}

func TestActionsHandler_InvalidJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		ClaimName: "groups",
	}
	dispatcher := service.NewDispatcher(service.NewRoleFlattener())
	h := NewActionsHandler(dispatcher, cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/actions", bytes.NewBufferString("{invalid json"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestActionsHandler_MethodNotAllowed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		ClaimName: "groups",
	}
	dispatcher := service.NewDispatcher(service.NewRoleFlattener())
	h := NewActionsHandler(dispatcher, cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/actions", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestActionsHandler_MultipleGrants(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		ClaimName:      "groups",
		RoleFormat:     "bare",
		LowercaseRoles: true,
	}
	dispatcher := service.NewDispatcher(service.NewRoleFlattener())
	h := NewActionsHandler(dispatcher, cfg, logger)

	payload := `{
		"function": "preaccesstoken",
		"user_grants": [
			{"projectId": "frigate", "roles": ["ADMIN", "viewer"]},
			{"projectId": "home-assistant", "roles": ["editor"]}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/actions", bytes.NewBufferString(payload))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp model.ActionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	expectedGroups := []string{"admin", "editor", "viewer"}
	if !reflect.DeepEqual(resp.Groups, expectedGroups) {
		t.Errorf("resp.Groups = %v, want %v", resp.Groups, expectedGroups)
	}
}
