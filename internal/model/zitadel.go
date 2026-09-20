package model

// ActionRequest represents the incoming webhook payload sent by ZITADEL.
// It is designed to handle both ZITADEL Actions V2 target payloads
// and Actions V1 / custom script payloads.
type ActionRequest struct {
	InstanceID string         `json:"instanceID,omitempty"`
	OrgID      string         `json:"orgID,omitempty"`
	UserID     string         `json:"userID,omitempty"`
	FullMethod string         `json:"fullMethod,omitempty"`
	UserGrants []UserGrant    `json:"user_grants,omitempty"`
	Grants     []UserGrant    `json:"grants,omitempty"`
	Claims     map[string]any `json:"claims,omitempty"`
	Request    map[string]any `json:"request,omitempty"`
	Response   map[string]any `json:"response,omitempty"`
}

// UserGrant represents a grant of roles for a specific project.
type UserGrant struct {
	ProjectID              string   `json:"projectId"`
	Roles                  []string `json:"roles"`
	UserGrantResourceOwner string   `json:"userGrantResourceOwner,omitempty"`
}

// ClaimItem represents a single key-value claim entry for ZITADEL's append_claims response.
type ClaimItem struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// ActionResponse represents the response sent back to ZITADEL or the caller.
// It includes:
// - AppendClaims: The standard ZITADEL Actions V2 field used to inject custom claims.
// - Groups: Convenience flat array for direct consumers.
// - Claims: Key-value map of all appended claims.
type ActionResponse struct {
	AppendClaims []ClaimItem    `json:"append_claims"`
	Groups       []string       `json:"groups"`
	Claims       map[string]any `json:"claims"`
}

// FlattenOptions defines options for customizing role extraction and formatting.
type FlattenOptions struct {
	// ClaimName is the name of the claim to populate (e.g. "groups").
	ClaimName string

	// RoleFormat specifies the formatting style: "bare" (default) or "prefixed".
	// "bare": "admin"
	// "prefixed": "projectId:admin"
	RoleFormat string

	// Lowercase determines whether role names are converted to lowercase.
	Lowercase bool

	// ProjectIDFilter, if set, restricts roles to only those from this project ID.
	ProjectIDFilter string
}
