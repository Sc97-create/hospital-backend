package labservice_test

import (
	"errors"
	"testing"
	"time"

	labservice "hospital-backend/internal/labService/labTest"
	"hospital-backend/internal/labService/labTest/dto"
	"hospital-backend/internal/labService/labTest/mocks"
	"hospital-backend/internal/testutil/servicetest"
	wrapError "hospital-backend/shared/error"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func validCreateReq() dto.CreateLabOrderRequest {
	return dto.CreateLabOrderRequest{
		OrganisationID:     "org-1",
		PatientID:          "pat-1",
		DoctorID:           "doc-1",
		DiagnosticCentreID: "dc-1",
		Tests: []dto.CreateOrderedTestInput{
			{Name: "CBC", SubTest: "Hb"},
		},
	}
}

func TestLabOrderServiceCreate(t *testing.T) {
	tests := []struct {
		name    string
		req     dto.CreateLabOrderRequest
		setup   func(*mocks.MockLabOrderRepository)
		wantErr error
	}{
		{
			name:    "missing org",
			req:     dto.CreateLabOrderRequest{PatientID: "p", DoctorID: "d", DiagnosticCentreID: "c", Tests: []dto.CreateOrderedTestInput{{Name: "CBC"}}},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name:    "empty tests",
			req:     dto.CreateLabOrderRequest{OrganisationID: "org-1", PatientID: "p", DoctorID: "d", DiagnosticCentreID: "c"},
			wantErr: wrapError.ErrInvalidRequest,
		},
		{
			name: "repo error",
			req:  validCreateReq(),
			setup: func(m *mocks.MockLabOrderRepository) {
				m.EXPECT().Create(gomock.Any(), gomock.Any()).Return(errors.New("insert failed"))
			},
			wantErr: wrapError.ErrLabOrderCreateFailed,
		},
		{
			name: "success without appointment",
			req:  validCreateReq(),
			setup: func(m *mocks.MockLabOrderRepository) {
				m.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&labservice.LabOrders{})).DoAndReturn(
					func(_ interface{}, row *labservice.LabOrders) error {
						if row.OrganisationID != "org-1" || row.PatientID != "pat-1" {
							t.Fatalf("unexpected row: %+v", row)
						}
						if row.AppointmentID != nil {
							t.Fatalf("expected nil appointment_id, got %v", row.AppointmentID)
						}
						if row.Status != labservice.OrderStatusPending {
							t.Fatalf("expected pending status, got %q", row.Status)
						}
						if len(row.LabTests) != 1 || row.LabTests[0].Name != "CBC" || row.LabTests[0].Status != labservice.TestStatusPending {
							t.Fatalf("unexpected tests: %+v", row.LabTests)
						}
						return nil
					},
				)
			},
		},
		{
			name: "success with appointment",
			req: func() dto.CreateLabOrderRequest {
				r := validCreateReq()
				r.AppointmentID = "appt-1"
				return r
			}(),
			setup: func(m *mocks.MockLabOrderRepository) {
				m.EXPECT().Create(gomock.Any(), gomock.AssignableToTypeOf(&labservice.LabOrders{})).DoAndReturn(
					func(_ interface{}, row *labservice.LabOrders) error {
						if row.AppointmentID == nil || *row.AppointmentID != "appt-1" {
							t.Fatalf("expected appointment_id appt-1, got %v", row.AppointmentID)
						}
						return nil
					},
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mocks.NewMockLabOrderRepository(ctrl)
			if tt.setup != nil {
				tt.setup(repo)
			}
			svc := labservice.NewLabOrderService(repo)
			id, err := svc.Create(servicetest.NopLogger(), tt.req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil || id == "" {
				t.Fatalf("expected id, got %q err=%v", id, err)
			}
		})
	}
}

func TestLabOrderServiceGetByID(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockLabOrderRepository(ctrl)
	svc := labservice.NewLabOrderService(repo)

	t.Run("empty id", func(t *testing.T) {
		_, err := svc.GetByID(servicetest.NopLogger(), " ")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo.EXPECT().FindMany(gomock.Any(), gomock.Any(), "missing").Return([]labservice.LabOrderListRow{}, nil)
		_, err := svc.GetByID(servicetest.NopLogger(), "missing")
		if !errors.Is(err, wrapError.ErrLabOrderNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		now := time.Now()
		repo.EXPECT().FindMany(gomock.Any(), gomock.Any(), "lo-1").Return([]labservice.LabOrderListRow{{
			ID:             "lo-1",
			OrganisationID: "org-1",
			PatientID:      "pat-1",
			PatientName:    "Jane Doe",
			PatientCode:    "CLI-001",
			PatientAge:     32,
			DoctorID:       "doc-1",
			DoctorName:     "John Smith",
			Status:         labservice.OrderStatusPending,
			LabTests:       labservice.OrderedLabTestList{{ID: "t1", Name: "CBC", Status: labservice.TestStatusPending}},
			CreatedAt:      now,
			UpdatedAt:      now,
		}}, nil)
		got, err := svc.GetByID(servicetest.NopLogger(), "lo-1")
		if err != nil || got.ID != "lo-1" || len(got.Tests) != 1 {
			t.Fatalf("got %+v err=%v", got, err)
		}
		if got.Patient.PatientName != "Jane Doe" || got.Patient.PatientCode != "CLI-001" || got.Patient.Age != 32 {
			t.Fatalf("unexpected patient details: %+v", got.Patient)
		}
		if got.Doctor.DoctorName != "John Smith" || got.Doctor.DoctorID != "doc-1" {
			t.Fatalf("unexpected doctor details: %+v", got.Doctor)
		}
	})
}

func TestLabOrderServiceList(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockLabOrderRepository(ctrl)
	svc := labservice.NewLabOrderService(repo)

	t.Run("missing org", func(t *testing.T) {
		_, _, err := svc.List(servicetest.NopLogger(), dto.ListLabOrdersRequest{})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		_, _, err := svc.List(servicetest.NopLogger(), dto.ListLabOrdersRequest{
			OrganisationID: "org-1",
			Status:         "archived",
		})
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo.EXPECT().FindMany(gomock.Any(), gomock.Any(), gomock.Any()).Return([]labservice.LabOrderListRow{
			{
				ID:          "lo-1",
				OrganisationID: "org-1",
				PatientName: "Jane Doe",
				DoctorName:  "John Smith",
				Status:      labservice.OrderStatusPending,
				LabTests:    labservice.OrderedLabTestList{{ID: "t1"}},
			},
		}, nil)
		repo.EXPECT().Count(gomock.Any(), gomock.Any(), gomock.Any()).Return(int64(1), nil)

		rows, total, err := svc.List(servicetest.NopLogger(), dto.ListLabOrdersRequest{
			OrganisationID: "org-1",
			Limit:          10,
			PageNo:         1,
		})
		if err != nil || total != 1 || len(rows) != 1 || rows[0].ID != "lo-1" {
			t.Fatalf("got rows=%+v count=%d err=%v", rows, total, err)
		}
		if rows[0].PatientName != "Jane Doe" || rows[0].DoctorName != "John Smith" || rows[0].TestsCount != 1 {
			t.Fatalf("unexpected list item: %+v", rows[0])
		}
	})
}

func TestLabOrderServiceMarkIsPayment(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockLabOrderRepository(ctrl)
	svc := labservice.NewLabOrderService(repo)

	t.Run("invalid empty id", func(t *testing.T) {
		err := svc.MarkIsPayment(servicetest.NopLogger(), "", true)
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo.EXPECT().Exec(
			gomock.Any(),
			gomock.Any(),
			true,
			labservice.OrderStatusPending,
			labservice.OrderStatusPaid,
			gomock.Any(),
			"lo-1",
		).Return(nil)
		if err := svc.MarkIsPayment(servicetest.NopLogger(), "lo-1", true); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo.EXPECT().Exec(
			gomock.Any(),
			gomock.Any(),
			true,
			labservice.OrderStatusPending,
			labservice.OrderStatusPaid,
			gomock.Any(),
			"missing",
		).Return(gorm.ErrRecordNotFound)
		err := svc.MarkIsPayment(servicetest.NopLogger(), "missing", true)
		if !errors.Is(err, wrapError.ErrLabOrderNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})
}

func TestLabOrderServiceUpdateStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockLabOrderRepository(ctrl)
	svc := labservice.NewLabOrderService(repo)

	t.Run("invalid status", func(t *testing.T) {
		err := svc.UpdateStatus(servicetest.NopLogger(), "lo-1", "paused")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo.EXPECT().Exec(gomock.Any(), gomock.Any(), labservice.OrderStatusCompleted, gomock.Any(), "lo-missing").Return(gorm.ErrRecordNotFound)
		err := svc.UpdateStatus(servicetest.NopLogger(), "lo-missing", "completed")
		if !errors.Is(err, wrapError.ErrLabOrderNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo.EXPECT().Exec(gomock.Any(), gomock.Any(), labservice.OrderStatusCancelled, gomock.Any(), "lo-1").Return(nil)
		if err := svc.UpdateStatus(servicetest.NopLogger(), "lo-1", "CANCELLED"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestLabOrderServiceUpdateTestStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockLabOrderRepository(ctrl)
	svc := labservice.NewLabOrderService(repo)

	t.Run("invalid", func(t *testing.T) {
		err := svc.UpdateTestStatus(servicetest.NopLogger(), "lo-1", "t1", "bogus")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("order not found", func(t *testing.T) {
		repo.EXPECT().FindByID(gomock.Any(), "lo-missing").Return(labservice.LabOrders{}, gorm.ErrRecordNotFound)
		err := svc.UpdateTestStatus(servicetest.NopLogger(), "lo-missing", "t1", "completed")
		if !errors.Is(err, wrapError.ErrLabOrderNotFound) {
			t.Fatalf("expected not found, got %v", err)
		}
	})

	t.Run("test not found", func(t *testing.T) {
		repo.EXPECT().FindByID(gomock.Any(), "lo-1").Return(labservice.LabOrders{
			ID:       "lo-1",
			Status:   labservice.OrderStatusPending,
			LabTests: labservice.OrderedLabTestList{{ID: "t1", Name: "CBC", Status: labservice.TestStatusPending}},
		}, nil)
		err := svc.UpdateTestStatus(servicetest.NopLogger(), "lo-1", "missing", "completed")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("completes order when all tests done", func(t *testing.T) {
		repo.EXPECT().FindByID(gomock.Any(), "lo-1").Return(labservice.LabOrders{
			ID:     "lo-1",
			Status: labservice.OrderStatusPending,
			LabTests: labservice.OrderedLabTestList{
				{ID: "t1", Name: "CBC", Status: labservice.TestStatusPending},
				{ID: "t2", Name: "Lipid", Status: labservice.TestStatusCompleted},
			},
		}, nil)
		repo.EXPECT().Save(gomock.Any(), gomock.AssignableToTypeOf(&labservice.LabOrders{})).DoAndReturn(
			func(_ interface{}, row *labservice.LabOrders) error {
				if row.LabTests[0].Status != labservice.TestStatusCompleted {
					t.Fatalf("expected t1 completed, got %q", row.LabTests[0].Status)
				}
				if row.Status != labservice.OrderStatusCompleted {
					t.Fatalf("expected order completed, got %q", row.Status)
				}
				return nil
			},
		)
		if err := svc.UpdateTestStatus(servicetest.NopLogger(), "lo-1", "t1", "completed"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestLabOrderServiceUpdateTestsStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockLabOrderRepository(ctrl)
	svc := labservice.NewLabOrderService(repo)

	t.Run("invalid empty ids", func(t *testing.T) {
		err := svc.UpdateTestsStatus(servicetest.NopLogger(), "lo-1", nil, "completed")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("partial missing test fails", func(t *testing.T) {
		repo.EXPECT().FindByID(gomock.Any(), "lo-1").Return(labservice.LabOrders{
			ID:     "lo-1",
			Status: labservice.OrderStatusPending,
			LabTests: labservice.OrderedLabTestList{
				{ID: "t1", Name: "CBC", Status: labservice.TestStatusPending},
			},
		}, nil)
		err := svc.UpdateTestsStatus(servicetest.NopLogger(), "lo-1", []string{"t1", "missing"}, "completed")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("test ids from another order fail", func(t *testing.T) {
		repo.EXPECT().FindByID(gomock.Any(), "lo-1").Return(labservice.LabOrders{
			ID:     "lo-1",
			Status: labservice.OrderStatusPending,
			LabTests: labservice.OrderedLabTestList{
				{ID: "t1", Name: "CBC", Status: labservice.TestStatusPending},
				{ID: "t2", Name: "FBS", Status: labservice.TestStatusPending},
			},
		}, nil)
		err := svc.UpdateTestsStatus(servicetest.NopLogger(), "lo-1", []string{"other-order-test"}, "completed")
		if !errors.Is(err, wrapError.ErrInvalidRequest) {
			t.Fatalf("expected invalid request, got %v", err)
		}
	})

	t.Run("bulk completes selected tests and order", func(t *testing.T) {
		repo.EXPECT().FindByID(gomock.Any(), "lo-1").Return(labservice.LabOrders{
			ID:     "lo-1",
			Status: labservice.OrderStatusPending,
			LabTests: labservice.OrderedLabTestList{
				{ID: "t1", Name: "CBC", Status: labservice.TestStatusPending},
				{ID: "t2", Name: "Lipid", Status: labservice.TestStatusPending},
			},
		}, nil)
		repo.EXPECT().Save(gomock.Any(), gomock.AssignableToTypeOf(&labservice.LabOrders{})).DoAndReturn(
			func(_ interface{}, row *labservice.LabOrders) error {
				if row.LabTests[0].Status != labservice.TestStatusCompleted || row.LabTests[1].Status != labservice.TestStatusCompleted {
					t.Fatalf("expected both tests completed, got %+v", row.LabTests)
				}
				if row.Status != labservice.OrderStatusCompleted {
					t.Fatalf("expected order completed, got %q", row.Status)
				}
				return nil
			},
		)
		if err := svc.UpdateTestsStatus(servicetest.NopLogger(), "lo-1", []string{"t1", "t2"}, "completed"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
