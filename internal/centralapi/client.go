package centralapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	checkEndPath       = "/api/v1/central/internal/subscription/checkEnd/"
	defaultHTTPTimeout = 15 * time.Second
)

// ErrRequestFailed means the central API did not return a usable response.
var ErrRequestFailed = errors.New("central api request failed")

// Client calls central APIs from the hospital service. It keeps its own base URL and Basic Auth.
type Client struct {
	BaseURL    string
	ClientID   string
	Secret     string
	HTTPClient *http.Client
}

func New(baseURL, clientID, secret string) *Client {
	return &Client{
		BaseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		ClientID:   clientID,
		Secret:     secret,
		HTTPClient: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

// CheckSubscription calls central subscription checkEnd for an organisation.
func (c *Client) CheckSubscription(ctx context.Context, organisationID string) (bool, string, error) {
	organisationID = strings.TrimSpace(organisationID)
	if c == nil || c.BaseURL == "" || c.ClientID == "" || c.Secret == "" || c.HTTPClient == nil || organisationID == "" {
		return false, "", ErrRequestFailed
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+checkEndPath+url.PathEscape(organisationID), nil)
	if err != nil {
		return false, "", ErrRequestFailed
	}
	req.SetBasicAuth(c.ClientID, c.Secret)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false, "", ErrRequestFailed
	}
	var out struct {
		Message string `json:"message"`
		Ended   bool   `json:"ended"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return false, "", ErrRequestFailed
	}
	return out.Ended, out.Message, nil
}
