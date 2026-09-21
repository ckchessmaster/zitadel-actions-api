package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/ckchessmaster/zitadel-actions-api/internal/model"
)

func TestClient_FetchUserGrants_Success(t *testing.T) {
	expectedUserID := "user-123"
	expectedToken := "test-pat-token"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/management/v1/users/grants/_search" {
			t.Errorf("Path = %q, want /management/v1/users/grants/_search", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer "+expectedToken {
			t.Errorf("Authorization = %q, want Bearer %s", authHeader, expectedToken)
		}

		var reqBody userGrantsSearchRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if len(reqBody.Queries) != 1 || reqBody.Queries[0].UserIDQuery == nil || reqBody.Queries[0].UserIDQuery.UserID != expectedUserID {
			t.Errorf("unexpected query: %+v", reqBody.Queries)
		}

		resp := map[string]any{
			"result": []map[string]any{
				{
					"projectId": "proj-frigate",
					"roles":     []string{"frigate-admin", "viewer"},
				},
				{
					"projectId": "proj-ha",
					"roles":     []string{"user"},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	c := New(ts.URL, expectedToken, ts.Client())
	grants, err := c.FetchUserGrants(context.Background(), expectedUserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedGrants := []model.UserGrant{
		{
			ProjectID: "proj-frigate",
			Roles:     []string{"frigate-admin", "viewer"},
		},
		{
			ProjectID: "proj-ha",
			Roles:     []string{"user"},
		},
	}

	if !reflect.DeepEqual(grants, expectedGrants) {
		t.Errorf("grants = %+v, want %+v", grants, expectedGrants)
	}
}

func TestClient_FetchUserGrants_EmptyUserID(t *testing.T) {
	c := New("https://example.com", "token", nil)
	grants, err := c.FetchUserGrants(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty userID, got nil")
	}
	if grants != nil {
		t.Errorf("grants = %v, want nil", grants)
	}
}

func TestClient_FetchUserGrants_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer ts.Close()

	c := New(ts.URL, "bad-token", ts.Client())
	grants, err := c.FetchUserGrants(context.Background(), "user-123")
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	if grants != nil {
		t.Errorf("grants = %v, want nil", grants)
	}
}

func TestClient_FetchUserGrants_InvalidJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer ts.Close()

	c := New(ts.URL, "token", ts.Client())
	grants, err := c.FetchUserGrants(context.Background(), "user-123")
	if err == nil {
		t.Fatal("expected error for invalid json, got nil")
	}
	if grants != nil {
		t.Errorf("grants = %v, want nil", grants)
	}
}

func TestClient_FetchUserGrants_EmptyResults(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": []}`))
	}))
	defer ts.Close()

	c := New(ts.URL, "token", ts.Client())
	grants, err := c.FetchUserGrants(context.Background(), "user-no-roles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(grants) != 0 {
		t.Errorf("len(grants) = %d, want 0", len(grants))
	}
}
