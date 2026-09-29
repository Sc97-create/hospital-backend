package plans

import (
	"time"

	"hospital-backend/pkg/constants"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	PlanNameFreeTrial = "free trial"
	PlanNameStandard  = "standard"
)

// AutoMigrate creates the plans table and seeds default catalog rows.
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Plan{}); err != nil {
		return err
	}
	return SeedDefaultPlans(db)
}

// SeedDefaultPlans upserts the free-trial and standard plans by name.
func SeedDefaultPlans(db *gorm.DB) error {
	now := time.Now()
	seeds := []Plan{
		{
			ID:           uuid.NewString(),
			Name:         PlanNameFreeTrial,
			Status:       constants.StatusActive,
			MonthlyPrice: 0,
			Discount:     0,
			PlanDetails: PlanDetails{
				"max_organisations":              float64(1),
				"max_employees_per_organisation": float64(5),
				"max_patients":                   float64(100),
				"max_payment_transactions":       float64(20),
				"max_appointments":               float64(-1),
				"patient_history_days":           float64(7),
				"trial_days":                     float64(7),
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:           uuid.NewString(),
			Name:         PlanNameStandard,
			Status:       constants.StatusActive,
			MonthlyPrice: 2000,
			Discount:     10,
			PlanDetails: PlanDetails{
				"max_organisations":              float64(3),
				"max_employees_per_organisation": float64(10),
				"max_patients":                   float64(-1),
				"max_payment_transactions":       float64(-1),
				"max_appointments":               float64(-1),
				"patient_history_days":           float64(-1),
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	for _, plan := range seeds {
		row := plan
		err := db.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "name"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"status",
				"monthly_price",
				"discount",
				"plan_details",
				"updated_at",
			}),
		}).Create(&row).Error
		if err != nil {
			return err
		}
	}
	return nil
}
