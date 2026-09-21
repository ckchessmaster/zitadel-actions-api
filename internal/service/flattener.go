package service

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"github.com/ckchessmaster/zitadel-actions-api/internal/client"
	"github.com/ckchessmaster/zitadel-actions-api/internal/model"
)

// Flattener defines the interface for role flattening business logic.
type Flattener interface {
	ActionProcessor
	FlattenRoles(req *model.ActionRequest, opts model.FlattenOptions) *model.ActionResponse
}

// RoleFlattener implements Flattener and ActionProcessor for ZITADEL Actions V2.
type RoleFlattener struct {
	fetcher client.GrantFetcher
}

// NewRoleFlattener creates a new RoleFlattener service with an optional GrantFetcher.
func NewRoleFlattener(fetchers ...client.GrantFetcher) *RoleFlattener {
	var f client.GrantFetcher
	if len(fetchers) > 0 {
		f = fetchers[0]
	}
	return &RoleFlattener{
		fetcher: f,
	}
}

// Name returns the identifier of this action processor.
func (f *RoleFlattener) Name() string {
	return "role_flattener"
}

// ShouldProcess determines if this request should have roles extracted and flattened.
func (f *RoleFlattener) ShouldProcess(req *model.ActionRequest) bool {
	if req == nil {
		return true
	}
	fn := strings.ToLower(strings.TrimSpace(req.Function))
	if strings.Contains(fn, "preaccesstoken") || strings.Contains(fn, "preuserinfo") {
		return true
	}
	if len(req.UserGrants) > 0 {
		return true
	}
	if req.UserID != "" || req.User != nil || req.UserInfo != nil {
		return true
	}
	// Default to true for complement token webhook calls
	return req.Function == ""
}

// Process implements ActionProcessor.
func (f *RoleFlattener) Process(ctx context.Context, req *model.ActionRequest, opts model.FlattenOptions) (*model.ActionResponse, error) {
	return f.FlattenRolesWithContext(ctx, req, opts), nil
}

// FlattenRoles extracts roles from the request, filters, normalizes, deduplicates,
// and returns an ActionResponse formatted for both ZITADEL Actions V2 and direct consumers.
func (f *RoleFlattener) FlattenRoles(req *model.ActionRequest, opts model.FlattenOptions) *model.ActionResponse {
	return f.FlattenRolesWithContext(context.Background(), req, opts)
}

// FlattenRolesWithContext extracts roles, querying the ZITADEL API if grants are missing in the request.
func (f *RoleFlattener) FlattenRolesWithContext(ctx context.Context, req *model.ActionRequest, opts model.FlattenOptions) *model.ActionResponse {
	claimName := opts.ClaimName
	if claimName == "" {
		claimName = "groups"
	}

	rawRoles := make(map[string]struct{})

	if req != nil {
		// If UserGrants is empty and a GrantFetcher is configured, query ZITADEL directly
		if len(req.UserGrants) == 0 && f.fetcher != nil {
			userID := req.UserID
			if userID == "" && req.User != nil {
				userID = req.User.ID
			}
			if userID == "" && req.UserInfo != nil {
				userID = req.UserInfo.Sub
			}

			orgID := req.OrgID
			if orgID == "" && req.Org != nil {
				orgID = req.Org.ID
			}
			if orgID == "" && req.User != nil {
				orgID = req.User.ResourceOwner
			}

			if userID != "" {
				slog.InfoContext(ctx, "fetching user grants from ZITADEL API",
					slog.String("user_id", userID),
					slog.String("org_id", orgID),
				)
				grants, err := f.fetcher.FetchUserGrants(ctx, userID, orgID)
				if err != nil {
					slog.ErrorContext(ctx, "failed to fetch user grants from ZITADEL API",
						slog.String("user_id", userID),
						slog.String("org_id", orgID),
						slog.Any("error", err),
					)
				} else {
					slog.InfoContext(ctx, "successfully fetched user grants from ZITADEL API",
						slog.String("user_id", userID),
						slog.Int("grants_count", len(grants)),
					)
					if len(grants) > 0 {
						req.UserGrants = grants
					}
				}
			}
		}

		for _, grant := range req.UserGrants {
			f.processGrant(grant, opts, rawRoles)
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
