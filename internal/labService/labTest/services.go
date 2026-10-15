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

type LabOrderService struct {
	repo LabOrderRepository
}

func NewLabOrderService(repo LabOrderRepository) *LabOrderService {
	return &LabOrderService{repo: repo}
}

func (s *LabOrderService) Create(log *zap.Logger, req dto.CreateLabOrderRequest) (string, error) {
	log = ensureLog(log)
	if err := s.validateCreate(req); err != nil {
		return "", err
	}

	tests := s.buildOrderedTests(req.Tests)
	now := time.Now()
	row := LabOrders{
		ID:                 uuid.New().String(),
		OrganisationID:     strings.TrimSpace(req.OrganisationID),
		PatientID:          strings.TrimSpace(req.PatientID),
		DoctorID:           strings.TrimSpace(req.DoctorID),
		DiagnosticCentreID: strings.TrimSpace(req.DiagnosticCentreID),
		AppointmentID:      optionalID(req.AppointmentID),
		LabTests:           tests,
		Status:             OrderStatusPending,
		IsPayment:          false,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.repo.Create(log, &row); err != nil {
		log.Error("lab order create failed",
			zap.String("organisation_id", row.OrganisationID),
			zap.String("reason", "db_create"),
			zap.Error(err),
		)
		return "", wrapError.ErrLabOrderCreateFailed
	}
	log.Info("lab order create success",
		zap.String("organisation_id", row.OrganisationID),
		zap.String("lab_order_id", row.ID),
	)
	return row.ID, nil
}

func (s *LabOrderService) GetByID(log *zap.Logger, id string) (dto.LabOrderResponse, error) {
	log = ensureLog(log)
	id = strings.TrimSpace(id)
	if id == "" {
		return dto.LabOrderResponse{}, wrapError.ErrInvalidRequest
	}
	query, args := s.buildGetQuery(id)
	rows, err := s.repo.FindMany(log, query, args...)
	if err != nil {
		log.Error("lab order get failed", zap.String("lab_order_id", id), zap.String("reason", "db_read"), zap.Error(err))
		return dto.LabOrderResponse{}, wrapError.ErrLabOrderFetchFailed
	}
	if len(rows) == 0 {
		log.Warn("lab order get failed", zap.String("lab_order_id", id), zap.String("reason", "not_found"))
		return dto.LabOrderResponse{}, wrapError.ErrLabOrderNotFound
	}
	return s.toResponse(rows[0]), nil
}

func (s *LabOrderService) List(log *zap.Logger, req dto.ListLabOrdersRequest) ([]dto.LabOrderListItem, int64, error) {
	log = ensureLog(log)
	req.OrganisationID = strings.TrimSpace(req.OrganisationID)
	req.PatientID = strings.TrimSpace(req.PatientID)
	req.Status = strings.TrimSpace(strings.ToLower(req.Status))
	if req.OrganisationID == "" {
		return nil, 0, wrapError.ErrInvalidRequest
	}
	if req.Status != "" && !isValidOrderStatus(req.Status) {
		return nil, 0, wrapError.ErrInvalidRequest
	}
	req.DBLimit, req.DBOffset = s.parsePagination(req.Limit, req.PageNo)

	listQuery, listArgs := s.buildListQuery(req)
	rows, err := s.repo.FindMany(log, listQuery, listArgs...)
	if err != nil {
		log.Error("lab order list failed",
			zap.String("organisation_id", req.OrganisationID),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrLabOrderFetchFailed
	}

	countQuery, countArgs := s.buildCountQuery(req)
	total, err := s.repo.Count(log, countQuery, countArgs...)
	if err != nil {
		log.Error("lab order list failed",
			zap.String("organisation_id", req.OrganisationID),
			zap.String("reason", "db_count"),
			zap.Error(err),
		)
		return nil, 0, wrapError.ErrLabOrderFetchFailed
	}

	out := make([]dto.LabOrderListItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, s.toListItem(row))
	}
	log.Info("lab order list success",
		zap.String("organisation_id", req.OrganisationID),
		zap.Int("result_count", len(out)),
		zap.Int64("count", total),
	)
	return out, total, nil
}

// MarkIsPayment sets lab_orders.is_payment. When paid, also moves status pending → paid
// (does not overwrite completed/cancelled).
func (s *LabOrderService) MarkIsPayment(log *zap.Logger, orderID string, isPayment bool) error {
	log = ensureLog(log)
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return wrapError.ErrInvalidRequest
	}
	query := `
		UPDATE lab_orders
		SET is_payment = $1,
		    status = CASE
		        WHEN $1 = true AND status = $2 THEN $3
		        ELSE status
		    END,
		    updated_at = $4
		WHERE id = $5
	`
	err := s.repo.Exec(log, query, isPayment, OrderStatusPending, OrderStatusPaid, time.Now(), orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("lab order payment flag update failed",
				zap.String("lab_order_id", orderID),
				zap.Bool("is_payment", isPayment),
				zap.String("reason", "not_found"),
			)
			return wrapError.ErrLabOrderNotFound
		}
		log.Error("lab order payment flag update failed",
			zap.String("lab_order_id", orderID),
			zap.Bool("is_payment", isPayment),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrLabOrderUpdateFailed
	}
	log.Info("lab order payment flag update success",
		zap.String("lab_order_id", orderID),
		zap.Bool("is_payment", isPayment),
	)
	return nil
}

