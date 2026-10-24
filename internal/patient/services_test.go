package patient_test

import (
	"errors"
	"strings"
	"testing"

	"hospital-backend/central/organisations"
	orgmocks "hospital-backend/central/organisations/mocks"
	"hospital-backend/internal/patient"
	"hospital-backend/internal/patient/dto"
	"hospital-backend/internal/patient/mocks"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func newPatientService(t *testing.T, repo patient.PatientRepository, org organisations.OrganisationServicer) *patient.PatientService {
	t.Helper()
	return patient.NewPatientService(repo, org, servicetest.NoopNotifier{})
}

func TestValidatePatient(t *testing.T) {
	svc := newPatientService(t, nil, nil)

	tests := []struct {
		name    string
		payload dto.PatientInfo
		wantErr bool
	}{
		{name: "missing name", payload: dto.PatientInfo{Gender: "male", Age: "30", Weight: "65"}, wantErr: true},
		{name: "missing gender", payload: dto.PatientInfo{Name: "Jane", Age: "30", Weight: "65"}, wantErr: true},
		{name: "negative age", payload: dto.PatientInfo{Name: "Jane", Gender: "female", Age: "-1", Weight: "65"}, wantErr: true},
		{name: "zero weight", payload: dto.PatientInfo{Name: "Jane", Gender: "female", Age: "30", Weight: "0"}, wantErr: true},
		{name: "success", payload: servicetest.ValidPatientInfo(), wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := svc.ValidatePatient(tt.payload)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestToPatientModel(t *testing.T) {
	svc := newPatientService(t, nil, nil)
	payload := servicetest.ValidPatientInfo()

	model, err := svc.ToPatientModel(30, 65, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model.ID == "" || model.UHID == "" {
		t.Fatal("expected generated id and uhid")
	}
	if model.Name != payload.Name {
		t.Fatalf("expected name %q, got %q", payload.Name, model.Name)
	}
	if model.Status != patient.StatusActive {
		t.Fatalf("expected active status, got %q", model.Status)
	}
}

func TestGetPageSkip(t *testing.T) {
	svc := newPatientService(t, nil, nil)

	limit, skip := svc.GetPageSkip("10", "1")
	if limit != 10 || skip != 0 {
		t.Fatalf("page 1: got limit=%d skip=%d", limit, skip)
	}

	limit, skip = svc.GetPageSkip("10", "3")
	if limit != 10 || skip != 20 {
		t.Fatalf("page 3: got limit=%d skip=%d", limit, skip)
	}
}

func TestFindOne(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*mocks.MockPatientRepository)
		wantErr  error
		wantName string
	}{
		{
			name: "not found",
			setup: func(m *mocks.MockPatientRepository) {
				m.EXPECT().ReadOne(gomock.Any(), "pat-1").Return(patient.Patient{}, gorm.ErrRecordNotFound)
			},
			wantErr: wrapError.ErrPatientNotFound,
		},
		{
			name: "db error",
			setup: func(m *mocks.MockPatientRepository) {
				m.EXPECT().ReadOne(gomock.Any(), "pat-1").Return(patient.Patient{}, errors.New("db down"))
			},
			wantErr: wrapError.ErrPatientFetchFailed,
		},
		{
			name: "success",
			setup: func(m *mocks.MockPatientRepository) {
				m.EXPECT().ReadOne(gomock.Any(), "pat-1").Return(patient.Patient{
					ID: "pat-1", Name: "Jane Doe", UHID: "CLI-123",
				}, nil)
			},
			wantName: "Jane Doe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockPatientRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := newPatientService(t, repo, nil)
			got, err := svc.FindOne(servicetest.NopLogger(), "pat-1")
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.PatientName != tt.wantName {
				t.Fatalf("expected name %q, got %q", tt.wantName, got.PatientName)
			}
		})
	}
}

func TestFindMany(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*mocks.MockPatientRepository)
		wantErr   error
		wantCount int
		wantTotal int64
	}{
		{
			name: "read error",
			setup: func(m *mocks.MockPatientRepository) {
				m.EXPECT().ReadMany(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("read fail"))
			},
			wantErr: wrapError.ErrPatientsFetchFailed,
		},
		{
			name: "count error",
			setup: func(m *mocks.MockPatientRepository) {
				m.EXPECT().ReadMany(gomock.Any(), gomock.Any(), gomock.Any()).Return([]patient.Patient{}, nil)
				m.EXPECT().Count(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(0), errors.New("count fail"))
			},
			wantErr: wrapError.ErrPatientsFetchFailed,
		},
		{
			name: "success",
			setup: func(m *mocks.MockPatientRepository) {
				m.EXPECT().ReadMany(gomock.Any(), gomock.Any(), gomock.Any()).Return([]patient.Patient{
					{ID: "pat-1", Name: "Jane"},
				}, nil)
				m.EXPECT().Count(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1), nil)
			},
			wantCount: 1,
			wantTotal: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockPatientRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := newPatientService(t, repo, nil)
			got, total, err := svc.FindMany(servicetest.NopLogger(), dto.PatientListReq{
				OrganisationID: "org-1",
				Limit:          10,
				PageNo:         1,
			})
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Fatalf("expected %d patients, got %d", tt.wantCount, len(got))
			}
			if total != tt.wantTotal {
				t.Fatalf("expected total %d, got %d", tt.wantTotal, total)
			}
		})
	}
}

