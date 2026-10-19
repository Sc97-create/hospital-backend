package diagnosticcentre

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"hospital-backend/internal/labService/diagnosticCentre/dto"
	wrapError "hospital-backend/shared/error"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DiagnosticCentreService struct {
	repo DiagnosticCentreRepository
}

func NewDiagnosticCentreService(repo DiagnosticCentreRepository) *DiagnosticCentreService {
	return &DiagnosticCentreService{repo: repo}
}

func (s *DiagnosticCentreService) Create(log *zap.Logger, req dto.CreateDiagnosticCentreRequest) (string, error) {
	log = ensureLog(log)
	req.OrganisationID = strings.TrimSpace(req.OrganisationID)
	req.Name = strings.TrimSpace(req.Name)
	req.Address = strings.TrimSpace(req.Address)
	req.ContactNumber = strings.TrimSpace(req.ContactNumber)
	req.Email = strings.TrimSpace(req.Email)
	if req.OrganisationID == "" || req.Name == "" {
		return "", wrapError.ErrInvalidRequest
	}

	row := DiagnosticCentre{
		ID:             uuid.New().String(),
		OrganisationID: req.OrganisationID,
		Name:           req.Name,
		Address:        req.Address,
		ContactNumber:  req.ContactNumber,
		Email:          req.Email,
		Status:         StatusActive,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.repo.Create(log, &row); err != nil {
		log.Error("diagnostic centre create failed",
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return "", wrapError.ErrDiagnosticCentreCreateFailed
	}
	log.Info("diagnostic centre create success",
		zap.String("diagnostic_centre_id", row.ID),
	)
	return row.ID, nil
}

func (s *DiagnosticCentreService) List(log *zap.Logger, req dto.ListDiagnosticCentresRequest) ([]dto.DiagnosticCentreResponse, int64, error) {
	log = ensureLog(log)
	req.OrganisationID = strings.TrimSpace(req.OrganisationID)
	req.Search = strings.TrimSpace(req.Search)
	req.Status = strings.TrimSpace(req.Status)
	if req.OrganisationID == "" {
		return nil, 0, wrapError.ErrInvalidRequest
	}
	if req.Status != "" && !isValidStatus(req.Status) {
		return nil, 0, wrapError.ErrInvalidRequest
	}
	req.DBLimit, req.DBOffset = s.parsePagination(req.Limit, req.PageNo)

	listQuery, listArgs := s.buildListQuery(req)
	rows, err := s.repo.FindMany(log, listQuery, listArgs...)
	if err != nil {
		log.Error("diagnostic centre list failed",
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrDiagnosticCentreFetchFailed
	}

	countQuery, countArgs := s.buildCountQuery(req)
	total, err := s.repo.Count(log, countQuery, countArgs...)
	if err != nil {
		log.Error("diagnostic centre list failed",
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrDiagnosticCentreFetchFailed
	}

	out := make([]dto.DiagnosticCentreResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, s.toResponse(row))
	}
	log.Info("diagnostic centre list success",
		zap.Int("result_count", len(out)),
		zap.Int64("total", total),
	)
	return out, total, nil
}

func (s *DiagnosticCentreService) Update(log *zap.Logger, id string, req dto.UpdateDiagnosticCentreRequest) error {
	log = ensureLog(log)
	id = strings.TrimSpace(id)
	req.Name = strings.TrimSpace(req.Name)
	req.Address = strings.TrimSpace(req.Address)
	req.ContactNumber = strings.TrimSpace(req.ContactNumber)
	req.Email = strings.TrimSpace(req.Email)
	if id == "" || req.Name == "" {
		return wrapError.ErrInvalidRequest
	}

	query := `
		UPDATE diagnostic_centres
		SET name = $1,
		    address = $2,
		    contact_number = $3,
		    email = $4,
		    updated_at = $5
		WHERE id = $6
	`
	err := s.repo.Exec(log, query, req.Name, req.Address, req.ContactNumber, req.Email, time.Now(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("diagnostic centre update failed", zap.String("diagnostic_centre_id", id), zap.String("reason", "not_found"))
			return wrapError.ErrDiagnosticCentreNotFound
		}
		log.Error("diagnostic centre update failed", zap.String("diagnostic_centre_id", id), zap.String("reason", "db_update"), zap.Error(err))
		return wrapError.ErrDiagnosticCentreUpdateFailed
	}
	log.Info("diagnostic centre update success", zap.String("diagnostic_centre_id", id))
	return nil
}

func (s *DiagnosticCentreService) UpdateStatus(log *zap.Logger, id, status string) error {
	log = ensureLog(log)
	id = strings.TrimSpace(id)
	status = strings.TrimSpace(strings.ToLower(status))
	if id == "" || !isValidStatus(status) {
		return wrapError.ErrInvalidRequest
	}

	query := `
		UPDATE diagnostic_centres
		SET status = $1,
		    updated_at = $2
		WHERE id = $3
	`
	err := s.repo.Exec(log, query, status, time.Now(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("diagnostic centre status update failed", zap.String("diagnostic_centre_id", id), zap.String("reason", "not_found"))
			return wrapError.ErrDiagnosticCentreNotFound
		}
		log.Error("diagnostic centre status update failed", zap.String("diagnostic_centre_id", id), zap.String("reason", "db_update"), zap.Error(err))
		return wrapError.ErrDiagnosticCentreUpdateFailed
	}
	log.Info("diagnostic centre status update success", zap.String("diagnostic_centre_id", id), zap.String("status", status))
	return nil
}

func (s *DiagnosticCentreService) buildListQuery(req dto.ListDiagnosticCentresRequest) (string, []interface{}) {
	query := `
		SELECT
			id,
			organisation_id,
			name,
			address,
			contact_number,
			email,
			status,
			created_at,
			updated_at
		FROM diagnostic_centres
		WHERE organisation_id = $1
	`
	args := []interface{}{req.OrganisationID}
	query, args, pos := s.appendFilters(query, req, args, 2)
	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", pos, pos+1)
	args = append(args, req.DBLimit, req.DBOffset)
	return query, args
}

func (s *DiagnosticCentreService) buildCountQuery(req dto.ListDiagnosticCentresRequest) (string, []interface{}) {
	query := `SELECT COUNT(*) FROM diagnostic_centres WHERE organisation_id = $1`
	args := []interface{}{req.OrganisationID}
	query, args, _ = s.appendFilters(query, req, args, 2)
	return query, args
}

func (s *DiagnosticCentreService) appendFilters(query string, req dto.ListDiagnosticCentresRequest, args []interface{}, pos int) (string, []interface{}, int) {
	if req.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", pos)
		args = append(args, req.Status)
		pos++
	}
	if req.Search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR address ILIKE $%d OR contact_number ILIKE $%d OR email ILIKE $%d)", pos, pos, pos, pos)
		args = append(args, "%"+req.Search+"%")
		pos++
	}
	return query, args, pos
}

func (s *DiagnosticCentreService) parsePagination(limit, pageNo float64) (int, int) {
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

func (s *DiagnosticCentreService) toResponse(row DiagnosticCentre) dto.DiagnosticCentreResponse {
	return dto.DiagnosticCentreResponse{
		ID:             row.ID,
		OrganisationID: row.OrganisationID,
		Name:           row.Name,
		Address:        row.Address,
		ContactNumber:  row.ContactNumber,
		Email:          row.Email,
		Status:         row.Status,
		CreatedAt:      row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      row.UpdatedAt.Format(time.RFC3339),
	}
}

func isValidStatus(status string) bool {
	return status == StatusActive || status == StatusInactive
}
