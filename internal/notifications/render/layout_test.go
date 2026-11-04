package render

import (
	"strings"
	"testing"

	"hospital-backend/internal/notifications/dto"
)

func TestPaymentEmailIncludesFloatohealthSignature(t *testing.T) {
	renderer, err := NewHTMLRenderer(nil)
	if err != nil {
		t.Fatal(err)
	}

	html, err := renderer.Render("payment_recieved", dto.NotificationModel{
		HospitalName:  "Sadanand clinic",
		PatientName:   "Kuvempu",
		Currency:      "INR",
		Amount:        "150.00",
		PaymentStatus: "payment_recieved",
		PaidAt:        "04 Nov 2026, 09:41 PM",
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"FLOATOHEALTH",
		"Sadanand clinic",
		"Sent by",
		"Floatohealth",
		"support@floatohealth.com",
		"INR 150.00",
		"Received",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing %q", want)
		}
	}
	if strings.Contains(html, "payment_recieved") {
		t.Fatal("raw payment status was rendered")
	}
}
