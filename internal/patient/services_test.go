package patient_test

import (
	"errors"
	"testing"

	"hospital-backend/internal/organisation"
	orgmocks "hospital-backend/internal/organisation/mocks"
	"hospital-backend/internal/patient"
	"hospital-backend/internal/patient/dto"
	"hospital-backend/internal/patient/mocks"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func newPatientService(t *testing.T, repo patient.PatientRepository, org organisation.OrganisationServicer) *patient.PatientService {
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
				org.EXPECT().GetOrgByID(gomock.Any(), "org-1").Return(organisation.Organisation{}, wrapError.ErrOrganisationNotFound)
			},
			wantErr: wrapError.ErrOrganisationNotFound,
		},
		{
			name:    "validation fail",
			payload: dto.PatientInfo{OrganisationID: "org-1", Name: "", Gender: "male", Age: "30", Weight: "65"},
			setup: func(_ *mocks.MockPatientRepository, org *orgmocks.MockOrganisationServicer) {
				org.EXPECT().GetOrgByID(gomock.Any(), "org-1").Return(organisation.Organisation{ID: "org-1"}, nil)
			},
		},
		{
			name:    "duplicate patient",
			payload: valid,
			setup: func(repo *mocks.MockPatientRepository, org *orgmocks.MockOrganisationServicer) {
				org.EXPECT().GetOrgByID(gomock.Any(), "org-1").Return(organisation.Organisation{ID: "org-1", OrganisationName: "City Hospital"}, nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("duplicate key"))
			},
			wantErr: wrapError.ErrPatientAlreadyExists,
		},
		{
			name:    "repo create error",
			payload: valid,
			setup: func(repo *mocks.MockPatientRepository, org *orgmocks.MockOrganisationServicer) {
				org.EXPECT().GetOrgByID(gomock.Any(), "org-1").Return(organisation.Organisation{ID: "org-1", OrganisationName: "City Hospital"}, nil)
				repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("db fail"))
			},
			wantErr: wrapError.ErrPatientCreateFailed,
		},
		{
			name:    "success",
			payload: valid,
			setup: func(repo *mocks.MockPatientRepository, org *orgmocks.MockOrganisationServicer) {
				org.EXPECT().GetOrgByID(gomock.Any(), "org-1").Return(organisation.Organisation{ID: "org-1", OrganisationName: "City Hospital"}, nil)
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