func (s *LabOrderService) UpdateStatus(log *zap.Logger, id, status string) error {
	log = ensureLog(log)
	id = strings.TrimSpace(id)
	status = strings.TrimSpace(strings.ToLower(status))
	if id == "" || !isValidOrderStatus(status) {
		return wrapError.ErrInvalidRequest
	}

	query := `
		UPDATE lab_orders
		SET status = $1,
		    updated_at = $2
		WHERE id = $3
	`
	err := s.repo.Exec(log, query, status, time.Now(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("lab order status update failed", zap.String("lab_order_id", id), zap.String("reason", "not_found"))
			return wrapError.ErrLabOrderNotFound
		}
		log.Error("lab order status update failed", zap.String("lab_order_id", id), zap.String("reason", "db_update"), zap.Error(err))
		return wrapError.ErrLabOrderUpdateFailed
	}
	log.Info("lab order status update success", zap.String("lab_order_id", id), zap.String("status", status))
	return nil
}

func (s *LabOrderService) UpdateTestStatus(log *zap.Logger, orderID, testID, status string) error {
	return s.UpdateTestsStatus(log, orderID, []string{testID}, status)
}

func (s *LabOrderService) UpdateTestsStatus(log *zap.Logger, orderID string, testIDs []string, status string) error {
	log = ensureLog(log)
	orderID = strings.TrimSpace(orderID)
	status = strings.TrimSpace(strings.ToLower(status))
	ids := normalizeTestIDs(testIDs)
	if orderID == "" || len(ids) == 0 || !isValidTestStatus(status) {
		return wrapError.ErrInvalidRequest
	}

	row, err := s.repo.FindByID(log, orderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("lab order test status update failed",
				zap.String("lab_order_id", orderID),
				zap.Strings("test_ids", ids),
				zap.String("reason", "not_found"),
			)
			return wrapError.ErrLabOrderNotFound
		}
		log.Error("lab order test status update failed",
			zap.String("lab_order_id", orderID),
			zap.Strings("test_ids", ids),
			zap.String("reason", "db_read"),
			zap.Error(err),
		)
		return wrapError.ErrLabOrderFetchFailed
	}

	if err := s.validateTestIDsBelongToOrder(row, ids); err != nil {
		log.Warn("lab order test status update failed",
			zap.String("lab_order_id", orderID),
			zap.Strings("test_ids", ids),
			zap.String("reason", "test_id_mismatch"),
		)
		return err
	}
	if err := s.applyTestsStatus(&row, ids, status); err != nil {
		log.Warn("lab order test status update failed",
			zap.String("lab_order_id", orderID),
			zap.Strings("test_ids", ids),
			zap.String("reason", "apply_status"),
		)
		return err
	}
	row.UpdatedAt = time.Now()
	if err := s.repo.Save(log, &row); err != nil {
		log.Error("lab order test status update failed",
			zap.String("lab_order_id", orderID),
			zap.Strings("test_ids", ids),
			zap.String("reason", "db_update"),
			zap.Error(err),
		)
		return wrapError.ErrLabOrderUpdateFailed
	}
	log.Info("lab order test status update success",
		zap.String("lab_order_id", orderID),
		zap.Strings("test_ids", ids),
		zap.String("status", status),
		zap.String("order_status", row.Status),
	)
	return nil
}

