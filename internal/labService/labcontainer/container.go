package labcontainer

import (
	diagnosticcentre "hospital-backend/internal/labService/diagnosticCentre"
	labservice "hospital-backend/internal/labService/labTest"

	"gorm.io/gorm"
)

type LabContainer struct {
	DB                      *gorm.DB
	LabOrderService         *labservice.LabOrderService
	LabTestTemplateService  *labservice.LabTestTemplateService
	DiagnosticCentreService *diagnosticcentre.DiagnosticCentreService
}

func NewLabContainer(db *gorm.DB) *LabContainer {
	labOrderRepo := labservice.NewLabOrderRepo(db)
	labTemplateRepo := labservice.NewLabTestTemplateRepo(db)
	diagnosticCentreRepo := diagnosticcentre.NewDiagnosticCentreRepo(db)
	return &LabContainer{
		DB:                      db,
		LabOrderService:         labservice.NewLabOrderService(labOrderRepo),
		LabTestTemplateService:  labservice.NewLabTestTemplateService(labTemplateRepo),
		DiagnosticCentreService: diagnosticcentre.NewDiagnosticCentreService(diagnosticCentreRepo),
	}
}
