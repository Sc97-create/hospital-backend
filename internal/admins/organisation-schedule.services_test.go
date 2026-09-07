package admins_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/admins"
	"hospital-backend/internal/admins/mocks"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func newOrgScheduleService(t *testing.T, repo admins.OrganisationScheduleRepository) *admins.OrganisationScheduleService {
	t.Helper()
	return admins.NewOrganisationScheduleService(repo)
}

func TestServiceCreate(t *testing.T) {
	log := servicetest.NopLogger()
	req := servicetest.ValidOrgScheduleReq()

	t.Run("repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockOrganisationScheduleRepository(ctrl)
		repo.EXPECT().Create(log, gomock.Any()).Return(errors.New("create failed"))

		svc := newOrgScheduleService(t, repo)
		if err := svc.Create(log, req); !errors.Is(err, wrapError.ErrOrgScheduleCreateFailed) {
			t.Fatalf("expected create failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockOrganisationScheduleRepository(ctrl)
		repo.EXPECT().Create(log, gomock.Any()).DoAndReturn(
			func(_ interface{}, sched *admins.OrganisationSchedule) error {
				if sched.OrganisationID != req.OrganisationID {
					t.Errorf("expected org %q, got %q", req.OrganisationID, sched.OrganisationID)
				}
				if sched.StartTime != req.StartTime || sched.EndTime != req.EndTime {
					t.Errorf("unexpected times: %+v", sched)
				}
				if sched.SlotDuration != int(req.SlotDuration) {
					t.Errorf("expected slot duration %d, got %d", int(req.SlotDuration), sched.SlotDuration)
				}
				if sched.ID == "" {
					t.Error("expected generated schedule id")
				}
				return nil
			},
		)

		svc := newOrgScheduleService(t, repo)
		if err := svc.Create(log, req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestServiceGetScheduleByOrganisationID(t *testing.T) {
	log := servicetest.NopLogger()

	tests := []struct {
		name    string
		orgID   string
		setup   func(*mocks.MockOrganisationScheduleRepository)
		wantErr error
		wantID  string
	}{
		{
			name:    "empty org id",
			orgID:   "  ",
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:  "not found",
			orgID: "org-missing",
			setup: func(m *mocks.MockOrganisationScheduleRepository) {
				m.EXPECT().GetByOrganisationID(log, gomock.Any(), "org-missing").Return(admins.OrganisationSchedule{}, gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrOrgScheduleNotFound,
		},
		{
			name:  "db error",
			orgID: "org-1",
			setup: func(m *mocks.MockOrganisationScheduleRepository) {
				m.EXPECT().GetByOrganisationID(log, gomock.Any(), "org-1").Return(admins.OrganisationSchedule{}, errors.New("db error"))
			},
			wantErr: wrapError.ErrOrgScheduleFetchFailed,
		},
		{
			name:  "success",
			orgID: "org-1",
			setup: func(m *mocks.MockOrganisationScheduleRepository) {
				m.EXPECT().GetByOrganisationID(log, gomock.Any(), "org-1").Return(admins.OrganisationSchedule{
					ID:             "sched-1",
					StartTime:      "09:00",
					EndTime:        "17:00",
					BreakStartTime: "13:00",
					BreakEndTime:   "14:00",
					SlotDuration:   30,
				}, nil)
			},
			wantID: "sched-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockOrganisationScheduleRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}

			svc := newOrgScheduleService(t, repo)
			got, err := svc.GetScheduleByOrganisationID(log, tt.orgID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tt.wantID {
				t.Fatalf("expected id %q, got %q", tt.wantID, got.ID)
			}
			if got.Slotduration != 30 {
				t.Fatalf("expected slot duration 30, got %d", got.Slotduration)
			}
		})
	}
}