func TestGetNotificationPatientByID(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mocks.MockPatientRepository)
		wantErr error
	}{
		{
			name: "repo error",
			setup: func(m *mocks.MockPatientRepository) {
				m.EXPECT().ReadOneWithOrganisationID(gomock.Any(), gomock.Any(), "pat-1").Return(nil, errors.New("db"))
			},
			wantErr: errors.New("db"),
		},
		{
			name: "empty result",
			setup: func(m *mocks.MockPatientRepository) {
				m.EXPECT().ReadOneWithOrganisationID(gomock.Any(), gomock.Any(), "pat-1").Return(map[string]interface{}{}, nil)
			},
			wantErr: wrapError.ErrPatientNotFound,
		},
		{
			name: "success",
			setup: func(m *mocks.MockPatientRepository) {
				m.EXPECT().ReadOneWithOrganisationID(gomock.Any(), gomock.Any(), "pat-1").Return(map[string]interface{}{
					"patient_id": "pat-1",
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockPatientRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := newPatientService(t, repo, nil)
			got, err := svc.GetNotificationPatientByID(servicetest.NopLogger(), "pat-1")
			if tt.wantErr != nil {
				if tt.wantErr == wrapError.ErrPatientNotFound {
					if !errors.Is(err, wrapError.ErrPatientNotFound) {
						t.Fatalf("expected ErrPatientNotFound, got %v", err)
					}
					return
				}
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) == 0 {
				t.Fatal("expected notification data")
			}
		})
	}
}

func TestCreatePatientSrv(t *testing.T) {
	valid := servicetest.ValidPatientInfo()

	tests := []struct {
		name    string
		payload dto.PatientInfo
		setup   func(*mocks.MockPatientRepository, *orgmocks.MockOrganisationServicer)
		wantErr error
	}{
		{
			name:    "org not found",
			payload: valid,
			setup: func(_ *mocks.MockPatientRepository, org *orgmocks.MockOrganisationServicer) {
				org.EXPECT().GetOrgByID(gomock.Any(), "org-1").Return(organisations.Organisation{}, wrapError.ErrOrganisationNotFound)
			},
			wantErr: wrapError.ErrOrganisationNotFound,
		},
		{
			name:    "validation fail",
			payload: dto.PatientInfo{OrganisationID: "org-1", Name: "", Gender: "male", Age: "30", Weight: "65"},
			setup: func(_ *mocks.MockPatientRepository, org *orgmocks.MockOrganisationServicer) {
				org.EXPECT().GetOrgByID(gomock.Any(), "org-1").Return(organisations.Organisation{ID: "org-1"}, nil)
			},
		},
		{
			name:    "duplicate patient",
			payload: valid,
			setup: func(repo *mocks.MockPatientRepository, org *orgmocks.MockOrganisationServicer) {
				org.EXPECT().GetOrgByID(gomock.Any(), "org-1").Return(organisations.Organisation{ID: "org-1", FacilityName: "City Hospital"}, nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("duplicate key"))
			},
			wantErr: wrapError.ErrPatientAlreadyExists,
		},
		{
			name:    "repo create error",
			payload: valid,
			setup: func(repo *mocks.MockPatientRepository, org *orgmocks.MockOrganisationServicer) {
				org.EXPECT().GetOrgByID(gomock.Any(), "org-1").Return(organisations.Organisation{ID: "org-1", FacilityName: "City Hospital"}, nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("db fail"))
			},
			wantErr: wrapError.ErrPatientCreateFailed,
		},
		{
			name:    "success",
			payload: valid,
			setup: func(repo *mocks.MockPatientRepository, org *orgmocks.MockOrganisationServicer) {
				org.EXPECT().GetOrgByID(gomock.Any(), "org-1").Return(organisations.Organisation{ID: "org-1", FacilityName: "City Hospital"}, nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockPatientRepository(ctrl)
			org := orgmocks.NewMockOrganisationServicer(ctrl)
			if tt.setup != nil {
				tt.setup(repo, org)
			}
			svc := newPatientService(t, repo, org)
			id, err := svc.CreatePatientSrv(servicetest.NopLogger(), tt.payload)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if tt.name == "validation fail" {
				if err == nil {
					t.Fatal("expected validation error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id == "" {
				t.Fatal("expected patient id")
			}
		})
	}
}

func TestUpdatePatient(t *testing.T) {
	existing := patient.Patient{
		ID: "pat-1", Name: "Jane", Age: 30, Weight: 65, Gender: "female",
		EmailID: "jane@example.com", MobileNumber: "9999999999", BloodGroup: "O+", Address: "old",
	}
	base := dto.UpdatePatientReq{PatientID: "pat-1", OrganisationID: "org-1", UpdatedBy: "user-2"}

	t.Run("missing updated_by", func(t *testing.T) {
		svc := newPatientService(t, nil, nil)
		req := base
		req.UpdatedBy = ""
		err := svc.Update(servicetest.NopLogger(), req)
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockPatientRepository(ctrl)
		repo.EXPECT().ReadOneByOrg(gomock.Any(), "pat-1", "org-1").Return(patient.Patient{}, gorm.ErrRecordNotFound)
		svc := newPatientService(t, repo, nil)
		err := svc.Update(servicetest.NopLogger(), base)
		if !errors.Is(err, wrapError.ErrPatientNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})

	t.Run("no profile changes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockPatientRepository(ctrl)
		repo.EXPECT().ReadOneByOrg(gomock.Any(), "pat-1", "org-1").Return(existing, nil)
		svc := newPatientService(t, repo, nil)
		req := base
		req.Name = "Jane"
		if err := svc.Update(servicetest.NopLogger(), req); err == nil {
			t.Fatal("expected no fields updated")
		}
	})

	t.Run("writes profile and audit columns", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockPatientRepository(ctrl)
		repo.EXPECT().ReadOneByOrg(gomock.Any(), "pat-1", "org-1").Return(existing, nil)
		repo.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ *zap.Logger, query string, args ...any) (int64, error) {
				if !strings.Contains(query, "name = $1") || !strings.Contains(query, "updated_by = $2") || !strings.Contains(query, "updated_at = NOW()") {
					t.Fatalf("unexpected query: %s", query)
				}
				if len(args) != 4 || args[0] != "Janet" || args[1] != "user-2" || args[2] != "pat-1" || args[3] != "org-1" {
					t.Fatalf("unexpected args: %#v", args)
				}
				return int64(1), nil
			},
		)
		svc := newPatientService(t, repo, nil)
		req := base
		req.Name = "Janet"
		if err := svc.Update(servicetest.NopLogger(), req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestDisablePatient(t *testing.T) {
	t.Run("sets status disable", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockPatientRepository(ctrl)
		repo.EXPECT().ReadOneByOrg(gomock.Any(), "pat-1", "org-1").Return(patient.Patient{ID: "pat-1", Status: patient.StatusActive}, nil)
		repo.EXPECT().Exec(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ *zap.Logger, query string, args ...any) (int64, error) {
				if !strings.Contains(query, "status = $1") || !strings.Contains(query, "updated_by = $2") || !strings.Contains(query, "updated_at = NOW()") {
					t.Fatalf("unexpected query: %s", query)
				}
				if len(args) != 4 || args[0] != "disable" || args[1] != "user-2" || args[2] != "pat-1" || args[3] != "org-1" {
					t.Fatalf("unexpected args: %#v", args)
				}
				return int64(1), nil
			},
		)
		svc := newPatientService(t, repo, nil)
		if err := svc.Disable(servicetest.NopLogger(), "pat-1", "org-1", "user-2"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("already disabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockPatientRepository(ctrl)
		repo.EXPECT().ReadOneByOrg(gomock.Any(), "pat-1", "org-1").Return(patient.Patient{Status: patient.StatusDisable}, nil)
		svc := newPatientService(t, repo, nil)
		if err := svc.Disable(servicetest.NopLogger(), "pat-1", "org-1", "user-2"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
