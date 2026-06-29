## invoice table
type Invoice struct {
	ID              string        `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PrescriptionID  *string       `gorm:"type:uuid" json:"prescription_id,omitempty"`
	Status          string        `gorm:"type:varchar(50);default:'UNPAID'" json:"status"` // 'UNPAID', 'PAID', etc.
	CashierID       string        `gorm:"type:uuid;not null" json:"cashier_id"`
	OrganisationID  string        `gorm:"type:uuid;not null" json:"organisation_id"`
	SubtotalAmount  float64       `gorm:"type:numeric(10,2);not null" json:"subtotal_amount"`
	TaxAmount       float64       `gorm:"type:numeric(10,2);not null;default:0.00" json:"tax_amount"`
	DiscountAmount  float64       `gorm:"type:numeric(10,2);not null;default:0.00" json:"discount_amount"`
	TotalAmountPaid float64       `gorm:"type:numeric(10,2);not null" json:"total_amount_paid"`
	CreatedAt       time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time     `gorm:"autoUpdateTime" json:"updated_at"`
	Items           []InvoiceItem `gorm:"foreignKey:InvoiceID" json:"items,omitempty"`
}

## invoice-item table
type InvoiceItem struct {
	ID            string  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InvoiceID     string  `gorm:"type:uuid;not null;index" json:"invoice_id"`
	MedicineID    string  `gorm:"type:uuid;not null" json:"medicine_id"`
	BatchID       *string `gorm:"type:uuid" json:"batch_id,omitempty"`
	SubTotalPrice float64 `gorm:"type:numeric(10,2);not null" json:"sub_total_price"` // Base price
	TotalPrice    float64 `gorm:"type:numeric(10,2);not null" json:"total_price"`     // Final price
	GivenQty      int     `gorm:"type:int;not null;default:0" json:"given_qty"`       // Dispensed
	PendingQty    int     `gorm:"type:int;not null;default:0" json:"pending_qty"`     // Owed/Backordered
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}


## payments
explained schema
Column	     Type	          Why?
id	         UUID	           Internal primary key
invoice_id	 UUID	          FK to invoice
clinic_id	 UUID	        Multi-tenant support
patient_id	 UUID	       Easier reporting without joining appointments
amount	    Decimal	      Invoice amount at payment creation
currency	VARCHAR(5)	    Future international support
status	    Enum	        Current payment status
source	    Enum	          LINK / QR / POS / MANUAL
channel	    Enum	          UPI / CARD / WALLET / UNKNOWN
initiated_by	Enum	    PATIENT / RECEPTIONIST / SYSTEM
expires_at	 Timestamp	    Payment expiry
paid_at	   Timestamp	        First successful payment
idempotency_key	  VARCHAR	  Prevent duplicate checkout
created_at 	Timestamp	          Audit
updated_at	Timestamp	         Audit
deleted_at	Soft delete	         GORM

## payment-attempts
Column	            Type	Why?
id	                UUID	PK
payment_id	        UUID	FK
provider	        Enum	Razorpay / Cashfree
attempt_no	         INT	Retry number
provider_order_id	VARCHAR	Gateway order reference
provider_payment_id	VARCHAR	Gateway payment reference
provider_link_id	VARCHAR	Payment Link reference (optional)
payment_link	    TEXT	URL sent to customer
qr_code_url	        TEXT	Dynamic QR
amount	           Decimal	Attempt amount
status	            Enum	Attempt lifecycle
failure_reason	    TEXT	Gateway failure
provider_data	   JSONB	Provider-specific fields
gateway_response	JSONB	Raw create-payment response
created_at	      Timestamp	Audit

## refunds (if any)

Column	                           Purpose
id	                                 PK
payment_attempt_id	                 FK
provider_refund_id	          Gateway refund id
amount	                         Refund amount
reason	                         Why refunded
status	                       Refund lifecycle
provider_data	                Provider fields
gateway_response	              Raw refund response
created_at	                         Audit
updated_at	                          Audit

## webhook events

Column	                            Purpose
id	                                   PK
payment_attempt_id	                   FK
provider_refund_id	             Gateway refund id
amount	                          Refund amount
reason	                           Why refunded
status	                         Refund lifecycle
provider_data	                 Provider fields
gateway_response	             Raw refund response
created_at	                          Audit
updated_at	                          Audit



## transaction table
type TransactionHistory struct {
	ID              string    `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InvoiceID       string    `gorm:"type:uuid;not null;index" json:"invoice_id"`
	PaymentType     string    `gorm:"type:varchar(50);not null" json:"payment_type"` // 'CASH', 'UPI', 'CARD'
	TxID            string    `gorm:"type:varchar(255)" json:"tx_id"`                // Reference Code
	Status          string    `gorm:"type:varchar(50);default:'PENDING'" json:"status"` // 'SUCCESS', 'FAILED'
	ReasonOrMessage string    `gorm:"type:text;column:reason_or_message" json:"reason_or_message"`
	Amount          float64   `gorm:"type:numeric(10,2);not null" json:"amount"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
}

## how it works

1) checkout api is called with below payload

{
  "prescription_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d",
  "clinic_id": "11111111-2222-3333-4444-555555555555",
  "patient_id": "a2b3c4d5-e6f7-8a9b-0c1d-2e3f4a5b6c7d",
  "cashier_id": "ffffffff-eeee-dddd-cccc-bbbbbbbbbbbb",
  "supplier_id":"uuid"
  "payment_mode": "UPI",
  "financials": {
    "subtotal_amount": 135.00,
    "tax_amount": 16.20,
    "discount_amount": 10.00,
    "total_amount_paid": 141.20
  },
  "dispensed_items": [
    {
      "medicine_id": "4a5c6d7e-8f9a-0b1c-2d3e-4f5a6b7c8d9e",
      "batch_id": "99999999-8888-7777-6666-555555555555",
      "batch_no": "BAT-4029",
      "quantity_sold_units": 25,
      "unit_price_charged": 2.00,
      "computed_item_total": 50.00
    },
    {
      "medicine_id": "4a5c6d7e-8f9a-0b1c-2d3e-4f5a6b7c8d9e",
      "batch_id": "22222222-3333-4444-5555-666666666666",
      "batch_no": "BAT-9112",
      "quantity_sold_units": 15,
      "unit_price_charged": 2.00,
      "computed_item_total": 30.00
    }
  ]
}

2) process the payload in billing module

3) save the necessary item in necessary table
    for invoice keep status as unpaid

4) generate qr code and they can pay from there, since everyone is connected to upi, if card then how to integrate swipe machine and get transaction id need to figure out

5) once webhook gets called and verify payment, verifies it and transaction history is stored with status what we get from gateway

6) change the status of invoice 

7) decrement the inventory count=> once paid

8) it should be done in one transaction all this


architecture for next 5 years we can think of

                      Kubernetes
                          │
                    Load Balancer
                          │
                 Multiple Go Pods
                          │
                 Modular Monolith
                          │
 ┌─────────────┬─────────────┬─────────────┬─────────────┐
 │ Auth        │ Patients    │ Billing     │ Payments    │
 ├─────────────┼─────────────┼─────────────┼─────────────┤
 │ Doctors     │ Pharmacy    │ Inventory   │ Notification│
 └─────────────┴─────────────┴─────────────┴─────────────┘
                          │
                 Payment Strategy
                          │
      Razorpay / Cashfree / Stripe / PhonePe
                          │
                  PostgreSQL + Redis
                          │
                Background Worker Pool
                          │
         Emails • SMS • Receipts • Webhooks • Retries


This design keeps operational complexity low while giving you a clean path to scale. You get the simplicity of a modular monolith, horizontal scaling through Kubernetes pods, efficient concurrency inside each pod where it actually helps, and the flexibility to extract the payment module into its own service later if your team size or deployment needs justify it.


                    Invoice
                       │
                       │ 1
                       ▼
                  Payments
                       │
          ┌────────────┴────────────┐
          │                         │
          ▼                         ▼
 Payment Attempt #1         Payment Attempt #2
          │                         │
          │                         ▼
          │                    Retry Payment
          │
          ├──────────────┐
          ▼              ▼
      Refunds      Webhook Events


invoice example lifecycle

            Checkout

                ↓

                Invoice Created

                ↓

                Payment Created

                ↓

                Attempt Created

                ↓

                Gateway Create Payment

                ↓

                Save Provider Response

                ↓

                Send Link

                ↓

                Customer Pays

                ↓

                Webhook Stored

                ↓

                Verify Signature

                ↓

                Update Attempt

                ↓

                Update Payment

                ↓

                Update Invoice



flow diagram of how payment processes

                        HTTP Handler
                             │
                             ▼
                     Payment Service
                             │
        ┌────────────────────┴───────────────────┐
        │                                        │
 Payment Repository                    Provider Factory
        │                                        │
        ▼                                        ▼
 PostgreSQL                            PaymentProvider Interface
                                                │
                   ┌────────────────────────────┴───────────────────────────┐
                   ▼                            ▼                           ▼
              RazorpayProvider          CashfreeProvider             StripeProvider



for payment webhook event

post => /webhook
x-signature => webhook-signature
how to verify webhook
create expected signature
webhook-secret + webhook_payload


payment.captured
  type=> 
  ```json
  upi
  {
  "entity": "event",
  "account_id": "acc_BFQ7uQEaa7j2z7",
  "event": "payment.captured",
  "contains": [
    "payment"
  ],
  "payload": {
    "payment": {
      "entity": {
        "id": "pay_DESyzxuld02Zul",
        "entity": "payment",
        "amount": 100,
        "currency": "<currency>",
        "base_amount": 100,
        "status": "captured",
        "order_id": "order_DESxiijbl9xjDB",
        "invoice_id": null,
        "international": false,
        "method": "upi",
        "amount_refunded": 0,
        "amount_transferred": 0,
        "refund_status": null,
        "captured": true,
        "description": null,
        "card_id": null,
        "bank": null,
        "wallet": null,
        "vpa": "gaurav.kumar@upi",
        "email": "<email>",
        "contact": "<phone>",
        "notes": [],
        "fee": 2,
        "tax": 0,
        "error_code": null,
        "error_description": null,
        "error_source": null,
        "error_step": null,
        "error_reason": null,
        "acquirer_data": {
          "rrn": "0125836177"
        },
        "created_at": 1567675356,
        "upi": {
          "payer_account_type": "credit_card",
          "vpa": "gaurav.kumar@upi",
          "flow": "intent"
        }
      }
    }
  },
  "created_at": 1567675356
}

Card

{
  "account_id": "acc_BFQ7uQEaa7j2z7",
  "contains": [
    "payment"
  ],
  "created_at": 1691735748,
  "entity": "event",
  "event": "payment.captured",
  "payload": {
    "payment": {
      "entity": {
        "acquirer_data": {
          "auth_code": "828553",
          "rrn": "322206890934"
        },
        "amount": 100,
        "amount_refunded": 0,
        "amount_transferred": 0,
        "bank": null,
        "captured": true,
        "card": {
          "emi": false,
          "entity": "card",
          "id": "card_DESp9fNnu0RoNc",
          "iin": "999999",
          "international": false,
          "issuer": null,
          "last4": "0153",
          "name": "<name>",
          "network": "Visa",
          "sub_type": "business",
          "type": "debit"
        },
        "card_id": "card_DESp9fNnu0RoNc",
        "contact": "<phone>",
        "created_at": 1567674797,
        "currency": "INR",
        "description": null,
        "email": "<email>",
        "entity": "payment",
        "error_code": "",
        "error_description": "",
        "error_reason": null,
        "error_source": null,
        "error_step": null,
        "fee": null,
        "id": "pay_DESp9bgForNoUd",
        "international": false,
        "invoice_id": null,
        "method": "card",
        "notes": [],
        "order_id": "order_DESoU0U4ikYA19",
        "refund_status": null,
        "status": "captured",
        "tax": null,
        "token_id": "token_MOfFlFTC1CBDOi",
        "vpa": null,
        "wallet": null
      }
    }
  }
}
```