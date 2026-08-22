package appointments_test

import (
	"errors"
	"testing"
	"time"

	admindto "hospital-backend/internal/admins/dto"
	adminmocks "hospital-backend/internal/admins/mocks"
	"hospital-backend/internal/appointments"
	"hospital-backend/internal/appointments/dto"
	apptmocks "hospital-backend/internal/appointments/mocks"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
)

func TestSelectStatus(t *testing.T) {
	svc := appointments.NewAppointmentService(nil, nil, nil, nil)

	tests := []struct {
		status  string
		wantErr bool
	}{
		{status: "completed", wantErr: false},
		{status: "cancelled", wantErr: false},
		{status: "scheduled", wantErr: false},
		{status: "ongoing", wantErr: false},
		{status: "bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			_, err := svc.SelectStatus(tt.status)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	tests := []struct {
		name    string
		req     dto.UpdateStatus
		setup   func(*apptmocks.MockAppointmentRepository)
		wantErr error
	}{
		{
			name:    "invalid status",
			req:     dto.UpdateStatus{AppointmentID: "appt-1", Status: "bad"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name: "repo error",
			req:  dto.UpdateStatus{AppointmentID: "appt-1", Status: "completed"},
			setup: func(m *apptmocks.MockAppointmentRepository) {
				m.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), appointments.StatusCompleted, "appt-1").Return(errors.New("db"))
			},
			wantErr: wrapError.ErrAppointmentUpdateFailed,
		},
		{
			name: "success",
			req:  dto.UpdateStatus{AppointmentID: "appt-1", Status: "completed"},
			setup: func(m *apptmocks.MockAppointmentRepository) {
				m.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), appointments.StatusCompleted, "appt-1").Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := apptmocks.NewMockAppointmentRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := appointments.NewAppointmentService(nil, repo, nil, servicetest.NoopNotifier{})
			err := svc.UpdateStatus(servicetest.NopLogger(), tt.req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestGetAppntmentByID(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*apptmocks.MockAppointmentRepository)
		wantErr error
	}{
		{
			name: "not found",
			setup: func(m *apptmocks.MockAppointmentRepository) {
				m.EXPECT().GetAppointmentByID(gomock.Any(), "appt-1").Return(appointments.Appointment{}, nil)
			},
			wantErr: wrapError.ErrAppointmentNotFound,
		},
		{
			name: "db error",
			setup: func(m *apptmocks.MockAppointmentRepository) {
				m.EXPECT().GetAppointmentByID(gomock.Any(), "appt-1").Return(appointments.Appointment{}, errors.New("db"))
			},
			wantErr: wrapError.ErrAppointmentFetchFailed,
		},
		{
			name: "success",
			setup: func(m *apptmocks.MockAppointmentRepository) {
				m.EXPECT().GetAppointmentByID(gomock.Any(), "appt-1").Return(appointments.Appointment{ID: "appt-1"}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := apptmocks.NewMockAppointmentRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := appointments.NewAppointmentService(nil, repo, nil, nil)
			got, err := svc.GetAppntmentByID(servicetest.NopLogger(), "appt-1")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != "appt-1" {
				t.Fatalf("expected appt-1, got %q", got.ID)
			}
		})
	}
}

func TestGetAppointmentPreview(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*apptmocks.MockAppointmentRepository)
		wantErr error
	}{
		{
			name: "not found",
			setup: func(m *apptmocks.MockAppointmentRepository) {
				m.EXPECT().GetAppointmentsPreview(gomock.Any(), gomock.Any(), "org-1", "appt-1").Return(nil, nil)
			},
			wantErr: wrapError.ErrAppointmentNotFound,
		},
		{
			name: "success",
			setup: func(m *apptmocks.MockAppointmentRepository) {
				m.EXPECT().GetAppointmentsPreview(gomock.Any(), gomock.Any(), "org-1", "appt-1").Return(map[string]interface{}{
					"appointment_id": "appt-1",
					"name":           "Jane",
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := apptmocks.NewMockAppointmentRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := appointments.NewAppointmentService(nil, repo, nil, nil)
			got, err := svc.GetAppointmentPreview(servicetest.NopLogger(), "org-1", "appt-1")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.AppointmentID != "appt-1" {
				t.Fatalf("expected appt-1, got %q", got.AppointmentID)
			}
		})
	}
}

func TestGetAppointmentsByOrgID(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*apptmocks.MockAppointmentRepository)
		wantErr error
		wantLen int
	}{
		{
			name: "list error",
			setup: func(m *apptmocks.MockAppointmentRepository) {
				m.EXPECT().FindManyByOrganisationID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("db"))
			},
			wantErr: wrapError.ErrAppointmentsFetchFailed,
		},
		{
			name: "success",
			setup: func(m *apptmocks.MockAppointmentRepository) {
				m.EXPECT().FindManyByOrganisationID(gomock.Any(), gomock.Any(), gomock.Any()).Return([]map[string]interface{}{
					{"appointment_id": "appt-1", "status": "scheduled"},
				}, nil)
				m.EXPECT().GetTotalAppointmentsByOrgID(gomock.Any(), gomock.Any(), gomock.Any()).Return(1, nil)
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := apptmocks.NewMockAppointmentRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := appointments.NewAppointmentService(nil, repo, nil, nil)
			got, total, err := svc.GetAppointmentsByOrgID(servicetest.NopLogger(), dto.GetDataReq{
				OrganisationID: "org-1",
				Limit:          10,
				PageNo:         1,
			})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantLen {
				t.Fatalf("expected %d items, got %d", tt.wantLen, len(got))
			}
			if total != tt.wantLen {
				t.Fatalf("expected total %d, got %d", tt.wantLen, total)
			}
		})
	}
}

