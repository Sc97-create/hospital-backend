package dashboard_test

import (
	"errors"
	"testing"

	apptdto "hospital-backend/internal/appointments/dto"
	billingdto "hospital-backend/internal/billing/dto"
	"hospital-backend/internal/dashboard"
	"hospital-backend/internal/dashboard/mocks"
	empdto "hospital-backend/internal/employee/dto"
	rxdto "hospital-backend/internal/prescription/dto"
	"hospital-backend/internal/testutil/servicetest"

	"go.uber.org/mock/gomock"
)

func newDashboardService(
	t *testing.T,
	setup func(
		appt *mocks.MockAppointmentDashboardReader,
		billing *mocks.MockBillingDashboardReader,
		employees *mocks.MockEmployeeDashboardReader,
		prescriptions *mocks.MockPrescriptionDashboardReader,
	),
) *dashboard.DashboardService {
	t.Helper()
	ctrl := gomock.NewController(t)
	appt := mocks.NewMockAppointmentDashboardReader(ctrl)
	billing := mocks.NewMockBillingDashboardReader(ctrl)
	employees := mocks.NewMockEmployeeDashboardReader(ctrl)
	prescriptions := mocks.NewMockPrescriptionDashboardReader(ctrl)
	if setup != nil {
		setup(appt, billing, employees, prescriptions)
	}
	return dashboard.NewDashboardService(appt, billing, employees, prescriptions)
}

func TestDashboardService_GetAppointmentsGroupedByStatus(t *testing.T) {
	svc := newDashboardService(t, func(appt *mocks.MockAppointmentDashboardReader, _ *mocks.MockBillingDashboardReader, _ *mocks.MockEmployeeDashboardReader, _ *mocks.MockPrescriptionDashboardReader) {
		appt.EXPECT().
			GetAppointmentsGroupedByStatus(gomock.Any(), "org-1").
			Return(apptdto.AppointmentStatusCounts{Scheduled: 2}, nil)
	})
	got, err := svc.GetAppointmentsGroupedByStatus(servicetest.NopLogger(), "org-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Scheduled != 2 {
		t.Fatalf("scheduled=%d want 2", got.Scheduled)
	}
}