func (s *LabOrderService) validateCreate(req dto.CreateLabOrderRequest) error {
	if strings.TrimSpace(req.OrganisationID) == "" {
		return wrapError.ErrInvalidRequest
	}
	if strings.TrimSpace(req.PatientID) == "" {
		return wrapError.ErrInvalidRequest
	}
	if strings.TrimSpace(req.DoctorID) == "" {
		return wrapError.ErrInvalidRequest
	}
	if strings.TrimSpace(req.DiagnosticCentreID) == "" {
		return wrapError.ErrInvalidRequest
	}
	if len(req.Tests) == 0 {
		return wrapError.ErrInvalidRequest
	}
	for _, t := range req.Tests {
		if strings.TrimSpace(t.Name) == "" {
			return wrapError.ErrInvalidRequest
		}
	}
	return nil
}

func (s *LabOrderService) buildOrderedTests(inputs []dto.CreateOrderedTestInput) OrderedLabTestList {
	out := make(OrderedLabTestList, 0, len(inputs))
	for _, t := range inputs {
		out = append(out, OrderedLabTest{
			ID:      uuid.New().String(),
			Name:    strings.TrimSpace(t.Name),
			SubTest: strings.TrimSpace(t.SubTest),
			Status:  TestStatusPending,
		})
	}
	return out
}

func (s *LabOrderService) applyTestStatus(row *LabOrders, testID, status string) error {
	return s.applyTestsStatus(row, []string{testID}, status)
}

// validateTestIDsBelongToOrder ensures every payload test_id exists on the fetched lab order.
func (s *LabOrderService) validateTestIDsBelongToOrder(row LabOrders, testIDs []string) error {
	owned := make(map[string]struct{}, len(row.LabTests))
	for _, t := range row.LabTests {
		owned[t.ID] = struct{}{}
	}
	for _, id := range testIDs {
		if _, ok := owned[id]; !ok {
			return wrapError.ErrInvalidRequest
		}
	}
	return nil
}

func (s *LabOrderService) applyTestsStatus(row *LabOrders, testIDs []string, status string) error {
	wanted := make(map[string]struct{}, len(testIDs))
	for _, id := range testIDs {
		wanted[id] = struct{}{}
	}
	for i := range row.LabTests {
		if _, ok := wanted[row.LabTests[i].ID]; !ok {
			continue
		}
		row.LabTests[i].Status = status
	}
	if allTestsCompleted(row.LabTests) {
		row.Status = OrderStatusCompleted
	}
	return nil
}

func normalizeTestIDs(testIDs []string) []string {
	if len(testIDs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(testIDs))
	out := make([]string, 0, len(testIDs))
	for _, raw := range testIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func allTestsCompleted(tests OrderedLabTestList) bool {
	if len(tests) == 0 {
		return false
	}
	for _, t := range tests {
		if t.Status != TestStatusCompleted {
			return false
		}
	}
	return true
}

func (s *LabOrderService) buildGetQuery(id string) (string, []interface{}) {
	query := `
		SELECT
			lo.id,
			lo.organisation_id,
			lo.patient_id,
			COALESCE(p.name, '') AS patient_name,
			COALESCE(p.uh_id, '') AS patient_code,
			COALESCE(p.age, 0) AS patient_age,
			lo.doctor_id,
			TRIM(CONCAT(COALESCE(u.first_name, ''), ' ', COALESCE(u.last_name, ''))) AS doctor_name,
			lo.diagnostic_centre_id,
			lo.appointment_id,
			lo.lab_tests,
			lo.status,
			lo.is_payment,
			lo.created_at,
			lo.updated_at
		FROM lab_orders lo
		LEFT JOIN patients p ON p.id = lo.patient_id
		LEFT JOIN users u ON u.id = lo.doctor_id
		WHERE lo.id = $1
		LIMIT 1
	`
	return query, []interface{}{id}
}

func (s *LabOrderService) buildListQuery(req dto.ListLabOrdersRequest) (string, []interface{}) {
	query := `
		SELECT
			lo.id,
			lo.organisation_id,
			lo.patient_id,
			COALESCE(p.name, '') AS patient_name,
			lo.doctor_id,
			TRIM(CONCAT(COALESCE(u.first_name, ''), ' ', COALESCE(u.last_name, ''))) AS doctor_name,
			lo.diagnostic_centre_id,
			lo.appointment_id,
			lo.lab_tests,
			lo.status,
			lo.is_payment,
			lo.created_at,
			lo.updated_at
		FROM lab_orders lo
		LEFT JOIN patients p ON p.id = lo.patient_id
		LEFT JOIN users u ON u.id = lo.doctor_id
		WHERE lo.organisation_id = $1
	`
	args := []interface{}{req.OrganisationID}
	query, args, pos := s.appendFilters(query, req, args, 2)
	query += " ORDER BY lo.created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", pos, pos+1)
	args = append(args, req.DBLimit, req.DBOffset)
	return query, args
}

