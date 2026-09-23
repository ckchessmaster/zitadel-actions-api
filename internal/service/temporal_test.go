package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/ckchessmaster/zitadel-actions-api/internal/model"
)

const testTemporalProjectID = "temporal-proj-123"

func TestTemporalPermissionsProcessor_Name(t *testing.T) {
	proc := NewTemporalPermissionsProcessor(testTemporalProjectID)
	if got := proc.Name(); got != "temporal_permissions" {
		t.Errorf("Name() = %q, want %q", got, "temporal_permissions")
	}
}

func TestTemporalPermissionsProcessor_ShouldProcess(t *testing.T) {
	proc := NewTemporalPermissionsProcessor(testTemporalProjectID)

	tests := []struct {
		name     string
		req      *model.ActionRequest
		expected bool
	}{
		{
			name:     "nil request",
			req:      nil,
			expected: false,
		},
		{
			name: "audience scope matches",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				Scopes: []string{
					"openid",
					"urn:zitadel:iam:org:project:id:temporal-proj-123:aud",
					"profile",
				},
			},
			expected: true,
		},
		{
			name: "direct project grant matches",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				UserGrants: []model.UserGrant{
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"temporal-admin"},
					},
				},
			},
			expected: true,
		},
		{
			name: "both audience scope and project grant match",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				Scopes: []string{
					"urn:zitadel:iam:org:project:id:temporal-proj-123:aud",
				},
				UserGrants: []model.UserGrant{
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"temporal-admin"},
					},
				},
			},
			expected: true,
		},
		{
			name: "different project ID",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				UserGrants: []model.UserGrant{
					{
						ProjectID: "other-project",
						Roles:     []string{"temporal-admin"},
					},
				},
			},
			expected: false,
		},
		{
			name: "different audience scope",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				Scopes: []string{
					"openid",
					"urn:zitadel:iam:org:project:id:other-project:aud",
				},
			},
			expected: false,
		},
		{
			name: "empty request no scopes no grants",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
			},
			expected: false,
		},
		{
			name: "scopes present but no temporal scope, grants for different project",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				Scopes:   []string{"openid", "profile"},
				UserGrants: []model.UserGrant{
					{
						ProjectID: "frigate-proj",
						Roles:     []string{"frigate-admin"},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := proc.ShouldProcess(tt.req)
			if got != tt.expected {
				t.Errorf("ShouldProcess() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTemporalPermissionsProcessor_Process(t *testing.T) {
	proc := NewTemporalPermissionsProcessor(testTemporalProjectID)
	ctx := context.Background()
	opts := model.FlattenOptions{} // opts are not used by this processor

	tests := []struct {
		name                string
		req                 *model.ActionRequest
		expectedPermissions []string
	}{
		{
			name: "temporal-admin role maps to system:admin",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				UserGrants: []model.UserGrant{
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"temporal-admin"},
					},
				},
			},
			expectedPermissions: []string{"system:admin"},
		},
		{
			name: "temporal-worker role maps to read, worker, write",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				UserGrants: []model.UserGrant{
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"temporal-worker"},
					},
				},
			},
			expectedPermissions: []string{"read", "worker", "write"},
		},
		{
			name: "both roles combined with deduplication",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				UserGrants: []model.UserGrant{
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"temporal-admin", "temporal-worker"},
					},
				},
			},
			expectedPermissions: []string{"read", "system:admin", "worker", "write"},
		},
		{
			name: "both roles across multiple grants with deduplication",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				UserGrants: []model.UserGrant{
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"temporal-admin"},
					},
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"temporal-worker"},
					},
				},
			},
			expectedPermissions: []string{"read", "system:admin", "worker", "write"},
		},
		{
			name: "duplicate roles across multiple grants produce no duplicate permissions",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				UserGrants: []model.UserGrant{
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"temporal-worker"},
					},
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"temporal-worker"},
					},
				},
			},
			expectedPermissions: []string{"read", "worker", "write"},
		},
		{
			name: "grants from different project are ignored",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				UserGrants: []model.UserGrant{
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"temporal-admin"},
					},
					{
						ProjectID: "other-project",
						Roles:     []string{"temporal-worker"},
					},
				},
			},
			expectedPermissions: []string{"system:admin"},
		},
		{
			name: "unrecognized roles are skipped",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				UserGrants: []model.UserGrant{
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"temporal-admin", "unknown-role", "viewer"},
					},
				},
			},
			expectedPermissions: []string{"system:admin"},
		},
		{
			name: "no recognized roles produces empty permissions",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				UserGrants: []model.UserGrant{
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"viewer", "editor"},
					},
				},
			},
			expectedPermissions: []string{},
		},
		{
			name: "empty roles list produces empty permissions",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				UserGrants: []model.UserGrant{
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{},
					},
				},
			},
			expectedPermissions: []string{},
		},
		{
			name: "whitespace-only roles are skipped",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
				UserGrants: []model.UserGrant{
					{
						ProjectID: testTemporalProjectID,
						Roles:     []string{"  ", "", "temporal-admin"},
					},
				},
			},
			expectedPermissions: []string{"system:admin"},
		},
		{
			name: "nil request returns nil response",
			req:  nil,
			// nil response expected, handled separately below
			expectedPermissions: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := proc.Process(ctx, tt.req, opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.req == nil {
				if res != nil {
					t.Errorf("expected nil response for nil request, got %+v", res)
				}
				return
			}

			if res == nil {
				t.Fatal("expected non-nil response, got nil")
			}

			// Verify append_claims structure
			if len(res.AppendClaims) != 1 {
				t.Fatalf("len(AppendClaims) = %d, want 1", len(res.AppendClaims))
			}

			claim := res.AppendClaims[0]
			if claim.Key != "temporal-permissions" {
				t.Errorf("claim.Key = %q, want %q", claim.Key, "temporal-permissions")
			}

			claimValue, ok := claim.Value.([]string)
			if !ok {
				t.Fatalf("claim.Value type = %T, want []string", claim.Value)
			}
			if !reflect.DeepEqual(claimValue, tt.expectedPermissions) {
				t.Errorf("claim.Value = %v, want %v", claimValue, tt.expectedPermissions)
			}

			// Verify claims map
			claimsMapValue, ok := res.Claims["temporal-permissions"]
			if !ok {
				t.Fatal("res.Claims missing temporal-permissions key")
			}
			if !reflect.DeepEqual(claimsMapValue, tt.expectedPermissions) {
				t.Errorf("res.Claims[temporal-permissions] = %v, want %v", claimsMapValue, tt.expectedPermissions)
			}
		})
	}
}

