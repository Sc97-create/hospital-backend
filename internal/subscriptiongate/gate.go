package subscriptiongate

import (
	"context"
	"strings"
	"sync"
	"time"

	"hospital-backend/internal/employee/dto"
	notificationdto "hospital-backend/internal/notifications/dto"
	"hospital-backend/pkg/constants"
	"hospital-backend/pkg/logger"

	"go.uber.org/zap"
)

const notifyCooldown = 24 * time.Hour

type staffDirectory interface {
	FindOrganisationIDByUserID(log *zap.Logger, userID string) (string, error)
	FindOne(log *zap.Logger, id string) (dto.EmployeeResponse, error)
}

type notificationEnqueuer interface {
	Create(ctx context.Context, data notificationdto.CreateRequest) error
}

// subscriptionAPI is the hospital-side client that calls central.
type subscriptionAPI interface {
	CheckSubscription(ctx context.Context, organisationID string) (ended bool, message string, err error)
}

// Gate calls the central subscription checkEnd API and emails the user when access is blocked.
type Gate struct {
	API      subscriptionAPI
	Staff    staffDirectory
	Notify   notificationEnqueuer
	notified sync.Map
}

func New(api subscriptionAPI, staff staffDirectory, notify notificationEnqueuer) *Gate {
	return &Gate{API: api, Staff: staff, Notify: notify}
}

// Allow reports whether the user's organisation subscription has ended.
// A true result means usage must be blocked. The returned message is the checkEnd text.
func (g *Gate) Allow(ctx context.Context, userID string) (bool, string, error) {
	if g == nil || g.API == nil || g.Staff == nil || strings.TrimSpace(userID) == "" {
		return false, "", nil
	}
	orgID, err := g.Staff.FindOrganisationIDByUserID(logger.Log, userID)
	if err != nil || strings.TrimSpace(orgID) == "" {
		return false, "", err
	}
	ended, message, err := g.checkEnd(ctx, orgID)
	if err != nil || !ended {
		return false, "", err
	}
	g.notify(ctx, userID, orgID, message)
	return true, message, nil
}

func (g *Gate) checkEnd(ctx context.Context, organisationID string) (bool, string, error) {
	return g.API.CheckSubscription(ctx, organisationID)
}

func (g *Gate) notify(ctx context.Context, userID, organisationID, message string) {
	if g.Notify == nil || !g.shouldNotify(userID) {
		return
	}
	staff, err := g.Staff.FindOne(logger.Log, userID)
	if err != nil || strings.TrimSpace(staff.EmployeeEmail) == "" {
		return
	}
	name := strings.TrimSpace(staff.EmployeeFirstName + " " + staff.EmployeeLastName)
	if name == "" {
		name = staff.EmployeeName
	}
	_ = g.Notify.Create(ctx, notificationdto.CreateRequest{
		NotificationType: constants.SubscriptionEndedEvent,
		Subject:          constants.SubscriptionEndedSubject,
		Data: map[string]interface{}{
			"employee_name":   name,
			"employee_email":  staff.EmployeeEmail,
			"employee_id":     userID,
			"organisation_id": organisationID,
			"message":         message,
		},
	})
}

func (g *Gate) shouldNotify(userID string) bool {
	now := time.Now()
	if previous, ok := g.notified.Load(userID); ok {
		if sent, ok := previous.(time.Time); ok && now.Sub(sent) < notifyCooldown {
			return false
		}
	}
	g.notified.Store(userID, now)
	return true
}
