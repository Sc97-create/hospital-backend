package dashboard

import (
	apptdto "hospital-backend/internal/appointments/dto"
	billingdto "hospital-backend/internal/billing/dto"
	empdto "hospital-backend/internal/employee/dto"
	rxdto "hospital-backend/internal/prescription/dto"

	"go.uber.org/zap"
)

// AppointmentDashboardReader is the narrow slice of appointments the dashboard needs.
type AppointmentDashboardReader interface {
	GetAppointmentsGroupedByStatus(log *zap.Logger, organisationID string) (apptdto.AppointmentStatusCounts, error)
	GetTodayLatestAppointments(log *zap.Logger, organisationID string) ([]apptdto.AppointmentList, error)
}

// BillingDashboardReader is the narrow slice of billing the dashboard needs.
type BillingDashboardReader interface {
	GetTodayCompletedInvoiceSummary(log *zap.Logger, organisationID string) (billingdto.TodayInvoiceCollectionSummary, error)
}

// EmployeeDashboardReader is the narrow slice of employee the dashboard needs.
type EmployeeDashboardReader interface {
	GetEmployeeStatusCounts(log *zap.Logger, organisationID string) (empdto.EmployeeStatusCounts, error)
}

// PrescriptionDashboardReader is the narrow slice of prescription the dashboard needs.
type PrescriptionDashboardReader interface {
	GetTodayPrescriptions(log *zap.Logger, organisationID string) (rxdto.TodayPrescriptionsSummary, error)
}

type DashboardServicer interface {
	GetAppointmentsGroupedByStatus(log *zap.Logger, organisationID string) (apptdto.AppointmentStatusCounts, error)
	GetTodayLatestAppointments(log *zap.Logger, organisationID string) ([]apptdto.AppointmentList, error)
	GetTodayCompletedInvoiceSummary(log *zap.Logger, organisationID string) (billingdto.TodayInvoiceCollectionSummary, error)
	GetEmployeeStatusCounts(log *zap.Logger, organisationID string) (empdto.EmployeeStatusCounts, error)
	GetTodayPrescriptions(log *zap.Logger, organisationID string) (rxdto.TodayPrescriptionsSummary, error)
}

type DashboardService struct {
	appointments  AppointmentDashboardReader
	billing       BillingDashboardReader
	employees     EmployeeDashboardReader
	prescriptions PrescriptionDashboardReader
}

func NewDashboardService(
	appointments AppointmentDashboardReader,
	billing BillingDashboardReader,
	employees EmployeeDashboardReader,
	prescriptions PrescriptionDashboardReader,
) *DashboardService {
	return &DashboardService{
		appointments:  appointments,
		billing:       billing,
		employees:     employees,
		prescriptions: prescriptions,
	}
}

func (s *DashboardService) GetAppointmentsGroupedByStatus(log *zap.Logger, organisationID string) (apptdto.AppointmentStatusCounts, error) {
	return s.appointments.GetAppointmentsGroupedByStatus(log, organisationID)
}

func (s *DashboardService) GetTodayLatestAppointments(log *zap.Logger, organisationID string) ([]apptdto.AppointmentList, error) {
	return s.appointments.GetTodayLatestAppointments(log, organisationID)
}

func (s *DashboardService) GetTodayCompletedInvoiceSummary(log *zap.Logger, organisationID string) (billingdto.TodayInvoiceCollectionSummary, error) {
	return s.billing.GetTodayCompletedInvoiceSummary(log, organisationID)
}

func (s *DashboardService) GetEmployeeStatusCounts(log *zap.Logger, organisationID string) (empdto.EmployeeStatusCounts, error) {
	return s.employees.GetEmployeeStatusCounts(log, organisationID)
}

func (s *DashboardService) GetTodayPrescriptions(log *zap.Logger, organisationID string) (rxdto.TodayPrescriptionsSummary, error) {
	return s.prescriptions.GetTodayPrescriptions(log, organisationID)
}
