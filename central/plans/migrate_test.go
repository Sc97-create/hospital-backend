package plans_test

import (
	"testing"

	"hospital-backend/central/plans"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSeedDefaultPlans(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:plans_seed?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := plans.AutoMigrate(db); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	var count int64
	if err := db.Model(&plans.Plan{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("count=%d want 2", count)
	}

	// Idempotent re-seed should not duplicate.
	if err := plans.SeedDefaultPlans(db); err != nil {
		t.Fatalf("SeedDefaultPlans: %v", err)
	}
	if err := db.Model(&plans.Plan{}).Count(&count).Error; err != nil {
		t.Fatalf("recount: %v", err)
	}
	if count != 2 {
		t.Fatalf("after reseed count=%d want 2", count)
	}

	var standard plans.Plan
	if err := db.Where("name = ?", plans.PlanNameStandard).First(&standard).Error; err != nil {
		t.Fatalf("load standard: %v", err)
	}
	if standard.MonthlyPrice != 2000 || standard.Discount != 10 {
		t.Fatalf("standard pricing: %+v", standard)
	}
}
