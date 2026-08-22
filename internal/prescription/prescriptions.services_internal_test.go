package prescription

import (
	"errors"
	"testing"

	invoicedto "hospital-backend/internal/billing/dto"
	"hospital-backend/internal/prescription/dto"
	"hospital-backend/internal/testutil/servicetest"
	"hospital-backend/pkg/constants"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type stubPrescriptionRepo struct {
	updateStatusErr error
	lastStatus      string
	lastPrescID     string
}

func (s *stubPrescriptionRepo) CreatePrescription(_ *zap.Logger, _ *gorm.DB, _ Prescription) error {
	return nil
}
func (s *stubPrescriptionRepo) FindMany(_ *zap.Logger, _ string, _ ...any) ([]dto.PrescriptionListItem, error) {
	return nil, nil
}
func (s *stubPrescriptionRepo) Count(_ *zap.Logger, _ string, _ ...any) (int64, error) {
	return 0, nil
}
func (s *stubPrescriptionRepo) FindByStatus(_ *zap.Logger, _, _ string, _, _ int) ([]dto.PrescriptionListItem, error) {
	return nil, nil
}
func (s *stubPrescriptionRepo) CountByStatus(_ *zap.Logger, _, _ string) (int64, error) {
	return 0, nil
}
func (s *stubPrescriptionRepo) UpdateStatus(_ *zap.Logger, _ *gorm.DB, status, prescriptionID string) error {
	s.lastStatus = status
	s.lastPrescID = prescriptionID
	return s.updateStatusErr
}
func (s *stubPrescriptionRepo) GetPrescriptionsByPatientID(_ *zap.Logger, _ string, _ ...any) ([]dto.PrescriptionListItem, error) {
	return nil, nil
}
func (s *stubPrescriptionRepo) GetPrescriptionByPatientIDCount(_ *zap.Logger, _ string) (int64, error) {
	return 0, nil
}
func (s *stubPrescriptionRepo) GetPrescriptionsByAppointmentID(_ *zap.Logger, _ string, _ ...any) ([]PrescriptionAppointmentData, error) {
	return nil, nil
}
func (s *stubPrescriptionRepo) GetPrescriptionByAppointmentIDCount(_ *zap.Logger, _ ...any) (int64, error) {
	return 0, nil
}
func (s *stubPrescriptionRepo) GetNotificationDetails(_ *zap.Logger, _, _ string) (PrescriptionNotificationData, error) {
	return PrescriptionNotificationData{}, nil
}
func (s *stubPrescriptionRepo) FindPrescriptionByID(_ *zap.Logger, _, _ string) (Prescription, error) {
	return Prescription{}, nil
}
func (s *stubPrescriptionRepo) GetPrescriptionByID(_ *zap.Logger, _ string) (*Prescription, error) {
	return nil, nil
}
func (s *stubPrescriptionRepo) GetPrescriptionsByDoctorID(_ *zap.Logger, _ string) ([]Prescription, error) {
	return nil, nil
}
func (s *stubPrescriptionRepo) DeletePrescription(_ *zap.Logger, _ string) error {
	return nil
}

func TestParseFilterStatus(t *testing.T) {
	svc := &PrescriptionService{}

	got, err := svc.parseFilterStatus(constants.StatusDraft)
	if err != nil || got != constants.StatusDraft {
		t.Fatalf("draft: got %q err=%v", got, err)
	}

	got, err = svc.parseFilterStatus(constants.StatusPaymentLinkCreated)
	if err != nil || got != constants.StatusPaymentPending {
		t.Fatalf("payment link alias: got %q err=%v", got, err)
	}

	if _, err := svc.parseFilterStatus("bad"); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestParseManualStatus(t *testing.T) {
	svc := &PrescriptionService{}

	got, err := svc.parseManualStatus(constants.StatusCancelled)
	if err != nil || got != constants.StatusCancelled {
		t.Fatalf("cancelled: got %q err=%v", got, err)
	}

	if _, err := svc.parseManualStatus("bad"); !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestPrescriptionParsePagination(t *testing.T) {
	svc := &PrescriptionService{}

	limit, skip := svc.parsePagination(0, 0)
	if limit != 10 || skip != 0 {
		t.Fatalf("defaults: got limit=%d skip=%d", limit, skip)
	}

	limit, skip = svc.parsePagination(20, 2)
	if limit != 20 || skip != 20 {
		t.Fatalf("page 2: got limit=%d skip=%d", limit, skip)
	}
}

func TestResolveAndUpdateParentStatus(t *testing.T) {
	log := servicetest.NopLogger()

	t.Run("all dispensed to completed", func(t *testing.T) {
		repo := &stubPrescriptionRepo{}
		svc := &PrescriptionService{prescriptionRepo: repo}
		items := []invoicedto.MedInvoiceItemResponse{
			{PrescriptionItemStatus: constants.StatusFullyDispensed},
			{PrescriptionItemStatus: constants.StatusFullyDispensed},
		}
		if err := svc.ResolveAndUpdateParentStatus(log, nil, "rx-1", items); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.lastStatus != constants.StatusCompleted || repo.lastPrescID != "rx-1" {
			t.Fatalf("expected completed for rx-1, got status=%q id=%q", repo.lastStatus, repo.lastPrescID)
		}
	})

	t.Run("partial to tentative", func(t *testing.T) {
		repo := &stubPrescriptionRepo{}
		svc := &PrescriptionService{prescriptionRepo: repo}
		items := []invoicedto.MedInvoiceItemResponse{
			{PrescriptionItemStatus: constants.StatusFullyDispensed},
			{PrescriptionItemStatus: constants.StatusPartiallyDispensed},
		}
		if err := svc.ResolveAndUpdateParentStatus(log, nil, "rx-1", items); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.lastStatus != constants.StatusTentative || repo.lastPrescID != "rx-1" {
			t.Fatalf("expected tentative for rx-1, got status=%q id=%q", repo.lastStatus, repo.lastPrescID)
		}
	})
}

func TestUpdateExtPrescriptionStatus(t *testing.T) {
	log := servicetest.NopLogger()
	svc := &PrescriptionService{}

	t.Run("invalid status", func(t *testing.T) {
		err := svc.UpdateExtPrescriptionStatus(log, nil, "rx-1", "bad")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("repo error", func(t *testing.T) {
		repo := &stubPrescriptionRepo{updateStatusErr: errors.New("db error")}
		svc := &PrescriptionService{prescriptionRepo: repo}
		err := svc.UpdateExtPrescriptionStatus(log, nil, "rx-1", constants.StatusCompleted)
		if !errors.Is(err, wrapError.ErrPrescriptionUpdateFailed) {
			t.Fatalf("expected update failed, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo := &stubPrescriptionRepo{}
		svc := &PrescriptionService{prescriptionRepo: repo}
		if err := svc.UpdateExtPrescriptionStatus(log, nil, "rx-1", constants.StatusTentative); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.lastStatus != constants.StatusTentative || repo.lastPrescID != "rx-1" {
			t.Fatalf("expected tentative for rx-1, got status=%q id=%q", repo.lastStatus, repo.lastPrescID)
		}
	})
}
