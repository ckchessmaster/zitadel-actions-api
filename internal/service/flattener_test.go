package service

import (
	"reflect"
	"testing"

	"github.com/ckchessmaster/zitadel-actions-api/internal/model"
)

func TestRoleFlattener_FlattenRoles(t *testing.T) {
	flattener := NewRoleFlattener()

	tests := []struct {
		name     string
		req      *model.ActionRequest
		opts     model.FlattenOptions
		expected []string
	}{
		{
			name: "Actions V2 user_grants bare format",
			req: &model.ActionRequest{
				UserGrants: []model.UserGrant{
					{
						ProjectID: "proj-1",
						Roles:     []string{"admin", "viewer"},
					},
					{
						ProjectID: "proj-2",
						Roles:     []string{"editor", "viewer"}, // viewer is duplicated
					},
				},
			},
			opts: model.FlattenOptions{
				ClaimName:  "groups",
				RoleFormat: "bare",
				Lowercase:  true,
			},
			expected: []string{"admin", "editor", "viewer"},
		},
		{
			name: "Case insensitive normalization and whitespace trimming",
			req: &model.ActionRequest{
				UserGrants: []model.UserGrant{
					{
						ProjectID: "proj-1",
						Roles:     []string{" Admin ", "VIEWER", "viewer"},
					},
				},
			},
			opts: model.FlattenOptions{
				ClaimName:  "groups",
				RoleFormat: "bare",
				Lowercase:  true,
			},
			expected: []string{"admin", "viewer"},
		},
		{
			name: "Prefixed format with project ID",
			req: &model.ActionRequest{
				UserGrants: []model.UserGrant{
					{
						ProjectID: "frigate",
						Roles:     []string{"admin", "user"},
					},
				},
			},
			opts: model.FlattenOptions{
				ClaimName:  "groups",
				RoleFormat: "prefixed",
				Lowercase:  true,
			},
			expected: []string{"frigate:admin", "frigate:user"},
		},
		{
			name: "Project ID filtering",
			req: &model.ActionRequest{
				UserGrants: []model.UserGrant{
					{
						ProjectID: "frigate-proj",
						Roles:     []string{"frigate-admin"},
					},
					{
						ProjectID: "home-assistant",
						Roles:     []string{"hass-admin"},
					},
				},
			},
			opts: model.FlattenOptions{
				ClaimName:       "groups",
				RoleFormat:      "bare",
				Lowercase:       true,
				ProjectIDFilter: "frigate-proj",
			},
			expected: []string{"frigate-admin"},
		},
		{
			name: "Empty user_grants returns empty groups",
			req: &model.ActionRequest{
				Function:   "preaccesstoken",
				UserGrants: []model.UserGrant{},
			},
			opts: model.FlattenOptions{
				ClaimName: "groups",
			},
			expected: []string{},
		},
		{
			name: "Nil request returns empty groups",
			req:  nil,
			opts: model.FlattenOptions{
				ClaimName: "groups",
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := flattener.FlattenRoles(tt.req, tt.opts)

			if !reflect.DeepEqual(res.Groups, tt.expected) {
				t.Errorf("Groups = %v, want %v", res.Groups, tt.expected)
			}

			// Verify append_claims structure
			if len(res.AppendClaims) != 1 {
				t.Fatalf("len(AppendClaims) = %d, want 1", len(res.AppendClaims))
			}

			claim := res.AppendClaims[0]
			expectedClaimName := tt.opts.ClaimName
			if expectedClaimName == "" {
				expectedClaimName = "groups"
			}
			if claim.Key != expectedClaimName {
				t.Errorf("claim.Key = %q, want %q", claim.Key, expectedClaimName)
			}
			if !reflect.DeepEqual(claim.Value, tt.expected) {
				t.Errorf("claim.Value = %v, want %v", claim.Value, tt.expected)
			}

			// Verify claims map
			if !reflect.DeepEqual(res.Claims[expectedClaimName], tt.expected) {
				t.Errorf("res.Claims[%s] = %v, want %v", expectedClaimName, res.Claims[expectedClaimName], tt.expected)
			}
		})
	}
}
