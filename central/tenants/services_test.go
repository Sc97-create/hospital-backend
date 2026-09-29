package tenants_test

import (
	"errors"
	"testing"

	"hospital-backend/central/customers"
	"hospital-backend/central/organisations"
	"hospital-backend/central/tenants"
	dto "hospital-backend/central/tenants/dto"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	orgdto "hospital-backend/central/organisations/dto"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubTenantRepo struct {
	createErr error
	created   *tenants.Tenant
	byID      tenants.Tenant
	byIDErr   error
	statusID  string
	status    string
	updates   map[string]interface{}
	updateID  string
}

func (s *stubTenantRepo) Create(_ *zap.Logger, _ *gorm.DB, tenant tenants.Tenant) error {
	if s.createErr != nil {
		return s.createErr
	}
	cp := tenant
	s.created = &cp
	return nil
}

func (s *stubTenantRepo) GetByID(_ *zap.Logger, _ string) (tenants.Tenant, error) {
	if s.byIDErr != nil {
		return tenants.Tenant{}, s.byIDErr
	}
	return s.byID, nil
}

func (s *stubTenantRepo) UpdateStatusByID(_ *zap.Logger, _ *gorm.DB, tenantID, status string) error {
	s.statusID = tenantID
	s.status = status
	return nil
}

func (s *stubTenantRepo) UpdateByID(_ *zap.Logger, _ *gorm.DB, tenantID string, updates map[string]interface{}) error {
	s.updateID = tenantID
	s.updates = updates
	if name, ok := updates["name"].(string); ok {
		s.byID.Name = name
	}
	if status, ok := updates["status"].(string); ok {
		s.byID.Status = status
		s.status = status
	}
	return nil
}

type stubOrgProvisioner struct {
	hasOrg     bool
	hasErr     error
	addErr     error
	added      *orgdto.OrganisationPayload
	orgID      string
	primary    organisations.Organisation
	primaryErr error
	byID       organisations.Organisation
	byIDErr    error
	updateErr  error
	updatedID  string
	updated    *orgdto.OrganisationPayload
}

func (s *stubOrgProvisioner) HasOrganisationsForTenant(_ *zap.Logger, _ *gorm.DB, _ string) (bool, error) {
	if s.hasErr != nil {
		return false, s.hasErr
	}
	return s.hasOrg, nil
}

func (s *stubOrgProvisioner) AddOrganisationTx(_ *zap.Logger, _ *gorm.DB, payload orgdto.OrganisationPayload) (string, error) {
	if s.addErr != nil {
		return "", s.addErr
	}
	cp := payload
	s.added = &cp
	if s.orgID == "" {
		return "org-1", nil
	}
	return s.orgID, nil
}

func (s *stubOrgProvisioner) GetPrimaryByTenantID(_ *zap.Logger, _ string) (organisations.Organisation, error) {
	if s.primaryErr != nil {
		return organisations.Organisation{}, s.primaryErr
	}
	return s.primary, nil
}

func (s *stubOrgProvisioner) GetOrgByID(_ *zap.Logger, _ string) (organisations.Organisation, error) {
	if s.byIDErr != nil {
		return organisations.Organisation{}, s.byIDErr
	}
	if s.byID.ID != "" {
		return s.byID, nil
	}
	return s.primary, nil
}

func (s *stubOrgProvisioner) Update(_ *zap.Logger, organisationID string, payload orgdto.OrganisationPayload) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updatedID = organisationID
	cp := payload
	s.updated = &cp
	return nil
}

type stubCustomerBinder struct {
	err              error
	boundCustomerID  string
	boundTenantID    string
}