func (s *LabOrderService) buildCountQuery(req dto.ListLabOrdersRequest) (string, []interface{}) {
	query := `SELECT COUNT(*) FROM lab_orders lo WHERE lo.organisation_id = $1`
	args := []interface{}{req.OrganisationID}
	query, args, _ = s.appendFilters(query, req, args, 2)
	return query, args
}

func (s *LabOrderService) appendFilters(query string, req dto.ListLabOrdersRequest, args []interface{}, pos int) (string, []interface{}, int) {
	if req.PatientID != "" {
		query += fmt.Sprintf(" AND lo.patient_id = $%d", pos)
		args = append(args, req.PatientID)
		pos++
	}
	if req.Status != "" {
		query += fmt.Sprintf(" AND lo.status = $%d", pos)
		args = append(args, req.Status)
		pos++
	}
	return query, args, pos
}

func (s *LabOrderService) parsePagination(limit, pageNo float64) (int, int) {
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

func (s *LabOrderService) toListItem(row LabOrderListRow) dto.LabOrderListItem {
	return dto.LabOrderListItem{
		ID:                 row.ID,
		OrganisationID:     row.OrganisationID,
		PatientID:          row.PatientID,
		PatientName:        strings.TrimSpace(row.PatientName),
		DoctorID:           row.DoctorID,
		DoctorName:         strings.TrimSpace(row.DoctorName),
		DiagnosticCentreID: row.DiagnosticCentreID,
		AppointmentID:      row.AppointmentID,
		TestsCount:         len(row.LabTests),
		Status:             row.Status,
		IsPayment:          row.IsPayment,
		ReceivedAt:         row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          row.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *LabOrderService) toResponse(row LabOrderListRow) dto.LabOrderResponse {
	tests := make([]dto.OrderedTestResponse, 0, len(row.LabTests))
	for _, t := range row.LabTests {
		tests = append(tests, dto.OrderedTestResponse{
			ID:      t.ID,
			Name:    t.Name,
			SubTest: t.SubTest,
			Status:  t.Status,
		})
	}
	return dto.LabOrderResponse{
		ID:             row.ID,
		OrganisationID: row.OrganisationID,
		Patient: dto.LabOrderPatientDetails{
			PatientID:   row.PatientID,
			PatientName: strings.TrimSpace(row.PatientName),
			PatientCode: strings.TrimSpace(row.PatientCode),
			Age:         row.PatientAge,
		},
		Doctor: dto.LabOrderDoctorDetails{
			DoctorID:   row.DoctorID,
			DoctorName: strings.TrimSpace(row.DoctorName),
		},
		DiagnosticCentreID: row.DiagnosticCentreID,
		AppointmentID:      row.AppointmentID,
		Tests:              tests,
		TestsCount:         len(tests),
		Status:             row.Status,
		IsPayment:          row.IsPayment,
		ReceivedAt:         row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          row.UpdatedAt.Format(time.RFC3339),
	}
}

func optionalID(raw string) *string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil
	}
	return &v
}

func isValidOrderStatus(status string) bool {
	return status == OrderStatusPending ||
		status == OrderStatusPaid ||
		status == OrderStatusCompleted ||
		status == OrderStatusCancelled
}

func isValidTestStatus(status string) bool {
	return status == TestStatusPending || status == TestStatusCompleted || status == TestStatusCancelled
}
