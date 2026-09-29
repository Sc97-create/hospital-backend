package internalapi

import (
	"context"
	"net/http"
	"strings"

	"hospital-backend/central/customers"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
)

const addFirstUserPath = "/api/v1/hospital/internal/organisation/addFirstUser"

type customerLookup interface {
	GetByTenantID(log *zap.Logger, tenantID string) (customers.Customer, error)
}

type addFirstUserRequest struct {
	OrganisationID string `json:"organisation_id"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	EmailID        string `json:"email_id"`
	Username       string `json:"username"`
}

// ProvisionHospitalAdmin loads the tenant customer and posts them to the internal first-user API.
func (c *Client) ProvisionHospitalAdmin(log *zap.Logger, tenantID, organisationID string) error {
	log = ensureLog(log)
	tenantID = strings.TrimSpace(tenantID)
	organisationID = strings.TrimSpace(organisationID)
	if tenantID == "" || organisationID == "" || c == nil || c.Customers == nil {
		log.Warn("hospital admin api call failed", zap.String("reason", "validation"))
		return wrapError.ErrInvalidRequest
	}
	customer, err := c.Customers.GetByTenantID(log, tenantID)
	if err != nil {
		log.Error("hospital admin api call failed",
			zap.String("tenant_id", tenantID),
			zap.String("organisation_id", organisationID),
			zap.String("reason", "customer_lookup"),
			zap.Error(err),
		)
		return err
	}
	status, body, err := c.Post(context.Background(), addFirstUserPath, firstUserBody(organisationID, customer))
	if err != nil {
		log.Error("hospital admin api call failed", zap.String("reason", "http_do"), zap.Error(err))
		return wrapError.ErrOrganisationSetupFailed
	}
	return mapFirstUserStatus(log, status, body)
}

func mapFirstUserStatus(log *zap.Logger, status int, body []byte) error {
	if status == http.StatusOK {
		return nil
	}
	if status == http.StatusUnauthorized {
		log.Warn("hospital admin api call failed", zap.String("reason", "unauthorized"), zap.Int("status", status))
		return wrapError.ErrUnauthorized
	}
	if status == http.StatusBadRequest {
		log.Warn("hospital admin api call failed", zap.String("reason", "bad_request"), zap.Int("status", status), zap.ByteString("body", body))
		return wrapError.ErrInvalidRequest
	}
	log.Error("hospital admin api call failed", zap.String("reason", "non_2xx"), zap.Int("status", status), zap.ByteString("body", body))
	return wrapError.ErrOrganisationSetupFailed
}

func firstUserBody(organisationID string, customer customers.Customer) addFirstUserRequest {
	firstName, lastName := splitFullName(customer.FullName)
	return addFirstUserRequest{
		OrganisationID: organisationID,
		FirstName:      firstName,
		LastName:       lastName,
		EmailID:        customer.WorkEmail,
		Username:       usernameFromEmail(customer.WorkEmail),
	}
}

func splitFullName(fullName string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(fullName))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func usernameFromEmail(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	local, _, found := strings.Cut(email, "@")
	if !found || local == "" {
		return email
	}
	return local
}
