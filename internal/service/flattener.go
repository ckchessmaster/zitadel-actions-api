package service

import (
	"sort"
	"strings"

	"github.com/ckchessmaster/zitadel-actions-api/internal/model"
)

// Flattener defines the interface for role flattening business logic.
type Flattener interface {
	FlattenRoles(req *model.ActionRequest, opts model.FlattenOptions) *model.ActionResponse
}

// RoleFlattener implements Flattener.
type RoleFlattener struct{}

// NewRoleFlattener creates a new RoleFlattener service.
func NewRoleFlattener() *RoleFlattener {
	return &RoleFlattener{}
}

// FlattenRoles extracts roles from the request, filters, normalizes, deduplicates,
// and returns an ActionResponse formatted for both ZITADEL Actions V2 and direct consumers.
func (f *RoleFlattener) FlattenRoles(req *model.ActionRequest, opts model.FlattenOptions) *model.ActionResponse {
	claimName := opts.ClaimName
	if claimName == "" {
		claimName = "groups"
	}

	rawRoles := make(map[string]struct{})

	if req != nil {
		// 1. Process user_grants (Actions V2 primary field)
		for _, grant := range req.UserGrants {
			f.processGrant(grant, opts, rawRoles)
		}

		// 2. Process grants (Actions V1 / custom script field)
		for _, grant := range req.Grants {
			f.processGrant(grant, opts, rawRoles)
		}

		// 3. Process claims map if roles are embedded directly in claims
		if req.Claims != nil {
			f.processClaims(req.Claims, opts, rawRoles)
		}
	}

	// Convert map to sorted slice for deterministic output
	groups := make([]string, 0, len(rawRoles))
	for role := range rawRoles {
		groups = append(groups, role)
	}
	sort.Strings(groups)

	return &model.ActionResponse{
		AppendClaims: []model.ClaimItem{
			{
				Key:   claimName,
				Value: groups,
			},
		},
		Groups: groups,
		Claims: map[string]any{
			claimName: groups,
		},
	}
}

func (f *RoleFlattener) processGrant(grant model.UserGrant, opts model.FlattenOptions, result map[string]struct{}) {
	// If project filter is set, skip grants for other projects
	if opts.ProjectIDFilter != "" && grant.ProjectID != "" && grant.ProjectID != opts.ProjectIDFilter {
		return
	}

	for _, role := range grant.Roles {
		formatted := f.formatRole(role, grant.ProjectID, opts)
		if formatted != "" {
			result[formatted] = struct{}{}
		}
	}
}

func (f *RoleFlattener) processClaims(claims map[string]any, opts model.FlattenOptions, result map[string]struct{}) {
	for k, v := range claims {
		// Check for Zitadel role claims:
		// - "urn:zitadel:iam:org:project:roles"
		// - "urn:zitadel:iam:org:project:<project_id>:roles"
		// - "roles"
		var projectID string
		isZitadelRoleClaim := false

		if k == "roles" {
			isZitadelRoleClaim = true
		} else if strings.HasPrefix(k, "urn:zitadel:iam:org:project:") && strings.HasSuffix(k, ":roles") {
			isZitadelRoleClaim = true
			parts := strings.Split(k, ":")
			// urn:zitadel:iam:org:project:<project_id>:roles has 7 parts
			if len(parts) == 7 {
				projectID = parts[5]
			}
		} else if k == "urn:zitadel:iam:org:project:roles" {
			isZitadelRoleClaim = true
		}

		if !isZitadelRoleClaim {
			continue
		}

		if opts.ProjectIDFilter != "" && projectID != "" && projectID != opts.ProjectIDFilter {
			continue
		}

		// ZITADEL roles claim can be a map where keys are role names:
		// { "admin": { "org123": "My Org" }, "viewer": { ... } }
		// or a slice of strings: ["admin", "viewer"]
		switch typedVal := v.(type) {
		case map[string]any:
			for roleName := range typedVal {
				formatted := f.formatRole(roleName, projectID, opts)
				if formatted != "" {
					result[formatted] = struct{}{}
				}
			}
		case []any:
			for _, item := range typedVal {
				if strRole, ok := item.(string); ok {
					formatted := f.formatRole(strRole, projectID, opts)
					if formatted != "" {
						result[formatted] = struct{}{}
					}
				}
			}
		case []string:
			for _, strRole := range typedVal {
				formatted := f.formatRole(strRole, projectID, opts)
				if formatted != "" {
					result[formatted] = struct{}{}
				}
			}
		}
	}
}

func (f *RoleFlattener) formatRole(role, projectID string, opts model.FlattenOptions) string {
	cleaned := strings.TrimSpace(role)
	if cleaned == "" {
		return ""
	}

	if opts.Lowercase {
		cleaned = strings.ToLower(cleaned)
	}

	if opts.RoleFormat == "prefixed" && projectID != "" {
		proj := strings.TrimSpace(projectID)
		if opts.Lowercase {
			proj = strings.ToLower(proj)
		}
		return proj + ":" + cleaned
	}

	return cleaned
}
