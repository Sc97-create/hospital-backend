package subscriptions

import (
	"strings"

	dto "hospital-backend/central/subscriptions/dto"
	wrapError "hospital-backend/shared/error"
)

const (
	statusActive          = "active"
	statusPendingPayment  = "pending_payment"
	freeTrialDurationDays = 7
	billingCycleTrial     = "7d"
)

var allowedBillingCycles = map[int]struct{}{
	4:  {},
	6:  {},
	12: {},
}

func validateCreatePayload(payload dto.CreateSubscriptionPayload) error {
	if strings.TrimSpace(payload.PlanID) == "" || strings.TrimSpace(payload.TenantID) == "" {
		return wrapError.ErrInvalidRequest
	}
	if payload.Price < 0 {
		return wrapError.ErrInvalidRequest
	}
	return nil
}

func isAllowedBillingCycle(months int) bool {
	_, ok := allowedBillingCycles[months]
	return ok
}
