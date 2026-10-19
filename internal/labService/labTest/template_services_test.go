package labservice_test

import (
	"errors"
	"testing"
	"time"

	labservice "hospital-backend/internal/labService/labTest"
	"hospital-backend/internal/labService/labTest/dto"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type memoryTemplateRepo struct {
	rows map[string]labservice.LabTestTemplate
	err  error
}

func newMemoryTemplateRepo() *memoryTemplateRepo {
	return &memoryTemplateRepo{rows: map[string]labservice.LabTestTemplate{}}
}

func (m *memoryTemplateRepo) Create(_ *zap.Logger, template *labservice.LabTestTemplate) error {
	if m.err != nil {
		return m.err
	}
	m.rows[template.ID] = *template
	return nil
}

func (m *memoryTemplateRepo) FindByID(_ *zap.Logger, id string) (labservice.LabTestTemplate, error) {
	row, ok := m.rows[id]
	if !ok {
		return labservice.LabTestTemplate{}, gorm.ErrRecordNotFound
	}
	return row, nil
}

func (m *memoryTemplateRepo) FindMany(_ *zap.Logger, _ string, args ...any) ([]labservice.LabTestTemplate, error) {
	orgID, _ := args[0].(string)
	out := make([]labservice.LabTestTemplate, 0)
	for _, row := range m.rows {
		if row.OrganisationID == orgID {
			out = append(out, row)
		}
	}
	return out, nil
}

func (m *memoryTemplateRepo) Count(_ *zap.Logger, _ string, args ...any) (int64, error) {
	orgID, _ := args[0].(string)
	var total int64
	for _, row := range m.rows {
		if row.OrganisationID == orgID {
			total++
		}
	}
	return total, nil
}

func TestLabTestTemplateListDefaults(t *testing.T) {
	repo := newMemoryTemplateRepo()
	now := time.Now()
	repo.rows["tpl-1"] = labservice.LabTestTemplate{
		ID:             "tpl-1",
		Name:           "Diabetes Panel",
		OrganisationID: "org-1",
		CreatedBy:      "user-1",
		Status:         labservice.StatusActive,
		Tests:          labservice.LabTemplateList{labservice.DefaultLabTests["BIO-04"]},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repo.rows["tpl-other"] = labservice.LabTestTemplate{
		ID:             "tpl-other",
		Name:           "Other Org",
		OrganisationID: "org-2",
		CreatedBy:      "user-2",
		Status:         labservice.StatusActive,
		Tests:          labservice.LabTemplateList{labservice.DefaultLabTests["HEM-01"]},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	svc := labservice.NewLabTestTemplateService(repo)

	got, err := svc.ListDefaults(servicetest.NopLogger(), "org-1")
	if err != nil {
		t.Fatalf("list defaults failed: %v", err)
	}
	if len(got.Defaults) != 21 {
		t.Fatalf("expected 21 defaults, got %d", len(got.Defaults))
	}
	if got.Defaults[0].TestCode != "BIO-04" || got.Defaults[0].TATMinutes != 120 {
		t.Fatalf("unexpected first default: %+v", got.Defaults[0])
	}
	if len(got.Templates) != 1 || got.Templates[0].ID != "tpl-1" {
		t.Fatalf("expected org templates, got %+v", got.Templates)
	}

	if _, err := svc.ListDefaults(servicetest.NopLogger(), ""); !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}
}

func TestLabTestTemplateCreateFromCodes(t *testing.T) {
	repo := newMemoryTemplateRepo()
	svc := labservice.NewLabTestTemplateService(repo)

	id, err := svc.Create(servicetest.NopLogger(), dto.CreateLabTestTemplateRequest{
		OrganisationID: "org-1",
		CreatedBy:      "user-1",
		Name:           "Diabetes Panel",
		TestCodes:      []string{"BIO-04", "BIO-12"},
	})
	if err != nil || id == "" {
		t.Fatalf("create failed: id=%q err=%v", id, err)
	}
	got, err := svc.GetByID(servicetest.NopLogger(), id)
	if err != nil || got.TestsCount != 2 || got.Tests[0].TestCode != "BIO-04" {
		t.Fatalf("get failed: %+v err=%v", got, err)
	}
}

func TestLabTestTemplateCreateValidation(t *testing.T) {
	svc := labservice.NewLabTestTemplateService(newMemoryTemplateRepo())
	_, err := svc.Create(servicetest.NopLogger(), dto.CreateLabTestTemplateRequest{
		OrganisationID: "org-1",
		CreatedBy:      "user-1",
		Name:           "Empty",
	})
	if !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected invalid request, got %v", err)
	}

	_, err = svc.Create(servicetest.NopLogger(), dto.CreateLabTestTemplateRequest{
		OrganisationID: "org-1",
		CreatedBy:      "user-1",
		Name:           "Bad",
		TestCodes:      []string{"UNKNOWN"},
	})
	if !errors.Is(err, wrapError.ErrInvalidRequest) {
		t.Fatalf("expected invalid code, got %v", err)
	}
}

func TestLabTestTemplateCreateRepoError(t *testing.T) {
	repo := newMemoryTemplateRepo()
	repo.err = errors.New("db")
	svc := labservice.NewLabTestTemplateService(repo)
	_, err := svc.Create(servicetest.NopLogger(), dto.CreateLabTestTemplateRequest{
		OrganisationID: "org-1",
		CreatedBy:      "user-1",
		Name:           "Panel",
		TestCodes:      []string{"HEM-01"},
	})
	if !errors.Is(err, wrapError.ErrLabTestTemplateCreateFailed) {
		t.Fatalf("expected create failed, got %v", err)
	}
}

func TestLabTestTemplateListAndGet(t *testing.T) {
	repo := newMemoryTemplateRepo()
	now := time.Now()
	repo.rows["tpl-1"] = labservice.LabTestTemplate{
		ID:             "tpl-1",
		Name:           "Panel",
		OrganisationID: "org-1",
		CreatedBy:      "user-1",
		Status:         labservice.StatusActive,
		Tests:          labservice.LabTemplateList{labservice.DefaultLabTests["HEM-01"]},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	svc := labservice.NewLabTestTemplateService(repo)

	rows, total, err := svc.List(servicetest.NopLogger(), dto.ListLabTestTemplatesRequest{
		OrganisationID: "org-1",
		Limit:          10,
		PageNo:         1,
	})
	if err != nil || total != 1 || len(rows) != 1 {
		t.Fatalf("list failed: rows=%+v total=%d err=%v", rows, total, err)
	}

	if _, err := svc.GetByID(servicetest.NopLogger(), "missing"); !errors.Is(err, wrapError.ErrLabTestTemplateNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}
