package medicine

import (
	"testing"
	"time"
)

func TestCalculatePaymentDueDate(t *testing.T) {
	svc := NewPurchaseEntryService(nil)
	invoiceDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		paymentTerms Paymentterms
		want         time.Time
	}{
		{name: "cash", paymentTerms: ICash, want: invoiceDate},
		{name: "net 30", paymentTerms: Net30, want: invoiceDate.AddDate(0, 0, 30)},
		{name: "net 45", paymentTerms: Net45, want: invoiceDate.AddDate(0, 0, 45)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.calculatePaymentDueDate(invoiceDate, tt.paymentTerms)
			if !got.Equal(tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
