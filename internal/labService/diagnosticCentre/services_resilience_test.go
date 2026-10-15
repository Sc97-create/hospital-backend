package diagnosticcentre_test

import (
	"errors"
	"testing"

	diagnosticcentre "hospital-backend/internal/labService/diagnosticCentre"
	"hospital-backend/internal/labService/diagnosticCentre/dto"
	"hospital-backend/internal/labService/diagnosticCentre/mocks"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
)

// TestDiagnosticCentreServiceRejectsBadPersistenceInputs ensures invalid status /
// empty ids never call repo.Exec (so bad values are not written to the DB).
func TestDiagnosticCentreServiceRejectsBadPersistenceInputs(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockDiagnosticCentreRepository(ctrl)
	svc := diagnosticcentre.NewDiagnosticCentreService(repo)
	log := servicetest.NopLogger()

	t.Run("create does not persist whitespace-only name", func(t *testing.T) {
		_, err := svc.Create(log, dto.CreateDiagnosticCentreRequest{
			OrganisationID: "org-1",
			Name:           "   ",
		})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("update rejects empty id without exec", func(t *testing.T) {
		err := svc.Update(log, "  ", dto.UpdateDiagnosticCentreRequest{Name: "Lab"})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("update status rejects unknown enum without exec", func(t *testing.T) {
		for _, status := range []string{"", "pending", "DELETED", "1", "true"} {
			err := svc.UpdateStatus(log, "dc-1", status)
			if !errors.Is(err, wrapError.ErrInvalidRequest) {
				t.Fatalf("status %q: expected invalid request, got %v", status, err)
			}
		}
	})

	t.Run("list rejects unknown status without find/count", func(t *testing.T) {
		_, _, err := svc.List(log, dto.ListDiagnosticCentresRequest{
			OrganisationID: "org-1",
			Status:         "archived",
		})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})
}
