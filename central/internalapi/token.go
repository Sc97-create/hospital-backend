package internalapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	wrapError "hospital-backend/shared/error"
)

const accessTokenPath = "/api/v1/hospital/internal/jwt/accessToken"

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
		return "", wrapError.ErrAccessTokenCreateFailed
	}
	status, body, err := c.Post(context.Background(), accessTokenPath, accessTokenRequest{
		UserID:         userID,
		OrganisationID: organisationID,
	})
	if err != nil || status != http.StatusOK {
		return "", wrapError.ErrAccessTokenCreateFailed
	}
	var out accessTokenResponse
	if err := json.Unmarshal(body, &out); err != nil || strings.TrimSpace(out.AccessToken) == "" {
		return "", wrapError.ErrAccessTokenCreateFailed
	}
	return out.AccessToken, nil
}
