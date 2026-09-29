package organisations_test

import (
	"errors"
	"testing"
	"time"

	"hospital-backend/central/organisations"
	"hospital-backend/internal/testutil/servicetest"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testRepoDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&organisations.Organisation{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func sampleOrg(tenantID, id string) organisations.Organisation {
	now := time.Now()
	return organisations.Organisation{
		ID:               id,
		TenantID:         tenantID,
		LegalEntityName:  "Acme Health Pvt Ltd",
		OrganisationType: "hospital",
		FacilityName:     "Main Campus",
		RegistrationNo:   "REG-1",
		LicenseNumber:    "LIC-1",
		GSTIN:            "29AAAAA0000A1Z5",
		Address: organisations.Address{
			CountryID: "IN",
			State:     "KA",
			City:      "BLR",
			CreatedAt: now,
		},
		DataSharing: organisations.DataSharing{
			PatientLookup: true,
			LabReports:    false,
		},
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestRepoCreateAndGetByID(t *testing.T) {
	log := servicetest.NopLogger()
	getByIDQuery := `select id,tenant_id,is_primary,legal_entity_name,organisation_type,facility_name,registration_no,license_number,license_expiry,gstin,address,data_sharing,status,created_at,updated_at from organisations where id=$1`

	tests := []struct {
		name      string
		seed      organisations.Organisation
		lookupID  string
		wantErr   error
		wantFound bool
	}{
		{
			name:      "create and get success",
			seed:      sampleOrg("tenant-1", "org-1"),
			lookupID:  "org-1",
			wantFound: true,
		},
		{
			name:     "get missing",
			seed:     sampleOrg("tenant-1", "org-1"),
			lookupID: "org-missing",
			wantErr:  gorm.ErrRecordNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testRepoDB(t)
			repo := organisations.NewOrganisationRepo(db)
			if err := repo.Create(log, nil, tt.seed); err != nil {
				t.Fatalf("Create: %v", err)
			}
			got, err := repo.GetByID(log, getByIDQuery, tt.lookupID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByID: %v", err)
			}
			if !tt.wantFound {
				return
			}
			if got.ID != tt.seed.ID || got.TenantID != tt.seed.TenantID {
				t.Fatalf("got %+v", got)
			}
			if got.LegalEntityName != tt.seed.LegalEntityName {
				t.Fatalf("legal_entity_name: got %q", got.LegalEntityName)
			}
			if !got.DataSharing.PatientLookup || got.DataSharing.LabReports {
				t.Fatalf("data_sharing: %+v", got.DataSharing)
			}
		})
	}
}

func TestRepoListByTenantID(t *testing.T) {
	log := servicetest.NopLogger()
	listQuery := `select id,tenant_id,is_primary,legal_entity_name,organisation_type,facility_name,registration_no,license_number,license_expiry,gstin,address,data_sharing,status,created_at,updated_at from organisations where tenant_id=$1 order by created_at desc`

	tests := []struct {
		name      string
		seed      []organisations.Organisation
		tenantID  string
		wantCount int
	}{
		{
			name: "two for tenant, one other",
			seed: []organisations.Organisation{
				sampleOrg("tenant-1", "org-1"),
				sampleOrg("tenant-1", "org-2"),
				sampleOrg("tenant-2", "org-3"),
			},
			tenantID:  "tenant-1",
			wantCount: 2,
		},
		{
			name: "empty tenant list",
			seed: []organisations.Organisation{
				sampleOrg("tenant-1", "org-1"),
			},
			tenantID:  "tenant-none",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testRepoDB(t)
			repo := organisations.NewOrganisationRepo(db)
			for _, org := range tt.seed {
				if err := repo.Create(log, nil, org); err != nil {
					t.Fatalf("Create: %v", err)
				}
			}
			got, err := repo.ListByTenantID(log, listQuery, tt.tenantID)
			if err != nil {
				t.Fatalf("ListByTenantID: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Fatalf("count: got %d want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestRepoUpdate(t *testing.T) {
	log := servicetest.NopLogger()
	getByIDQuery := `select id,tenant_id,is_primary,legal_entity_name,organisation_type,facility_name,registration_no,license_number,license_expiry,gstin,address,data_sharing,status,created_at,updated_at from organisations where id=$1`

	tests := []struct {
		name    string
		seed    organisations.Organisation
		orgID   string
		query   string
		args    []any
		wantErr error
		check   func(*testing.T, organisations.Organisation)
	}{
		{
			name:  "update success",
			seed:  sampleOrg("tenant-1", "org-1"),
			orgID: "org-1",
			query: `update organisations set facility_name=$1, gstin=$2 where id=$3`,
			args:  []any{"West Campus", "29BBBBB0000B1Z5", "org-1"},
			check: func(t *testing.T, got organisations.Organisation) {
				t.Helper()
				if got.FacilityName != "West Campus" || got.GSTIN != "29BBBBB0000B1Z5" {
					t.Fatalf("unexpected update: %+v", got)
				}
			},
		},
		{
			name:    "update missing",
			seed:    sampleOrg("tenant-1", "org-1"),
			orgID:   "org-missing",
			query:   `update organisations set facility_name=$1 where id=$2`,
			args:    []any{"X", "org-missing"},
			wantErr: gorm.ErrRecordNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testRepoDB(t)
			repo := organisations.NewOrganisationRepo(db)
			if err := repo.Create(log, nil, tt.seed); err != nil {
				t.Fatalf("Create: %v", err)
			}
			err := repo.Update(log, tt.query, tt.args...)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			got, err := repo.GetByID(log, getByIDQuery, tt.orgID)
			if err != nil {
				t.Fatalf("GetByID: %v", err)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestRepoUpdateAddressByID(t *testing.T) {
	log := servicetest.NopLogger()
	getByIDQuery := `select id,tenant_id,is_primary,legal_entity_name,organisation_type,facility_name,registration_no,license_number,license_expiry,gstin,address,data_sharing,status,created_at,updated_at from organisations where id=$1`
	updateAddressQuery := `update organisations set address=$1, data_sharing=$2 where id=$3`

	tests := []struct {
		name    string
		seed    organisations.Organisation
		patch   *organisations.Organisation
		wantErr error
		check   func(*testing.T, organisations.Organisation)
	}{
		{
			name: "address success",
			seed: sampleOrg("tenant-1", "org-1"),
			patch: &organisations.Organisation{
				ID: "org-1",
				Address: organisations.Address{
					CountryID: "IN",
					State:     "MH",
					City:      "MUM",
				},
				DataSharing: organisations.DataSharing{
					PatientLookup: false,
					LabReports:    true,
				},
			},
			check: func(t *testing.T, got organisations.Organisation) {
				t.Helper()
				if got.Address.City != "MUM" || got.Address.State != "MH" {
					t.Fatalf("address: %+v", got.Address)
				}
				if got.DataSharing.PatientLookup || !got.DataSharing.LabReports {
					t.Fatalf("data_sharing: %+v", got.DataSharing)
				}
			},
		},
		{
			name: "address missing",
			seed: sampleOrg("tenant-1", "org-1"),
			patch: &organisations.Organisation{
				ID: "org-missing",
				Address: organisations.Address{
					City: "X",
				},
			},
			wantErr: gorm.ErrRecordNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testRepoDB(t)
			repo := organisations.NewOrganisationRepo(db)
			if err := repo.Create(log, nil, tt.seed); err != nil {
				t.Fatalf("Create: %v", err)
			}
			err := repo.UpdateAddressByID(log, updateAddressQuery, tt.patch.Address, tt.patch.DataSharing, tt.patch.ID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got err %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpdateAddressByID: %v", err)
			}
			got, err := repo.GetByID(log, getByIDQuery, tt.patch.ID)
			if err != nil {
				t.Fatalf("GetByID: %v", err)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
