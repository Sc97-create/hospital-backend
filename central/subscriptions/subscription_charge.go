package subscriptions

import (
	"context"
	"errors"
	"strings"

	"hospital-backend/central/payments"
	paymentdto "hospital-backend/central/payments/dto"
	wrapError "hospital-backend/shared/error"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// paidSubscription is the data needed to open a Razorpay order, invoice, and payment.
type paidSubscription struct {
	Sub            Subscription
	HospitalName   string
	CustomerID     string
	OrganisationID string
	Price          float64
	OrderID        string
	Renewal        bool
}

func (s *SubscriptionService) openPaidSubscription(log *zap.Logger, ctx context.Context, paid paidSubscription) (string, error) {
	customerID, organisationID, err := s.chargeParty(log, paid.Sub.TenantID, paid.CustomerID, paid.OrganisationID)
	if err != nil {
		return "", err
	}
	orderID, err := s.createSubscriptionOrder(log, ctx, paid.Sub, paid.HospitalName, paid.Price)
	if err != nil {
		return "", err
	}
	paid.CustomerID = customerID
	paid.OrganisationID = organisationID
	paid.OrderID = orderID
	if err := s.recordSubscriptionCharge(log, ctx, paid); err != nil {
		return "", err
	}
	return orderID, nil
}

func (s *SubscriptionService) chargeParty(log *zap.Logger, tenantID, customerID, organisationID string) (string, string, error) {
	customerID = strings.TrimSpace(customerID)
	organisationID = strings.TrimSpace(organisationID)
	if _, err := uuid.Parse(customerID); err != nil {
		log.Warn("subscription charge failed", zap.String("reason", "invalid_customer_id"))
		return "", "", wrapError.ErrInvalidRequest
	}
	if organisationID == "" {
		id, err := s.organisationIDByTenant(log, tenantID)
		if err != nil {
			return "", "", err
		}
		organisationID = id
	}
	if _, err := uuid.Parse(organisationID); err != nil {
		log.Warn("subscription charge failed", zap.String("reason", "invalid_organisation_id"))
		return "", "", wrapError.ErrInvalidRequest
	}
	return customerID, organisationID, nil
}

func (s *SubscriptionService) organisationIDByTenant(log *zap.Logger, tenantID string) (string, error) {
	if s.DB == nil {
		log.Error("subscription charge failed", zap.String("reason", "db_missing"))
		return "", wrapError.ErrOrganisationFetchFailed
	}
	var row struct {
		ID string `gorm:"column:id"`
	}
	err := s.DB.Table("organisations").Select("id").Where("tenant_id = ?", tenantID).
		Order("is_primary DESC").Limit(1).Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("subscription charge failed", zap.String("tenant_id", tenantID), zap.String("reason", "organisation_not_found"))
			return "", wrapError.ErrOrganisationNotFound
		}
		log.Error("subscription charge failed", zap.String("tenant_id", tenantID), zap.String("reason", "organisation_lookup"), zap.Error(err))
		return "", wrapError.ErrOrganisationFetchFailed
	}
	if strings.TrimSpace(row.ID) == "" {
		return "", wrapError.ErrOrganisationNotFound
	}
	return row.ID, nil
}

func (s *SubscriptionService) recordSubscriptionCharge(log *zap.Logger, ctx context.Context, paid paidSubscription) error {
	if s.Billing == nil {
		log.Error("subscription charge failed", zap.String("subscription_id", paid.Sub.ID), zap.String("reason", "billing_client_missing"))
		return wrapError.ErrOrderCreateFailed
	}
	invoice, err := s.Billing.CreateSubscriptionInvoice(log, ctx, paymentdto.SubscriptionInvoiceRequest{
		SubscriptionID: paid.Sub.ID,
		CustomerID:     paid.CustomerID,
		OrganisationID: paid.OrganisationID,
		Amount:         paid.Price,
		Renewal:        paid.Renewal,
	})
	if err != nil {
		log.Error("subscription charge failed", zap.String("subscription_id", paid.Sub.ID), zap.String("reason", "invoice_api"), zap.Error(err))
		return err
	}
	err = s.Billing.RecordSubscriptionPayment(log, ctx, paymentdto.SubscriptionPaymentRequest{
		InvoiceID:  invoice.InvoiceID,
		CustomerID: paid.CustomerID,
		Amount:     paid.Price,
		OrderID:    paid.OrderID,
		Currency:   payments.CurrencyINR,
	})
	if err != nil {
		log.Error("subscription charge failed",
			zap.String("subscription_id", paid.Sub.ID),
			zap.String("invoice_id", invoice.InvoiceID),
			zap.String("order_id", paid.OrderID),
			zap.String("reason", "payment_api"),
			zap.Error(err),
		)
		return err
	}
	return nil
}

func (s *SubscriptionService) latestOrderID(log *zap.Logger, subscriptionID string) string {
	if s == nil || s.DB == nil || strings.TrimSpace(subscriptionID) == "" {
		return ""
	}
	var orderID string
	err := s.DB.Raw(`
		SELECT p.order_id
		FROM invoices i
		JOIN payments p ON p.invoice_id = i.id
		WHERE i.billing_id = ?
		  AND i.payment_type IN ('subscription_new', 'subscription_renewal')
		ORDER BY p.created_at DESC
		LIMIT 1
	`, subscriptionID).Scan(&orderID).Error
	if err != nil {
		log.Warn("subscription lookup order failed", zap.String("subscription_id", subscriptionID), zap.Error(err))
		return ""
	}
	return strings.TrimSpace(orderID)
}