func TestServiceGetAppointmentByPatientID(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*apptmocks.MockAppointmentRepository)
		wantErr error
	}{
		{
			name: "list error",
			setup: func(m *apptmocks.MockAppointmentRepository) {
				m.EXPECT().GetAppointmentByPatientID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("db"))
			},
			wantErr: wrapError.ErrAppointmentsFetchFailed,
		},
		{
			name: "success",
			setup: func(m *apptmocks.MockAppointmentRepository) {
				m.EXPECT().GetAppointmentByPatientID(gomock.Any(), gomock.Any(), gomock.Any()).Return([]map[string]interface{}{
					{"appointment_id": "appt-1", "status": "scheduled"},
				}, nil)
				m.EXPECT().GetAppointmentByPatientIDCount(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := apptmocks.NewMockAppointmentRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := appointments.NewAppointmentService(nil, repo, nil, nil)
			got, err := svc.GetAppointmentByPatientID(servicetest.NopLogger(), dto.PatientAppntment{
				PatientID:      "pat-1",
				OrganisationID: "org-1",
				Limit:          10,
				Pageno:         1,
			})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Code != appointments.StatusOk {
				t.Fatalf("expected success code, got %q", got.Code)
			}
		})
	}
}

func TestGetNotificationDetails(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := apptmocks.NewMockAppointmentRepository(ctrl)
	repo.EXPECT().GetNotificationsDetails(gomock.Any(), gomock.Any(), "appt-1").Return(map[string]interface{}{
		"patient_name": "Jane",
	}, nil)

	svc := appointments.NewAppointmentService(nil, repo, nil, nil)
	got, err := svc.GetNotificationDetails(servicetest.NopLogger(), "appt-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["patient_name"] != "Jane" {
		t.Fatalf("unexpected payload: %v", got)
	}
}

