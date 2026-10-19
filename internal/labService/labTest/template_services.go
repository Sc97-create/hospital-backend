package labservice

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"hospital-backend/internal/labService/labTest/dto"
	wrapError "hospital-backend/shared/error"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type LabTestTemplateService struct {
	repo LabTestTemplateRepository
}

func NewLabTestTemplateService(repo LabTestTemplateRepository) *LabTestTemplateService {
	return &LabTestTemplateService{repo: repo}
}

// ListDefaults returns the built-in catalog plus organisation-built templates.
func (s *LabTestTemplateService) ListDefaults(log *zap.Logger, organisationID string) (dto.DefaultsListResponse, error) {
	log = ensureLog(log)
	organisationID = strings.TrimSpace(organisationID)
	if organisationID == "" {
		return dto.DefaultsListResponse{}, wrapError.ErrInvalidRequest
	}

	catalog := DefaultLabTestList()
	defaults := make([]dto.LabTemplateResponse, 0, len(catalog))
	for _, t := range catalog {
		defaults = append(defaults, toTemplateResponse(t))
	}

	listQuery, listArgs := s.buildTemplateListQuery(dto.ListLabTestTemplatesRequest{
		OrganisationID: organisationID,
		Status:         StatusActive,
		DBLimit:        1000,
		DBOffset:       0,
	})
	rows, err := s.repo.FindMany(log, listQuery, listArgs...)
	if err != nil {
		log.Error("lab test defaults list failed",
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return dto.DefaultsListResponse{}, wrapError.ErrLabTestTemplateFetchFailed
	}

	templates := make([]dto.LabTestTemplateResponse, 0, len(rows))
	for _, row := range rows {
		templates = append(templates, s.toTemplateRowResponse(row))
	}

	log.Info("lab test defaults list success",
		zap.Int("defaults_count", len(defaults)),
		zap.Int("templates_count", len(templates)),
	)
	return dto.DefaultsListResponse{
		Defaults:  defaults,
		Templates: templates,
	}, nil
}

func (s *LabTestTemplateService) Create(log *zap.Logger, req dto.CreateLabTestTemplateRequest) (string, error) {
	log = ensureLog(log)
	req.OrganisationID = strings.TrimSpace(req.OrganisationID)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if req.OrganisationID == "" || req.CreatedBy == "" || req.Name == "" {
		return "", wrapError.ErrInvalidRequest
	}

	tests, err := s.resolveTemplateTests(req)
	if err != nil {
		return "", err
	}

	now := time.Now()
	row := LabTestTemplate{
		ID:             uuid.New().String(),
		Name:           req.Name,
		Description:    req.Description,
		Tests:          tests,
		Status:         StatusActive,
		OrganisationID: req.OrganisationID,
		CreatedBy:      req.CreatedBy,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.Create(log, &row); err != nil {
		log.Error("lab test template create failed",
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return "", wrapError.ErrLabTestTemplateCreateFailed
	}
	log.Info("lab test template create success",
		zap.String("lab_test_template_id", row.ID),
		zap.Int("tests_count", len(row.Tests)),
	)
	return row.ID, nil
}

func (s *LabTestTemplateService) GetByID(log *zap.Logger, id string) (dto.LabTestTemplateResponse, error) {
	log = ensureLog(log)
	id = strings.TrimSpace(id)
	if id == "" {
		return dto.LabTestTemplateResponse{}, wrapError.ErrInvalidRequest
	}
	row, err := s.repo.FindByID(log, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("lab test template get failed", zap.String("lab_test_template_id", id), zap.String("reason", "not_found"))
			return dto.LabTestTemplateResponse{}, wrapError.ErrLabTestTemplateNotFound
		}
		log.Error("lab test template get failed", zap.String("lab_test_template_id", id), zap.String("reason", "db_read"), zap.Error(err))
		return dto.LabTestTemplateResponse{}, wrapError.ErrLabTestTemplateFetchFailed
	}
	return s.toTemplateRowResponse(row), nil
}

func (s *LabTestTemplateService) List(log *zap.Logger, req dto.ListLabTestTemplatesRequest) ([]dto.LabTestTemplateResponse, int64, error) {
	log = ensureLog(log)
	req.OrganisationID = strings.TrimSpace(req.OrganisationID)
	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.OrganisationID == "" {
		return nil, 0, wrapError.ErrInvalidRequest
	}
	if req.Status != "" && req.Status != StatusActive && req.Status != StatusInactive {
		return nil, 0, wrapError.ErrInvalidRequest
	}
	req.DBLimit, req.DBOffset = s.parseTemplatePagination(req.Limit, req.PageNo)

	listQuery, listArgs := s.buildTemplateListQuery(req)
	rows, err := s.repo.FindMany(log, listQuery, listArgs...)
	if err != nil {
		log.Error("lab test template list failed",
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrLabTestTemplateFetchFailed
	}

	countQuery, countArgs := s.buildTemplateCountQuery(req)
	total, err := s.repo.Count(log, countQuery, countArgs...)
	if err != nil {
		log.Error("lab test template list failed",
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrLabTestTemplateFetchFailed
	}

	out := make([]dto.LabTestTemplateResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, s.toTemplateRowResponse(row))
	}
	log.Info("lab test template list success",
		zap.Int("result_count", len(out)),
		zap.Int64("count", total),
	)
	return out, total, nil
}

func (s *LabTestTemplateService) resolveTemplateTests(req dto.CreateLabTestTemplateRequest) (LabTemplateList, error) {
	if len(req.TestCodes) > 0 {
		return s.testsFromCodes(req.TestCodes)
	}
	if len(req.Tests) > 0 {
		return s.testsFromPayload(req.Tests)
	}
	return nil, wrapError.ErrInvalidRequest
}

func (s *LabTestTemplateService) testsFromCodes(codes []string) (LabTemplateList, error) {
	out := make(LabTemplateList, 0, len(codes))
	for _, raw := range codes {
		code := strings.TrimSpace(raw)
		if code == "" {
			return nil, wrapError.ErrInvalidRequest
		}
		t, ok := DefaultLabTests[code]
		if !ok {
			return nil, wrapError.ErrInvalidRequest
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *LabTestTemplateService) testsFromPayload(items []dto.LabTemplatePayload) (LabTemplateList, error) {
	out := make(LabTemplateList, 0, len(items))
	for _, item := range items {
		code := strings.TrimSpace(item.TestCode)
		name := strings.TrimSpace(item.TestName)
		if code == "" || name == "" {
			return nil, wrapError.ErrInvalidRequest
		}
		specimens := item.Specimens
		if specimens == nil {
			specimens = []string{}
		}
		out = append(out, LabTemplate{
			TestCode:    code,
			TestName:    name,
			Specimens:   specimens,
			Type:        strings.TrimSpace(item.Type),
			TATMinutes:  item.TATMinutes,
			Description: strings.TrimSpace(item.Description),
		})
	}
	return out, nil
}

func (s *LabTestTemplateService) buildTemplateListQuery(req dto.ListLabTestTemplatesRequest) (string, []interface{}) {
	query := `
		SELECT
			id,
			name,
			description,
			tests,
			status,
			organisation_id,
			created_by,
			created_at,
			updated_at
		FROM lab_test_templates
		WHERE organisation_id = $1
	`
	args := []interface{}{req.OrganisationID}
	pos := 2
	if req.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", pos)
		args = append(args, req.Status)
		pos++
	}
	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", pos, pos+1)
	args = append(args, req.DBLimit, req.DBOffset)
	return query, args
}

func (s *LabTestTemplateService) buildTemplateCountQuery(req dto.ListLabTestTemplatesRequest) (string, []interface{}) {
	query := `SELECT COUNT(*) FROM lab_test_templates WHERE organisation_id = $1`
	args := []interface{}{req.OrganisationID}
	if req.Status != "" {
		query += " AND status = $2"
		args = append(args, req.Status)
	}
	return query, args
}

func (s *LabTestTemplateService) parseTemplatePagination(limit, pageNo float64) (int, int) {
	numLimit := int(limit)
	if numLimit <= 0 {
		numLimit = 10
	}
	numPage := int(pageNo)
	if numPage <= 0 {
		numPage = 1
	}
	return numLimit, (numPage - 1) * numLimit
}

func (s *LabTestTemplateService) toTemplateRowResponse(row LabTestTemplate) dto.LabTestTemplateResponse {
	tests := make([]dto.LabTemplateResponse, 0, len(row.Tests))
	for _, t := range row.Tests {
		tests = append(tests, toTemplateResponse(t))
	}
	return dto.LabTestTemplateResponse{
		ID:             row.ID,
		Name:           row.Name,
		Description:    row.Description,
		Tests:          tests,
		TestsCount:     len(tests),
		Status:         row.Status,
		OrganisationID: row.OrganisationID,
		CreatedBy:      row.CreatedBy,
		CreatedAt:      row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      row.UpdatedAt.Format(time.RFC3339),
	}
}

func toTemplateResponse(t LabTemplate) dto.LabTemplateResponse {
	specimens := t.Specimens
	if specimens == nil {
		specimens = []string{}
	}
	return dto.LabTemplateResponse{
		TestCode:    t.TestCode,
		TestName:    t.TestName,
		Specimens:   specimens,
		Type:        t.Type,
		TATMinutes:  t.TATMinutes,
		Description: t.Description,
	}
}
