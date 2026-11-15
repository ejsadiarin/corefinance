// Package gateway is corefinance's client for the few coregateway calls it
// makes itself, outside a proxied request.
package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TimezoneClient looks users' timezones up from the gateway
// (GET /api/users/{id}/timezone, coregateway ADR-0005) for the worker, which
// has no internal token to read them from. It authenticates with an API key
// owned by an admin and holding the corefinance:admin scope.
type TimezoneClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewTimezoneClient is a client for the gateway at baseURL (e.g.
// http://coregateway-api:8080), authenticating with apiKey.
func NewTimezoneClient(baseURL, apiKey string) *TimezoneClient {
	return &TimezoneClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

type timezoneResponse struct {
	Timezone string `json:"timezone"`
}

// Zone is the user's timezone: UTC for a user who hasn't set one. Every other
// outcome (no answer, a non-200, a zone this build can't load) is an error,
// so the caller skips the user instead of guessing a zone.
func (c *TimezoneClient) Zone(ctx context.Context, userID uuid.UUID) (*time.Location, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/users/"+userID.String()+"/timezone", nil)
	if err != nil {
		return nil, fmt.Errorf("gateway timezone lookup: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gateway timezone lookup: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return nil, fmt.Errorf("gateway timezone lookup: %s: %s", res.Status, strings.TrimSpace(string(body)))
	}
	var out timezoneResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("gateway timezone lookup: decode: %w", err)
	}
	if out.Timezone == "" {
		return nil, fmt.Errorf("gateway timezone lookup: no zone in the answer")
	}
	loc, err := time.LoadLocation(out.Timezone)
	if err != nil {
		return nil, fmt.Errorf("gateway timezone lookup: zone %q: %w", out.Timezone, err)
	}
	return loc, nil
}