func TestCreateApptmnt(t *testing.T) {
	payload := servicetest.ValidAppointmentPayload()

	tests := []struct {
		name    string
		payload dto.NewApptmnt
		setup   func(*apptmocks.MockAppointmentRepository, *adminmocks.MockOrganisationScheduleServicer)
		wantErr error
	}{
		{
			name:    "schedule not found",
			payload: payload,
			setup: func(_ *apptmocks.MockAppointmentRepository, sched *adminmocks.MockOrganisationScheduleServicer) {
				sched.EXPECT().GetScheduleByOrganisationID(gomock.Any(), "org-1").Return(admindto.GetResponse{}, wrapError.ErrOrgScheduleNotFound)
			},
			wantErr: wrapError.ErrOrgScheduleNotFound,
		},
		{
			name: "validation fail",
			payload: dto.NewApptmnt{
				OrganisationID: "org-1",
				PatientID:      "",
				DoctorID:       "doc-1",
			},
			setup: func(_ *apptmocks.MockAppointmentRepository, sched *adminmocks.MockOrganisationScheduleServicer) {
				sched.EXPECT().GetScheduleByOrganisationID(gomock.Any(), "org-1").Return(servicetest.ValidOrgSchedule(), nil)
			},
		},
		{
			name:    "repo create error",
			payload: payload,
			setup: func(repo *apptmocks.MockAppointmentRepository, sched *adminmocks.MockOrganisationScheduleServicer) {
				sched.EXPECT().GetScheduleByOrganisationID(gomock.Any(), "org-1").Return(servicetest.ValidOrgSchedule(), nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("db"))
			},
			wantErr: wrapError.ErrAppointmentCreateFailed,
		},
		{
			name:    "success",
			payload: payload,
			setup: func(repo *apptmocks.MockAppointmentRepository, sched *adminmocks.MockOrganisationScheduleServicer) {
				sched.EXPECT().GetScheduleByOrganisationID(gomock.Any(), "org-1").Return(servicetest.ValidOrgSchedule(), nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				repo.EXPECT().GetNotificationsDetails(gomock.Any(), gomock.Any(), gomock.Any()).Return(map[string]interface{}{
					"patient_id": "pat-1",
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := apptmocks.NewMockAppointmentRepository(ctrl)
			sched := adminmocks.NewMockOrganisationScheduleServicer(ctrl)
			if tt.setup != nil {
				tt.setup(repo, sched)
			}
			svc := appointments.NewAppointmentService(nil, repo, sched, servicetest.NoopNotifier{})
			resp, err := svc.CreateApptmnt(servicetest.NopLogger(), tt.payload)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if tt.name == "validation fail" {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.ID == "" {
				t.Fatal("expected appointment id")
			}
		})
	}
}

func TestServiceGetSlots(t *testing.T) {
	future := time.Now().AddDate(0, 0, 1).Format(time.DateOnly)

	tests := []struct {
		name    string
		setup   func(*apptmocks.MockAppointmentRepository, *adminmocks.MockOrganisationScheduleServicer)
		wantErr error
	}{
		{
			name: "schedule not found",
			setup: func(repo *apptmocks.MockAppointmentRepository, sched *adminmocks.MockOrganisationScheduleServicer) {
				repo.EXPECT().GetAppointmentsByIDs(gomock.Any(), gomock.Any(), "doc-1", "org-1", future).Return(nil, nil)
				sched.EXPECT().GetScheduleByOrganisationID(gomock.Any(), "org-1").Return(admindto.GetResponse{}, wrapError.ErrOrgScheduleNotFound)
			},
			wantErr: wrapError.ErrOrgScheduleNotFound,
		},
		{
			name: "repo error",
			setup: func(repo *apptmocks.MockAppointmentRepository, _ *adminmocks.MockOrganisationScheduleServicer) {
				repo.EXPECT().GetAppointmentsByIDs(gomock.Any(), gomock.Any(), "doc-1", "org-1", future).Return(nil, errors.New("db"))
			},
			wantErr: wrapError.ErrAppointmentSlotsFailed,
		},
		{
			name: "success",
			setup: func(repo *apptmocks.MockAppointmentRepository, sched *adminmocks.MockOrganisationScheduleServicer) {
				repo.EXPECT().GetAppointmentsByIDs(gomock.Any(), gomock.Any(), "doc-1", "org-1", future).Return([]appointments.Appointment{}, nil)
				sched.EXPECT().GetScheduleByOrganisationID(gomock.Any(), "org-1").Return(servicetest.ValidOrgSchedule(), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := apptmocks.NewMockAppointmentRepository(ctrl)
			sched := adminmocks.NewMockOrganisationScheduleServicer(ctrl)
			if tt.setup != nil {
				tt.setup(repo, sched)
			}
			svc := appointments.NewAppointmentService(nil, repo, sched, nil)
			resp, err := svc.GetSlots(servicetest.NopLogger(), "doc-1", "org-1", future)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Code != appointments.StatusOk {
				t.Fatalf("expected ok code, got %q", resp.Code)
			}
		})
	}
}
