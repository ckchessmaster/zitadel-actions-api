package model

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestActionRequest_UnmarshalOfficialSchema(t *testing.T) {
	jsonInput := `{
		"function": "preaccesstoken",
		"instanceID": "inst-1",
		"orgID": "org-1",
		"userID": "usr-1",
		"user": {
			"id": "usr-1",
			"username": "admin",
			"preferred_login_name": "admin@example.com"
		},
		"user_grants": [
			{
				"projectId": "proj-1",
				"roles": ["admin", "viewer"],
				"userGrantResourceOwner": "org-1",
				"userGrantResourceOwnerName": "Main Org"
			}
		]
	}`

	var req ActionRequest
	if err := json.Unmarshal([]byte(jsonInput), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if req.Function != "preaccesstoken" {
		t.Errorf("Function = %q, want preaccesstoken", req.Function)
	}
	if req.InstanceID != "inst-1" {
		t.Errorf("InstanceID = %q, want inst-1", req.InstanceID)
	}
	if req.OrgID != "org-1" {
		t.Errorf("OrgID = %q, want org-1", req.OrgID)
	}
	if req.UserID != "usr-1" {
		t.Errorf("UserID = %q, want usr-1", req.UserID)
	}
	if req.User == nil || req.User.Username != "admin" {
		t.Errorf("User = %+v, want username admin", req.User)
	}
	if len(req.UserGrants) != 1 {
		t.Fatalf("len(UserGrants) = %d, want 1", len(req.UserGrants))
	}
	expectedRoles := []string{"admin", "viewer"}
	if !reflect.DeepEqual(req.UserGrants[0].Roles, expectedRoles) {
		t.Errorf("Roles = %v, want %v", req.UserGrants[0].Roles, expectedRoles)
	}
	if req.UserGrants[0].ProjectID != "proj-1" {
		t.Errorf("ProjectID = %q, want proj-1", req.UserGrants[0].ProjectID)
	}
	if req.UserGrants[0].UserGrantResourceOwnerName != "Main Org" {
		t.Errorf("UserGrantResourceOwnerName = %q, want Main Org", req.UserGrants[0].UserGrantResourceOwnerName)
	}
}
