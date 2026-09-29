package internalapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultHTTPTimeout = 15 * time.Second

// ErrNotConfigured means the base URL or Basic Auth credentials are missing.
var ErrNotConfigured = errors.New("internal api client not configured")

// Client is the single HTTP client for internal APIs. It holds the base URL and Basic Auth.
type Client struct {
	BaseURL         string
	BasicAuthID     string
	BasicAuthSecret string
	HTTPClient      *http.Client
	Customers       customerLookup
}

func New(baseURL, basicAuthID, basicAuthSecret string) *Client {
	return &Client{
		BaseURL:         strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		BasicAuthID:     basicAuthID,
		BasicAuthSecret: basicAuthSecret,
		HTTPClient:      &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// Post sends a JSON body to path with Basic Auth and returns the status and raw body.
func (c *Client) Post(ctx context.Context, path string, payload any) (int, []byte, error) {
	if c == nil || c.BaseURL == "" || c.BasicAuthID == "" || c.BasicAuthSecret == "" || c.HTTPClient == nil {
		return 0, nil, ErrNotConfigured
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.BasicAuthID, c.BasicAuthSecret)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, respBody, nil
}