func TestDashboardService_GetAppointmentsGroupedByStatus_error(t *testing.T) {
	svc := newDashboardService(t, func(appt *mocks.MockAppointmentDashboardReader, _ *mocks.MockBillingDashboardReader, _ *mocks.MockEmployeeDashboardReader, _ *mocks.MockPrescriptionDashboardReader) {
		appt.EXPECT().
			GetAppointmentsGroupedByStatus(gomock.Any(), "org-1").
			Return(apptdto.AppointmentStatusCounts{}, errors.New("db"))
	})
	_, err := svc.GetAppointmentsGroupedByStatus(servicetest.NopLogger(), "org-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDashboardService_GetTodayLatestAppointments(t *testing.T) {
	svc := newDashboardService(t, func(appt *mocks.MockAppointmentDashboardReader, _ *mocks.MockBillingDashboardReader, _ *mocks.MockEmployeeDashboardReader, _ *mocks.MockPrescriptionDashboardReader) {
		appt.EXPECT().
			GetTodayLatestAppointments(gomock.Any(), "org-1").
			Return([]apptdto.AppointmentList{{AppointmentID: "appt-1"}}, nil)
	})
	got, err := svc.GetTodayLatestAppointments(servicetest.NopLogger(), "org-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].AppointmentID != "appt-1" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestDashboardService_GetTodayLatestAppointments_error(t *testing.T) {
	svc := newDashboardService(t, func(appt *mocks.MockAppointmentDashboardReader, _ *mocks.MockBillingDashboardReader, _ *mocks.MockEmployeeDashboardReader, _ *mocks.MockPrescriptionDashboardReader) {
		appt.EXPECT().
			GetTodayLatestAppointments(gomock.Any(), "org-1").
			Return(nil, errors.New("db"))
	})
	_, err := svc.GetTodayLatestAppointments(servicetest.NopLogger(), "org-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDashboardService_GetTodayCompletedInvoiceSummary(t *testing.T) {
	svc := newDashboardService(t, func(_ *mocks.MockAppointmentDashboardReader, billing *mocks.MockBillingDashboardReader, _ *mocks.MockEmployeeDashboardReader, _ *mocks.MockPrescriptionDashboardReader) {
		billing.EXPECT().
			GetTodayCompletedInvoiceSummary(gomock.Any(), "org-1").
			Return(billingdto.TodayInvoiceCollectionSummary{TotalInvoices: 2, TotalAmount: 500}, nil)
	})
	got, err := svc.GetTodayCompletedInvoiceSummary(servicetest.NopLogger(), "org-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalInvoices != 2 || got.TotalAmount != 500 {
		t.Fatalf("unexpected summary: %+v", got)
	}
}

func TestDashboardService_GetTodayCompletedInvoiceSummary_error(t *testing.T) {
	svc := newDashboardService(t, func(_ *mocks.MockAppointmentDashboardReader, billing *mocks.MockBillingDashboardReader, _ *mocks.MockEmployeeDashboardReader, _ *mocks.MockPrescriptionDashboardReader) {
		billing.EXPECT().
			GetTodayCompletedInvoiceSummary(gomock.Any(), "org-1").
			Return(billingdto.TodayInvoiceCollectionSummary{}, errors.New("db"))
	})
	_, err := svc.GetTodayCompletedInvoiceSummary(servicetest.NopLogger(), "org-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDashboardService_GetEmployeeStatusCounts(t *testing.T) {
	svc := newDashboardService(t, func(_ *mocks.MockAppointmentDashboardReader, _ *mocks.MockBillingDashboardReader, employees *mocks.MockEmployeeDashboardReader, _ *mocks.MockPrescriptionDashboardReader) {
		employees.EXPECT().
			GetEmployeeStatusCounts(gomock.Any(), "org-1").
			Return(empdto.EmployeeStatusCounts{Active: 3, Inactive: 1, Total: 4}, nil)
	})
	got, err := svc.GetEmployeeStatusCounts(servicetest.NopLogger(), "org-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Active != 3 || got.Inactive != 1 || got.Total != 4 {
		t.Fatalf("unexpected counts: %+v", got)
	}
}

func TestDashboardService_GetEmployeeStatusCounts_error(t *testing.T) {
	svc := newDashboardService(t, func(_ *mocks.MockAppointmentDashboardReader, _ *mocks.MockBillingDashboardReader, employees *mocks.MockEmployeeDashboardReader, _ *mocks.MockPrescriptionDashboardReader) {
		employees.EXPECT().
			GetEmployeeStatusCounts(gomock.Any(), "org-1").
			Return(empdto.EmployeeStatusCounts{}, errors.New("db"))
	})
	_, err := svc.GetEmployeeStatusCounts(servicetest.NopLogger(), "org-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDashboardService_GetTodayPrescriptions(t *testing.T) {
	svc := newDashboardService(t, func(_ *mocks.MockAppointmentDashboardReader, _ *mocks.MockBillingDashboardReader, _ *mocks.MockEmployeeDashboardReader, prescriptions *mocks.MockPrescriptionDashboardReader) {
		prescriptions.EXPECT().
			GetTodayPrescriptions(gomock.Any(), "org-1").
			Return(rxdto.TodayPrescriptionsSummary{
				Prescriptions: []rxdto.PrescriptionListItem{{ID: "rx-1"}},
				Total:         2,
			}, nil)
	})
	got, err := svc.GetTodayPrescriptions(servicetest.NopLogger(), "org-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Prescriptions) != 1 || got.Total != 2 {
		t.Fatalf("unexpected summary: %+v", got)
	}
}

func TestDashboardService_GetTodayPrescriptions_error(t *testing.T) {
	svc := newDashboardService(t, func(_ *mocks.MockAppointmentDashboardReader, _ *mocks.MockBillingDashboardReader, _ *mocks.MockEmployeeDashboardReader, prescriptions *mocks.MockPrescriptionDashboardReader) {
		prescriptions.EXPECT().
			GetTodayPrescriptions(gomock.Any(), "org-1").
			Return(rxdto.TodayPrescriptionsSummary{}, errors.New("db"))
	})
	_, err := svc.GetTodayPrescriptions(servicetest.NopLogger(), "org-1")
	if err == nil {
		t.Fatal("expected error")
	}
}
