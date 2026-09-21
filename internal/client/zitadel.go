package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ckchessmaster/zitadel-actions-api/internal/model"
)

// GrantFetcher defines the interface for fetching user grants from an external identity provider.
type GrantFetcher interface {
	FetchUserGrants(ctx context.Context, userID string) ([]model.UserGrant, error)
}

// Client interacts with the ZITADEL Management API using a Personal Access Token (PAT).
type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

// New creates a new ZITADEL Management API client.
func New(baseURL, apiToken string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 5 * time.Second,
		}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiToken:   strings.TrimSpace(apiToken),
		httpClient: httpClient,
	}
}

type userGrantsSearchRequest struct {
	Queries []grantQuery `json:"queries"`
}

type grantQuery struct {
	UserIDQuery *userIDQuery `json:"userIdQuery,omitempty"`
}

type userIDQuery struct {
	UserID string `json:"userId"`
}

type userGrantsSearchResponse struct {
	Result []struct {
		ProjectID                  string   `json:"projectId"`
		Roles                      []string `json:"roles"`
		UserGrantResourceOwner     string   `json:"userGrantResourceOwner,omitempty"`
		UserGrantResourceOwnerName string   `json:"userGrantResourceOwnerName,omitempty"`
	} `json:"result"`
}

// FetchUserGrants retrieves the list of user grants for the given user ID.
func (c *Client) FetchUserGrants(ctx context.Context, userID string) ([]model.UserGrant, error) {
	trimmedID := strings.TrimSpace(userID)
	if trimmedID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	searchReq := userGrantsSearchRequest{
		Queries: []grantQuery{
			{
				UserIDQuery: &userIDQuery{
					UserID: trimmedID,
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	url := fmt.Sprintf("%s/management/v1/users/grants/_search", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if c.apiToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiToken)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("zitadel api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read zitadel api response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("zitadel api returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var searchResp userGrantsSearchResponse
	if err := json.Unmarshal(respBytes, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode zitadel api response: %w", err)
	}

	grants := make([]model.UserGrant, 0, len(searchResp.Result))
	for _, r := range searchResp.Result {
		grants = append(grants, model.UserGrant{
			ProjectID:                  r.ProjectID,
			Roles:                      r.Roles,
			UserGrantResourceOwner:     r.UserGrantResourceOwner,
			UserGrantResourceOwnerName: r.UserGrantResourceOwnerName,
		})
	}

	return grants, nil
}