func TestTemporalPermissionsProcessor_ProcessDeterministicOutput(t *testing.T) {
	proc := NewTemporalPermissionsProcessor(testTemporalProjectID)
	ctx := context.Background()
	opts := model.FlattenOptions{}

	req := &model.ActionRequest{
		Function: "preaccesstoken",
		UserGrants: []model.UserGrant{
			{
				ProjectID: testTemporalProjectID,
				Roles:     []string{"temporal-worker", "temporal-admin"},
			},
		},
	}

	// Run multiple times to verify deterministic (sorted) output
	expected := []string{"read", "system:admin", "worker", "write"}
	for i := 0; i < 10; i++ {
		res, err := proc.Process(ctx, req, opts)
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		claimValue := res.AppendClaims[0].Value.([]string)
		if !reflect.DeepEqual(claimValue, expected) {
			t.Fatalf("iteration %d: got %v, want %v", i, claimValue, expected)
		}
	}
}

func TestTemporalPermissionsProcessor_Integration_WithDispatcher(t *testing.T) {
	// Verify that the Temporal processor integrates correctly with the dispatcher
	// alongside the RoleFlattener, each producing their own claims.
	flattener := NewRoleFlattener()
	temporal := NewTemporalPermissionsProcessor(testTemporalProjectID)
	dispatcher := NewDispatcher(flattener, temporal)

	req := &model.ActionRequest{
		Function: "preaccesstoken",
		UserGrants: []model.UserGrant{
			{
				ProjectID: testTemporalProjectID,
				Roles:     []string{"temporal-admin"},
			},
			{
				ProjectID: "frigate-proj",
				Roles:     []string{"frigate-admin"},
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

	// Verify the flattener produced groups
	expectedGroups := []string{"frigate-admin", "temporal-admin"}
	if !reflect.DeepEqual(res.Groups, expectedGroups) {
		t.Errorf("res.Groups = %v, want %v", res.Groups, expectedGroups)
	}

	// Verify the temporal processor produced temporal-permissions claim
	temporalPerms, ok := res.Claims["temporal-permissions"]
	if !ok {
		t.Fatal("res.Claims missing temporal-permissions key")
	}
	expectedPerms := []string{"system:admin"}
	if !reflect.DeepEqual(temporalPerms, expectedPerms) {
		t.Errorf("res.Claims[temporal-permissions] = %v, want %v", temporalPerms, expectedPerms)
	}

	// Verify both claims appear in append_claims (groups + temporal-permissions)
	if len(res.AppendClaims) != 2 {
		t.Fatalf("len(AppendClaims) = %d, want 2", len(res.AppendClaims))
	}
}

func TestTemporalPermissionsProcessor_Integration_NoOpForOtherProject(t *testing.T) {
	// Verify that when the request targets a different project,
	// the temporal processor is skipped entirely by the dispatcher.
	flattener := NewRoleFlattener()
	temporal := NewTemporalPermissionsProcessor(testTemporalProjectID)
	dispatcher := NewDispatcher(flattener, temporal)

	req := &model.ActionRequest{
		Function: "preaccesstoken",
		UserGrants: []model.UserGrant{
			{
				ProjectID: "frigate-proj",
				Roles:     []string{"frigate-admin"},
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

	// Groups should only have frigate roles
	expectedGroups := []string{"frigate-admin"}
	if !reflect.DeepEqual(res.Groups, expectedGroups) {
		t.Errorf("res.Groups = %v, want %v", res.Groups, expectedGroups)
	}

	// temporal-permissions should not be present
	if _, ok := res.Claims["temporal-permissions"]; ok {
		t.Error("res.Claims[temporal-permissions] should not be present for non-Temporal project")
	}

	// Only 1 claim (groups) should be present
	if len(res.AppendClaims) != 1 {
		t.Fatalf("len(AppendClaims) = %d, want 1", len(res.AppendClaims))
	}
}

func TestTemporalPermissionsProcessor_ScopeBasedDetection_WithGrants(t *testing.T) {
	// When the Temporal project is detected via audience scope, the processor
	// should still correctly extract roles from matching user grants.
	proc := NewTemporalPermissionsProcessor(testTemporalProjectID)
	ctx := context.Background()
	opts := model.FlattenOptions{}

	req := &model.ActionRequest{
		Function: "preaccesstoken",
		Scopes: []string{
			"openid",
			"urn:zitadel:iam:org:project:id:temporal-proj-123:aud",
		},
		UserGrants: []model.UserGrant{
			{
				ProjectID: testTemporalProjectID,
				Roles:     []string{"temporal-worker"},
			},
		},
	}

	if !proc.ShouldProcess(req) {
		t.Fatal("ShouldProcess() = false, want true (audience scope match)")
	}

	res, err := proc.Process(ctx, req, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"read", "worker", "write"}
	claimValue := res.AppendClaims[0].Value.([]string)
	if !reflect.DeepEqual(claimValue, expected) {
		t.Errorf("claim.Value = %v, want %v", claimValue, expected)
	}
}

func TestTemporalPermissionsProcessor_ScopeMatchButNoTemporalGrants(t *testing.T) {
	// When the audience scope matches but no user grants are for the Temporal project,
	// the processor should return empty permissions.
	proc := NewTemporalPermissionsProcessor(testTemporalProjectID)
	ctx := context.Background()
	opts := model.FlattenOptions{}

	req := &model.ActionRequest{
		Function: "preaccesstoken",
		Scopes: []string{
			"urn:zitadel:iam:org:project:id:temporal-proj-123:aud",
		},
		UserGrants: []model.UserGrant{
			{
				ProjectID: "other-project",
				Roles:     []string{"temporal-admin"},
			},
		},
	}

	if !proc.ShouldProcess(req) {
		t.Fatal("ShouldProcess() = false, want true (audience scope match)")
	}

	res, err := proc.Process(ctx, req, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{}
	claimValue := res.AppendClaims[0].Value.([]string)
	if !reflect.DeepEqual(claimValue, expected) {
		t.Errorf("claim.Value = %v, want %v", claimValue, expected)
	}
}
