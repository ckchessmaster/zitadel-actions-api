package service

import (
	"context"
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

type mockGrantFetcher struct {
	calledUserID string
	calledOrgID  string
	grants       []model.UserGrant
	err          error
}

func (m *mockGrantFetcher) FetchUserGrants(ctx context.Context, userID string, orgIDs ...string) ([]model.UserGrant, error) {
	m.calledUserID = userID
	if len(orgIDs) > 0 {
		m.calledOrgID = orgIDs[0]
	}
	return m.grants, m.err
}

func TestRoleFlattener_ShouldProcess(t *testing.T) {
	flattener := NewRoleFlattener()

	tests := []struct {
		name     string
		req      *model.ActionRequest
		expected bool
	}{
		{
			name:     "nil request",
			req:      nil,
			expected: true,
		},
		{
			name: "function/preuserinfo prefix",
			req: &model.ActionRequest{
				Function: "function/preuserinfo",
			},
			expected: true,
		},
		{
			name: "function/preaccesstoken prefix",
			req: &model.ActionRequest{
				Function: "function/preaccesstoken",
			},
			expected: true,
		},
		{
			name: "preuserinfo without prefix",
			req: &model.ActionRequest{
				Function: "preuserinfo",
			},
			expected: true,
		},
		{
			name: "preaccesstoken without prefix",
			req: &model.ActionRequest{
				Function: "preaccesstoken",
			},
			expected: true,
		},
		{
			name: "empty function",
			req: &model.ActionRequest{
				Function: "",
			},
			expected: true,
		},
		{
			name: "with user grants",
			req: &model.ActionRequest{
				Function: "some_other_hook",
				UserGrants: []model.UserGrant{
					{ProjectID: "p1", Roles: []string{"admin"}},
				},
			},
			expected: true,
		},
		{
			name: "with user object",
			req: &model.ActionRequest{
				Function: "some_other_hook",
				User:     &model.UserInfo{ID: "123"},
			},
			expected: true,
		},
		{
			name: "unrelated function with no user or grants",
			req: &model.ActionRequest{
				Function: "some_other_hook",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := flattener.ShouldProcess(tt.req)
			if got != tt.expected {
				t.Errorf("ShouldProcess() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRoleFlattener_FlattenRoles_WithFetcher(t *testing.T) {
	mockFetcher := &mockGrantFetcher{
		grants: []model.UserGrant{
			{
				ProjectID: "frigate",
				Roles:     []string{"frigate-admin", "viewer"},
			},
		},
	}

	flattener := NewRoleFlattener(mockFetcher)

	// Case 1: Empty UserGrants, user ID in req.User.ID (like preuserinfo) with Org
	req := &model.ActionRequest{
		Function:   "function/preuserinfo",
		UserGrants: []model.UserGrant{},
		User: &model.UserInfo{
			ID:            "user-391212",
			ResourceOwner: "org-391212",
		},
		Org: &model.OrgInfo{
			ID: "org-391212",
		},
	}

	opts := model.FlattenOptions{
		ClaimName:  "groups",
		RoleFormat: "bare",
		Lowercase:  true,
	}

	res := flattener.FlattenRoles(req, opts)

	if mockFetcher.calledUserID != "user-391212" {
		t.Errorf("calledUserID = %q, want user-391212", mockFetcher.calledUserID)
	}
	if mockFetcher.calledOrgID != "org-391212" {
		t.Errorf("calledOrgID = %q, want org-391212", mockFetcher.calledOrgID)
	}

	expectedGroups := []string{"frigate-admin", "viewer"}
	if !reflect.DeepEqual(res.Groups, expectedGroups) {
		t.Errorf("res.Groups = %v, want %v", res.Groups, expectedGroups)
	}

	// Case 2: UserGrants already populated - fetcher should NOT be called
	mockFetcher.calledUserID = ""
	mockFetcher.calledOrgID = ""
	reqWithGrants := &model.ActionRequest{
		UserGrants: []model.UserGrant{
			{
				ProjectID: "proj-1",
				Roles:     []string{"editor"},
			},
		},
		User: &model.UserInfo{
			ID: "user-391212",
		},
	}

	res2 := flattener.FlattenRoles(reqWithGrants, opts)
	if mockFetcher.calledUserID != "" {
		t.Errorf("fetcher should not be called when UserGrants is already populated")
	}
	if !reflect.DeepEqual(res2.Groups, []string{"editor"}) {
		t.Errorf("res2.Groups = %v, want [editor]", res2.Groups)
	}

	// Case 3: Empty UserGrants, user ID in req.UserInfo.Sub
	mockFetcher.calledUserID = ""
	mockFetcher.calledOrgID = ""
	reqWithSub := &model.ActionRequest{
		Function:   "function/preuserinfo",
		UserGrants: []model.UserGrant{},
		UserInfo: &model.UserClaims{
			Sub: "sub-999",
		},
	}

	res3 := flattener.FlattenRoles(reqWithSub, opts)
	if mockFetcher.calledUserID != "sub-999" {
		t.Errorf("calledUserID = %q, want sub-999", mockFetcher.calledUserID)
	}
	if !reflect.DeepEqual(res3.Groups, expectedGroups) {
		t.Errorf("res3.Groups = %v, want %v", res3.Groups, expectedGroups)
	}
}