func (s *stubCustomerBinder) BindTenantID(_ *zap.Logger, _ *gorm.DB, customerID, tenantID string) error {
	if s.err != nil {
		return s.err
	}
	s.boundCustomerID = customerID
	s.boundTenantID = tenantID
	return nil
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func validPayload() dto.CreateTenantPayload {
	return dto.CreateTenantPayload{
		LegalEntityName: "Acme Health Pvt Ltd",
		FacilityName:    "Acme Main Campus",
		HospitalType:    "hospital",
		FacilityAddress: dto.FacilityAddress{
			Address1: "12 MG Road",
			Address2: "Near Park",
			City:     "Bengaluru",
			State:    "KA",
		},
	}
}

func TestServiceCreateTenant(t *testing.T) {
	log := servicetest.NopLogger()

	tests := []struct {
		name       string
		customerID string
		payload    dto.CreateTenantPayload
		repo       *stubTenantRepo
		org        *stubOrgProvisioner
		binder     *stubCustomerBinder
		wantErr    error
	}{
		{
			name:       "missing customer_id",
			customerID: "",
			payload:    validPayload(),
			wantErr:    wrapError.ErrInvalidRequest,
		},
		{
			name:       "missing legal_entity_name",
			customerID: "cust-1",
			payload:    dto.CreateTenantPayload{FacilityName: "F", HospitalType: "hospital", FacilityAddress: dto.FacilityAddress{Address1: "A", City: "C", State: "S"}},
			wantErr:    wrapError.ErrInvalidRequest,
		},
		{
			name:       "invalid hospital_type",
			customerID: "cust-1",
			payload:    dto.CreateTenantPayload{LegalEntityName: "L", FacilityName: "F", HospitalType: "lab", FacilityAddress: dto.FacilityAddress{Address1: "A", City: "C", State: "S"}},
			wantErr:    wrapError.ErrInvalidRequest,
		},
		{
			name:       "tenant repo create fails",
			customerID: "cust-1",
			payload:    validPayload(),
			repo:       &stubTenantRepo{createErr: errors.New("db")},
			org:        &stubOrgProvisioner{},
			binder:     &stubCustomerBinder{},
			wantErr:    wrapError.ErrTenantCreateFailed,
		},
		{
			name:       "organisation already exists",
			customerID: "cust-1",
			payload:    validPayload(),
			repo:       &stubTenantRepo{},
			org:        &stubOrgProvisioner{hasOrg: true},
			binder:     &stubCustomerBinder{},
			wantErr:    wrapError.ErrTenantAlreadyHasOrganisation,
		},
		{
			name:       "organisation create fails",
			customerID: "cust-1",
			payload:    validPayload(),
			repo:       &stubTenantRepo{},
			org:        &stubOrgProvisioner{addErr: wrapError.ErrOrganisationCreateFailed},
			binder:     &stubCustomerBinder{},
			wantErr:    wrapError.ErrOrganisationCreateFailed,
		},
		{
			name:       "customer not found",
			customerID: "missing-customer",
			payload:    validPayload(),
			repo:       &stubTenantRepo{},
			org:        &stubOrgProvisioner{orgID: "org-99"},
			binder:     &stubCustomerBinder{err: customers.ErrCustomerNotFound},
			wantErr:    wrapError.ErrCustomerNotFound,
		},
		{
			name:       "success creates primary organisation and binds customer",
			customerID: "cust-1",
			payload:    validPayload(),
			repo:       &stubTenantRepo{},
			org:        &stubOrgProvisioner{orgID: "org-99"},
			binder:     &stubCustomerBinder{},
			wantErr:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repo
			if repo == nil {
				repo = &stubTenantRepo{}
			}
			org := tt.org
			if org == nil {
				org = &stubOrgProvisioner{}
			}
			binder := tt.binder
			if binder == nil {
				binder = &stubCustomerBinder{}
			}
			svc := tenants.NewTenantService(openTestDB(t), repo, org, binder)
			got, err := svc.CreateTenant(log, tt.customerID, tt.payload)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err=%v want=%v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got.TenantID == "" || got.OrganisationID != "org-99" {
				t.Fatalf("unexpected result: %+v", got)
			}
			if repo.created == nil || repo.created.Status != "pending" {
				t.Fatalf("expected pending tenant, got %+v", repo.created)
			}
			if org.added == nil || !org.added.IsPrimary {
				t.Fatalf("expected primary org payload, got %+v", org.added)
			}
			if binder.boundCustomerID != "cust-1" || binder.boundTenantID != got.TenantID {
				t.Fatalf("customer bind mismatch: customer=%q tenant=%q", binder.boundCustomerID, binder.boundTenantID)
			}
		})
	}
}

func TestGetTenantByID(t *testing.T) {
	log := servicetest.NopLogger()
	repo := &stubTenantRepo{byID: tenants.Tenant{ID: "tenant-1", Name: "Acme", Status: "active"}}
	org := &stubOrgProvisioner{primary: organisations.Organisation{
		ID: "org-1", TenantID: "tenant-1", IsPrimary: true, LegalEntityName: "Acme", FacilityName: "Main",
	}}
	svc := tenants.NewTenantService(openTestDB(t), repo, org, &stubCustomerBinder{})

	got, err := svc.GetTenantByID(log, "tenant-1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.Tenant.ID != "tenant-1" || got.Organisation.ID != "org-1" || !got.Organisation.IsPrimary {
		t.Fatalf("got=%+v", got)
	}
}

func TestUpdateTenantOrg(t *testing.T) {
	log := servicetest.NopLogger()
	repo := &stubTenantRepo{byID: tenants.Tenant{ID: "tenant-1", Name: "Old", Status: "pending"}}
	org := &stubOrgProvisioner{byID: organisations.Organisation{
		ID: "org-1", TenantID: "tenant-1", IsPrimary: true, LegalEntityName: "Old", FacilityName: "Old Facility",
	}}
	svc := tenants.NewTenantService(openTestDB(t), repo, org, &stubCustomerBinder{})

	got, err := svc.UpdateTenantOrg(log, dto.UpdateTenantOrgPayload{
		OrganisationID:  "org-1",
		LegalEntityName: "New Name",
		FacilityName:    "New Facility",
		HospitalType:    "clinic",
		FacilityAddress: dto.FacilityAddress{Address1: "1 Main St", City: "Pune", State: "MH"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if repo.updates["name"] != "New Name" {
		t.Fatalf("tenant updates=%v", repo.updates)
	}
	if org.updated == nil || org.updated.FacilityName != "New Facility" || org.updated.OrganisationType != "clinic" {
		t.Fatalf("org updated=%+v", org.updated)
	}
	if got.Tenant.Name != "New Name" {
		t.Fatalf("result tenant=%+v", got.Tenant)
	}
}
