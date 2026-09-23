package model

// ActionRequest represents the incoming webhook payload sent by ZITADEL Actions V2.
type ActionRequest struct {
	Function   string      `json:"function,omitempty"`
	InstanceID string      `json:"instanceID,omitempty"`
	OrgID      string      `json:"orgID,omitempty"`
	UserID     string      `json:"userID,omitempty"`
	UserGrants []UserGrant `json:"user_grants,omitempty"`
	User       *UserInfo   `json:"user,omitempty"`
	UserInfo   *UserClaims `json:"userinfo,omitempty"`
	Org        *OrgInfo    `json:"org,omitempty"`
	Scopes     []string    `json:"scopes,omitempty"`
}

// OrgInfo represents the organization context provided in ZITADEL's payload.
type OrgInfo struct {
	ID            string `json:"id,omitempty"`
	Name          string `json:"name,omitempty"`
	PrimaryDomain string `json:"primary_domain,omitempty"`
}

// UserClaims represents the userinfo claims provided in ZITADEL's payload (e.g. sub).
type UserClaims struct {
	Sub string `json:"sub,omitempty"`
}

// UserInfo represents the user details provided in ZITADEL's payload.
type UserInfo struct {
	ID                 string `json:"id,omitempty"`
	ResourceOwner      string `json:"resource_owner,omitempty"`
	Username           string `json:"username,omitempty"`
	PreferredLoginName string `json:"preferred_login_name,omitempty"`
}

// UserGrant represents a grant of roles for a specific project.
type UserGrant struct {
	ProjectID                  string   `json:"projectId"`
	Roles                      []string `json:"roles"`
	UserGrantResourceOwner     string   `json:"userGrantResourceOwner,omitempty"`
	UserGrantResourceOwnerName string   `json:"userGrantResourceOwnerName,omitempty"`
}

// ClaimItem represents a single key-value claim entry for ZITADEL's append_claims response.
type ClaimItem struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// ActionResponse represents the response sent back to ZITADEL or the caller.
// It includes:
// - AppendClaims: The standard ZITADEL Actions V2 field used to inject custom claims.
// - AppendLogClaims: Optional log messages to append to ZITADEL execution logs.
// - Groups: Convenience flat array for direct consumers.
// - Claims: Key-value map of all appended claims.
type ActionResponse struct {
	AppendClaims    []ClaimItem    `json:"append_claims"`
	AppendLogClaims []string       `json:"append_log_claims,omitempty"`
	Groups          []string       `json:"groups"`
	Claims          map[string]any `json:"claims,omitempty"`
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
