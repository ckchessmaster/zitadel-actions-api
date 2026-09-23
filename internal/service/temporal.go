package service

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/ckchessmaster/zitadel-actions-api/internal/model"
)

// temporalClaimName is the claim key injected into tokens for Temporal permissions.
const temporalClaimName = "temporal-permissions"

// roleMapping defines the mapping from ZITADEL project roles to Temporal permission strings.
var roleMapping = map[string][]string{
	"temporal-admin":  {"system:admin"},
	"temporal-worker": {"worker", "read", "write"},
}

// TemporalPermissionsProcessor implements ActionProcessor and maps ZITADEL project roles
// for a configured Temporal project into Temporal-compatible permission strings.
type TemporalPermissionsProcessor struct {
	projectID string
}

// NewTemporalPermissionsProcessor creates a new processor for the given Temporal project ID.
// The projectID must not be empty; callers should guard against registering a processor
// with an empty ID.
func NewTemporalPermissionsProcessor(projectID string) *TemporalPermissionsProcessor {
	return &TemporalPermissionsProcessor{
		projectID: projectID,
	}
}

// Name returns the identifier of this action processor.
func (t *TemporalPermissionsProcessor) Name() string {
	return "temporal_permissions"
}

// ShouldProcess determines if this request targets the configured Temporal project.
// It checks two detection methods:
//  1. Audience Scope: Whether the request scopes contain
//     "urn:zitadel:iam:org:project:id:<projectID>:aud"
//  2. Direct Project Grant Match: Whether any user grant's project ID matches
//     the configured Temporal project ID.
//
// Returns false (early exit) if neither condition is met.
func (t *TemporalPermissionsProcessor) ShouldProcess(req *model.ActionRequest) bool {
	if req == nil {
		return false
	}

	// Check audience scope
	expectedScope := fmt.Sprintf("urn:zitadel:iam:org:project:id:%s:aud", t.projectID)
	for _, scope := range req.Scopes {
		if scope == expectedScope {
			return true
		}
	}

	// Check direct project grant match
	for _, grant := range req.UserGrants {
		if grant.ProjectID == t.projectID {
			return true
		}
	}

	return false
}

// Process extracts the user's roles for the Temporal project, maps them to Temporal
// permission strings, deduplicates and sorts the result, and returns an ActionResponse
// with the temporal-permissions claim.
func (t *TemporalPermissionsProcessor) Process(ctx context.Context, req *model.ActionRequest, _ model.FlattenOptions) (*model.ActionResponse, error) {
	if req == nil {
		return nil, nil
	}

	permissions := make(map[string]struct{})

	for _, grant := range req.UserGrants {
		if grant.ProjectID != t.projectID {
			continue
		}

		for _, role := range grant.Roles {
			cleaned := strings.TrimSpace(role)
			if cleaned == "" {
				continue
			}

			mapped, ok := roleMapping[cleaned]
			if !ok {
				slog.DebugContext(ctx, "skipping unmapped Temporal role",
					slog.String("role", cleaned),
					slog.String("project_id", t.projectID),
				)
				continue
			}

			for _, perm := range mapped {
				permissions[perm] = struct{}{}
			}
		}
	}

	// Convert map to sorted slice for deterministic output
	result := make([]string, 0, len(permissions))
	for perm := range permissions {
		result = append(result, perm)
	}
	sort.Strings(result)

	slog.InfoContext(ctx, "temporal permissions processed",
		slog.String("project_id", t.projectID),
		slog.Any("permissions", result),
	)

	return &model.ActionResponse{
		AppendClaims: []model.ClaimItem{
			{
				Key:   temporalClaimName,
				Value: result,
			},
		},
		Claims: map[string]any{
			temporalClaimName: result,
		},
	}, nil
}
