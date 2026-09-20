package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/ckchessmaster/zitadel-actions-api/internal/model"
)

// mockProcessor adds custom claims for testing
type mockProcessor struct {
	name          string
	shouldProcess bool
	claimKey      string
	claimValue    any
}

func (m *mockProcessor) Name() string {
	return m.name
}

func (m *mockProcessor) ShouldProcess(req *model.ActionRequest) bool {
	return m.shouldProcess
}

func (m *mockProcessor) Process(ctx context.Context, req *model.ActionRequest, opts model.FlattenOptions) (*model.ActionResponse, error) {
	return &model.ActionResponse{
		AppendClaims: []model.ClaimItem{
			{Key: m.claimKey, Value: m.claimValue},
		},
		Claims: map[string]any{
			m.claimKey: m.claimValue,
		},
	}, nil
}

func TestDispatcher_Dispatch(t *testing.T) {
	flattener := NewRoleFlattener()

	mockProc := &mockProcessor{
		name:          "custom_meta",
		shouldProcess: true,
		claimKey:      "tier",
		claimValue:    "premium",
	}

	mockSkipped := &mockProcessor{
		name:          "skipped_proc",
		shouldProcess: false,
		claimKey:      "skipped",
		claimValue:    "none",
	}

	dispatcher := NewDispatcher(flattener, mockProc, mockSkipped)

	req := &model.ActionRequest{
		Function: "preaccesstoken",
		UserGrants: []model.UserGrant{
			{
				ProjectID: "proj-1",
				Roles:     []string{"admin", "viewer"},
			},
		},
	}

	opts := model.FlattenOptions{
		ClaimName:  "groups",
		RoleFormat: "bare",
		Lowercase:  true,
	}

	res, err := dispatcher.Dispatch(context.Background(), req, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedGroups := []string{"admin", "viewer"}
	if !reflect.DeepEqual(res.Groups, expectedGroups) {
		t.Errorf("res.Groups = %v, want %v", res.Groups, expectedGroups)
	}

	if len(res.AppendClaims) != 2 {
		t.Fatalf("len(AppendClaims) = %d, want 2 (groups + tier)", len(res.AppendClaims))
	}

	// Verify both claims exist in merged claims map
	if !reflect.DeepEqual(res.Claims["groups"], expectedGroups) {
		t.Errorf("claims[groups] = %v, want %v", res.Claims["groups"], expectedGroups)
	}
	if res.Claims["tier"] != "premium" {
		t.Errorf("claims[tier] = %v, want 'premium'", res.Claims["tier"])
	}
	if _, ok := res.Claims["skipped"]; ok {
		t.Errorf("claims[skipped] should not be present")
	}
}
