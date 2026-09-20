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

func TestFlattenRolesHandler_ValidPayload(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		ClaimName:      "groups",
		RoleFormat:     "bare",
		LowercaseRoles: true,
	}
	flattener := service.NewRoleFlattener()
	h := NewFlattenRolesHandler(flattener, cfg, logger)

	payload := `{
		"user_grants": [
			{"projectId": "frigate", "roles": ["admin", "viewer"]}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/v1/actions/flatten-roles", bytes.NewBufferString(payload))
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

func TestFlattenRolesHandler_QueryParamsOverride(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		ClaimName:      "groups",
		RoleFormat:     "bare",
		LowercaseRoles: true,
	}
	flattener := service.NewRoleFlattener()
	h := NewFlattenRolesHandler(flattener, cfg, logger)

	payload := `{
		"user_grants": [
			{"projectId": "frigate", "roles": ["ADMIN"]}
		]
	}`

	url := "/v1/actions/flatten-roles?claim_name=roles&format=prefixed"
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewBufferString(payload))
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

func TestFlattenRolesHandler_EmptyBody(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		ClaimName:      "groups",
		RoleFormat:     "bare",
		LowercaseRoles: true,
	}
	flattener := service.NewRoleFlattener()
	h := NewFlattenRolesHandler(flattener, cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/v1/actions/flatten-roles", bytes.NewBufferString(""))
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

func TestFlattenRolesHandler_InvalidJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		ClaimName: "groups",
	}
	flattener := service.NewRoleFlattener()
	h := NewFlattenRolesHandler(flattener, cfg, logger)

	req := httptest.NewRequest(http.MethodPost, "/v1/actions/flatten-roles", bytes.NewBufferString("{invalid json"))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestFlattenRolesHandler_MethodNotAllowed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{
		ClaimName: "groups",
	}
	flattener := service.NewRoleFlattener()
	h := NewFlattenRolesHandler(flattener, cfg, logger)

	req := httptest.NewRequest(http.MethodGet, "/v1/actions/flatten-roles", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
