package internalapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"hospital-backend/pkg/logger"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
)

const accessTokenPath = "/api/v1/hospital/internal/jwt/accessToken"

var errEmptyAccessToken = errors.New("empty access token")

type accessTokenRequest struct {
	UserID         string `json:"user_id"`
	OrganisationID string `json:"organisation_id"`
}

type accessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// AccessToken asks the internal JWT API for a short-lived access token.
func (c *Client) AccessToken(userID, organisationID string) (string, error) {
	userID = strings.TrimSpace(userID)
	organisationID = strings.TrimSpace(organisationID)
	if userID == "" {
		return "", c.failAccessToken(userID, "missing_user_id", 0, nil)
	}

	status, body, err := c.Post(context.Background(), accessTokenPath, accessTokenRequest{
		UserID:         userID,
		OrganisationID: organisationID,
	})
	if err != nil {
		return "", c.failAccessToken(userID, accessTokenRequestReason(err), 0, err)
	}
	if status != http.StatusOK {
		return "", c.failAccessToken(userID, accessTokenStatusReason(status), status, statusError(status, body))
	}

	token, err := readAccessToken(body)
	if err != nil {
		return "", c.failAccessToken(userID, accessTokenBodyReason(err), status, err)
	}
	return token, nil
}

func (c *Client) failAccessToken(userID, reason string, status int, err error) error {
	fields := []zap.Field{
		zap.String("user_id", userID),
		zap.String("reason", reason),
		zap.String("base_url", c.baseURL()),
		zap.String("path", accessTokenPath),
	}
	if status > 0 {
		fields = append(fields, zap.Int("status", status))
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	accessLogger().Error("access token request failed", fields...)
	if err == nil {
		return fmt.Errorf("%w: %s", wrapError.ErrAccessTokenCreateFailed, reason)
	}
	return fmt.Errorf("%w: %s: %w", wrapError.ErrAccessTokenCreateFailed, reason, err)
}

func (c *Client) baseURL() string {
	if c == nil {
		return ""
	}
	return c.BaseURL
}

func accessTokenRequestReason(err error) string {
	if errors.Is(err, ErrNotConfigured) {
		return "not_configured"
	}
	return "request"
}

func accessTokenStatusReason(status int) string {
	switch status {
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusBadRequest:
		return "rejected"
	case http.StatusInternalServerError:
		return "issuer"
	default:
		return "http_status"
	}
}

func accessTokenBodyReason(err error) string {
	if errors.Is(err, errEmptyAccessToken) {
		return "empty_token"
	}
	return "decode"
}

func statusError(status int, body []byte) error {
	hint := responseHint(body)
	if hint == "" {
		return fmt.Errorf("status %d", status)
	}
	return fmt.Errorf("status %d: %s", status, hint)
}

func responseHint(body []byte) string {
	text := strings.TrimSpace(string(body))
	if strings.Contains(text, "access_token") {
		return ""
	}
	if len(text) > 180 {
		return text[:180]
	}
	return text
}

func readAccessToken(body []byte) (string, error) {
	var out accessTokenResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return "", errEmptyAccessToken
	}
	return out.AccessToken, nil
}

func accessLogger() *zap.Logger {
	if logger.Log != nil {
		return logger.Log
	}
	return zap.NewNop()
}
