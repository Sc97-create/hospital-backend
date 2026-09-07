package prescription_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/prescription"
	"hospital-backend/internal/prescription/dto"
	prescmocks "hospital-backend/internal/prescription/mocks"
	"hospital-backend/internal/testutil/servicetest"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPrescriptionService(
	t *testing.T,
	db *gorm.DB,
	repo prescription.PrescriptionRepositoryInterface,
	appt prescription.AppointmentLookup,
	status prescription.AppointmentStatusUpdater,
	items prescription.PrescriptionItemAdder,
	notifier prescription.NotificationEnqueuer,
) *prescription.PrescriptionService {
	t.Helper()
	return prescription.NewPrescriptionService(db, repo, appt, status, items, notifier)
}

func testPrescriptionDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

func TestServiceFindMany(t *testing.T) {
	log := servicetest.NopLogger()

	t.Run("invalid status", func(t *testing.T) {
		svc := newPrescriptionService(t, nil, nil, nil, nil, nil, servicetest.NoopNotifier{})
		_, _, err := svc.FindMany(log, dto.FindManyRequest{OrganisationID: "org-1", Status: "bad"})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("find many repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescriptionRepositoryInterface(ctrl)
		repo.EXPECT().FindMany(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
		svc := newPrescriptionService(t, nil, repo, nil, nil, nil, servicetest.NoopNotifier{})
		_, _, err := svc.FindMany(log, dto.FindManyRequest{OrganisationID: "org-1"})
		if !errors.Is(err, wrapError.ErrPrescriptionsFetchFailed) {
			t.Fatalf("expected fetch failed, got %v", err)
		}
	})

	t.Run("count error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescriptionRepositoryInterface(ctrl)
		repo.EXPECT().FindMany(gomock.Any(), gomock.Any(), gomock.Any()).Return([]dto.PrescriptionListItem{}, nil)
		repo.EXPECT().Count(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), errors.New("count error"))
		svc := newPrescriptionService(t, nil, repo, nil, nil, nil, servicetest.NoopNotifier{})
		_, _, err := svc.FindMany(log, dto.FindManyRequest{OrganisationID: "org-1"})
		if !errors.Is(err, wrapError.ErrPrescriptionsFetchFailed) {
			t.Fatalf("expected fetch failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescriptionRepositoryInterface(ctrl)
		list := []dto.PrescriptionListItem{{ID: "rx-1", Code: "PRX001"}}
		repo.EXPECT().FindMany(gomock.Any(), gomock.Any(), gomock.Any()).Return(list, nil)
		repo.EXPECT().Count(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1), nil)
		svc := newPrescriptionService(t, nil, repo, nil, nil, nil, servicetest.NoopNotifier{})
		got, total, err := svc.FindMany(log, dto.FindManyRequest{OrganisationID: "org-1", Limit: 10, PageNo: 1})
		if err != nil || len(got) != 1 || total != 1 {
			t.Fatalf("got %+v total=%d err=%v", got, total, err)
		}
	})
}

func TestServiceGetTodayPrescriptions(t *testing.T) {
	log := servicetest.NopLogger()

	t.Run("missing organisation id", func(t *testing.T) {
		svc := newPrescriptionService(t, nil, nil, nil, nil, nil, servicetest.NoopNotifier{})
		_, err := svc.GetTodayPrescriptions(log, "")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("find many repo error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescriptionRepositoryInterface(ctrl)
		repo.EXPECT().FindMany(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("db error"))
		svc := newPrescriptionService(t, nil, repo, nil, nil, nil, servicetest.NoopNotifier{})
		_, err := svc.GetTodayPrescriptions(log, "org-1")
		if !errors.Is(err, wrapError.ErrPrescriptionsFetchFailed) {
			t.Fatalf("expected fetch failed, got %v", err)
		}
	})

	t.Run("count error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescriptionRepositoryInterface(ctrl)
		repo.EXPECT().FindMany(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]dto.PrescriptionListItem{}, nil)
		repo.EXPECT().Count(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), errors.New("count error"))
		svc := newPrescriptionService(t, nil, repo, nil, nil, nil, servicetest.NoopNotifier{})
		_, err := svc.GetTodayPrescriptions(log, "org-1")
		if !errors.Is(err, wrapError.ErrPrescriptionsFetchFailed) {
			t.Fatalf("expected fetch failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescriptionRepositoryInterface(ctrl)
		list := []dto.PrescriptionListItem{{ID: "rx-1", Code: "PRX001"}}
		repo.EXPECT().FindMany(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(list, nil)
		repo.EXPECT().Count(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(3), nil)
		svc := newPrescriptionService(t, nil, repo, nil, nil, nil, servicetest.NoopNotifier{})
		got, err := svc.GetTodayPrescriptions(log, "org-1")
		if err != nil || len(got.Prescriptions) != 1 || got.Total != 3 {
			t.Fatalf("got %+v err=%v", got, err)
		}
	})
}

func TestServiceFindByStatus(t *testing.T) {
	log := servicetest.NopLogger()

	t.Run("invalid status", func(t *testing.T) {
		svc := newPrescriptionService(t, nil, nil, nil, nil, nil, servicetest.NoopNotifier{})
		_, _, err := svc.FindByStatus(log, 10, 1, "org-1", "bad")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescriptionRepositoryInterface(ctrl)
		list := []dto.PrescriptionListItem{{ID: "rx-1", Status: constants.StatusDraft}}
		repo.EXPECT().FindByStatus(gomock.Any(), "org-1", constants.StatusDraft, 10, 0).Return(list, nil)
		repo.EXPECT().CountByStatus(gomock.Any(), "org-1", constants.StatusDraft).Return(int64(1), nil)
		svc := newPrescriptionService(t, nil, repo, nil, nil, nil, servicetest.NoopNotifier{})
		got, total, err := svc.FindByStatus(log, 10, 1, "org-1", constants.StatusDraft)
		if err != nil || len(got) != 1 || total != 1 {
			t.Fatalf("got %+v total=%d err=%v", got, total, err)
		}
	})
}

func TestServiceGetPrescriptionsByPatientID(t *testing.T) {
	log := servicetest.NopLogger()
	req := dto.PatientPrescriptionsRequest{PatientID: "pat-1", Limit: 10, PageNo: 1}

	t.Run("read error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescriptionRepositoryInterface(ctrl)
		repo.EXPECT().GetPrescriptionsByPatientID(gomock.Any(), gomock.Any(), "pat-1", 10, 0).Return(nil, errors.New("db error"))
		svc := newPrescriptionService(t, nil, repo, nil, nil, nil, servicetest.NoopNotifier{})
		_, err := svc.GetPrescriptionsByPatientID(log, req)
		if !errors.Is(err, wrapError.ErrPrescriptionsFetchFailed) {
			t.Fatalf("expected fetch failed, got %v", err)
		}
	})

	t.Run("count error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescriptionRepositoryInterface(ctrl)
		repo.EXPECT().GetPrescriptionsByPatientID(gomock.Any(), gomock.Any(), "pat-1", 10, 0).Return([]dto.PrescriptionListItem{}, nil)
		repo.EXPECT().GetPrescriptionByPatientIDCount(gomock.Any(), "pat-1").Return(int64(0), errors.New("count error"))
		svc := newPrescriptionService(t, nil, repo, nil, nil, nil, servicetest.NoopNotifier{})
		_, err := svc.GetPrescriptionsByPatientID(log, req)
		if !errors.Is(err, wrapError.ErrPrescriptionsFetchFailed) {
			t.Fatalf("expected fetch failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescriptionRepositoryInterface(ctrl)
		list := []dto.PrescriptionListItem{{ID: "rx-1", PatientID: "pat-1"}}
		repo.EXPECT().GetPrescriptionsByPatientID(gomock.Any(), gomock.Any(), "pat-1", 10, 0).Return(list, nil)
		repo.EXPECT().GetPrescriptionByPatientIDCount(gomock.Any(), "pat-1").Return(int64(1), nil)
		svc := newPrescriptionService(t, nil, repo, nil, nil, nil, servicetest.NoopNotifier{})
		resp, err := svc.GetPrescriptionsByPatientID(log, req)
		if err != nil || resp.Total != 1 {
			t.Fatalf("got %+v err=%v", resp, err)
		}
	})
}

func TestServiceUpdateManualStatus(t *testing.T) {
	log := servicetest.NopLogger()

	t.Run("invalid status", func(t *testing.T) {
		svc := newPrescriptionService(t, nil, nil, nil, nil, nil, servicetest.NoopNotifier{})
		err := svc.UpdateManualStatus(log, "rx-1", "appt-1", "bad")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("missing appointment on sent", func(t *testing.T) {
		svc := newPrescriptionService(t, nil, nil, nil, nil, nil, servicetest.NoopNotifier{})
		err := svc.UpdateManualStatus(log, "rx-1", "", constants.StatusSent)
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("success with draft", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := prescmocks.NewMockPrescriptionRepositoryInterface(ctrl)
		repo.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), constants.StatusDraft, "rx-1").Return(nil)
		svc := newPrescriptionService(t, testPrescriptionDB(t), repo, nil, nil, nil, servicetest.NoopNotifier{})
		if err := svc.UpdateManualStatus(log, "rx-1", "", constants.StatusDraft); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestServiceAddPrescriptionItems(t *testing.T) {
	log := servicetest.NopLogger()
	payload := dto.UpdateRequest{
		PrescriptionID: "rx-1",
		UserID:         "doc-1",
		MedicineArr:    []dto.MedicineArray{servicetest.ValidMedicineArray()},
	}

	t.Run("item adder error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		items := prescmocks.NewMockPrescriptionItemAdder(ctrl)
		items.EXPECT().AddItems(gomock.Any(), gomock.Any(), payload.MedicineArr, "rx-1", "doc-1").Return(errors.New("db error"))
		svc := newPrescriptionService(t, nil, nil, nil, nil, items, servicetest.NoopNotifier{})
		err := svc.AddPrescriptionItems(log, payload)
		if !errors.Is(err, wrapError.ErrPrescriptionUpdateFailed) {
			t.Fatalf("expected update failed, got %v", err)
		}
	})

	t.Run("duplicate medicine", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		items := prescmocks.NewMockPrescriptionItemAdder(ctrl)
		items.EXPECT().AddItems(gomock.Any(), gomock.Any(), payload.MedicineArr, "rx-1", "doc-1").Return(wrapError.ErrMedicineAlreadyPresent)
		svc := newPrescriptionService(t, nil, nil, nil, nil, items, servicetest.NoopNotifier{})
		err := svc.AddPrescriptionItems(log, payload)
		if !errors.Is(err, wrapError.ErrMedicineAlreadyPresent) {
			t.Fatalf("expected duplicate medicine, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		items := prescmocks.NewMockPrescriptionItemAdder(ctrl)
		items.EXPECT().AddItems(gomock.Any(), gomock.Any(), payload.MedicineArr, "rx-1", "doc-1").Return(nil)
		svc := newPrescriptionService(t, nil, nil, nil, nil, items, servicetest.NoopNotifier{})
		if err := svc.AddPrescriptionItems(log, payload); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
