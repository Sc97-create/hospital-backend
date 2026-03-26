package utils

import (
	"hospital-backend/internal/department"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultDeptAdmin           = "Administration"
	DefaultDeptEmergency       = "Emergency"
	DefaultDeptGeneralMedicine = "General Medicine"
	DefaultDeptCardiology      = "Cardiology"
	DefaultDeptRadiology       = "Radiology"
	DefaultDeptLaboratory      = "Laboratory"
	DefaultDeptPharmacy        = "Pharmacy"
	DefaultDeptNursing         = "Nursing"
)

type DeptArr []string

var DefaultDeptArr DeptArr = DeptArr{
	DefaultDeptAdmin,
	DefaultDeptEmergency,
	DefaultDeptGeneralMedicine,
	DefaultDeptCardiology,
	DefaultDeptRadiology,
	DefaultDeptLaboratory,
	DefaultDeptPharmacy,
	DefaultDeptNursing,
}

func AddDefaultDepartment(userID string) []department.Department {
	defaultDepartments := []department.Department{}
	for _, each := range DefaultDeptArr {
		defaultDepartments = append(defaultDepartments, department.Department{
			ID:        uuid.NewString(),
			Name:      each,
			CreatedBy: userID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UpdatedBy: userID,
			IsDefault: true,
		})
	}
	return defaultDepartments
}
