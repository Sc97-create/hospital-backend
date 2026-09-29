package plans_test

import (
	"errors"
	"testing"

	"hospital-backend/central/plans"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
)

type stubPlanRepo struct {
	rows []plans.Plan
	err  error
}

func (s *stubPlanRepo) GetByID(_ *zap.Logger, _ string) (plans.Plan, error) {
	return plans.Plan{}, nil
}

func (s *stubPlanRepo) GetByName(_ *zap.Logger, _ string) (plans.Plan, error) {
	return plans.Plan{}, nil
}

func (s *stubPlanRepo) ListActive(_ *zap.Logger) ([]plans.Plan, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rows, nil
}

func TestServiceListPlans(t *testing.T) {
	log := servicetest.NopLogger()

	t.Run("repo failure", func(t *testing.T) {
		svc := plans.NewPlanService(&stubPlanRepo{err: errors.New("db")})
		_, err := svc.ListPlans(log)
		if !errors.Is(err, wrapError.ErrPlanFetchFailed) {
			t.Fatalf("err=%v want=%v", err, wrapError.ErrPlanFetchFailed)
		}
	})

	t.Run("success includes effective price", func(t *testing.T) {
		svc := plans.NewPlanService(&stubPlanRepo{rows: []plans.Plan{
			{
				ID:           "p1",
				Name:         plans.PlanNameFreeTrial,
				Status:       "active",
				MonthlyPrice: 0,
				Discount:     0,
				PlanDetails:  plans.PlanDetails{"max_organisations": float64(1)},
			},
			{
				ID:           "p2",
				Name:         plans.PlanNameStandard,
				Status:       "active",
				MonthlyPrice: 2000,
				Discount:     10,
				PlanDetails:  plans.PlanDetails{"max_organisations": float64(3)},
			},
		}})
		got, err := svc.ListPlans(log)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len=%d want 2", len(got))
		}
		if got[1].ID != "p2" || got[1].EffectiveMonthlyPrice != 1800 {
			t.Fatalf("unexpected standard plan: %+v", got[1])
		}
	})
}
