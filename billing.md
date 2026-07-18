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

Append-only audit log of every inbound gateway webhook (success or failure). Used for idempotency, debugging, and payment history. Processing updates `payment_attempts`, `payments`, and invoice state separately.
Column	                            Type	Purpose
id	                                   UUID	PK
payment_attempt_id	                   UUID	FK (nullable until matched via provider_order_id / provider_payment_id)
provider	                         Enum	Razorpay / Cashfree
provider_event_id	               VARCHAR	Gateway event id (X-Razorpay-Event-Id), unique for idempotency
event_type	                       VARCHAR	e.g. payment.captured, payment.failed
provider_payment_id	               VARCHAR	Gateway payment reference (pay_...)
provider_order_id	               VARCHAR	Gateway order reference (order_...)
amount	                          Decimal	Denormalized amount from payload
currency	                       VARCHAR(5)	e.g. INR
gateway_status	                   Enum	captured / failed / authorized (from payload)
signature_valid	                  Boolean	Whether webhook signature verification passed
processing_status	                 Enum	received / processed / failed / ignored
processing_error	                  TEXT	Error if internal processing failed
raw_payload	                       JSONB	Full webhook JSON body
received_at	                   Timestamp	When webhook was received
processed_at	                   Timestamp	When internal processing completed
created_at	                   Timestamp	Audit

```go
type WebhookEvents struct {
	ID                string            `json:"id" gorm:"type:uuid;not null;primaryKey"`
	PaymentAttemptID  *string           `json:"payment_attempt_id,omitempty" gorm:"type:uuid"`
	Provider          string            `json:"provider"`
	ProviderEventID   string            `json:"provider_event_id" gorm:"type:varchar;uniqueIndex"`
	EventType         string            `json:"event_type" gorm:"type:varchar"`
	ProviderPaymentID string            `json:"provider_payment_id" gorm:"type:varchar"`
	ProviderOrderID   string            `json:"provider_order_id" gorm:"type:varchar"`
	Amount            float64           `json:"amount" gorm:"type:numeric(10,2)"`
	Currency          string            `json:"currency" gorm:"type:varchar(5)"`
	GatewayStatus     string            `json:"gateway_status" gorm:"type:varchar(50)"`
	SignatureValid    bool              `json:"signature_valid"`
	ProcessingStatus  string            `json:"processing_status" gorm:"type:varchar(50)"`
	ProcessingError   string            `json:"processing_error,omitempty" gorm:"type:text"`
	RawPayload        datatypes.JSONMap `json:"raw_payload" gorm:"type:jsonb"`
	ReceivedAt        time.Time         `json:"received_at" gorm:"type:timestamptz"`
	ProcessedAt       *time.Time        `json:"processed_at,omitempty" gorm:"type:timestamptz"`
	CreatedAt         time.Time         `json:"created_at" gorm:"autoCreateTime"`
}
```



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

1. checkout api is called with below payload

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

1. process the payload in billing module
2. save the necessary item in necessary table
 for invoice keep status as unpaid
3. generate qr code and they can pay from there, since everyone is connected to upi, if card then how to integrate swipe machine and get transaction id need to figure out
4. once webhook gets called and verify payment, verifies it and transaction history is stored with status what we get from gateway
5. change the status of invoice 
6. decrement the inventory count=> once paid
7. it should be done in one transaction all this

architecture for next 5 years we can think of

```
                  Kubernetes
                      │
                Load Balancer
                      │
             Multiple Go Pods
                      │
             Modular Monolith
                      │
```

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

```
                Invoice
                   │
                   │ 1
                   ▼
              Payments
                   │
      ┌────────────┴────────────┐
      │                         │
      ▼                         ▼
```

 Payment Attempt #1         Payment Attempt #2
          │                         │
          │                         ▼
          │                    Retry Payment
          │
          ├──────────────┐
          ▼              ▼
      Refunds      Webhook Events

invoice example lifecycle

```
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
```

flow diagram of how payment processes

```
                    HTTP Handler
                         │
                         ▼
                 Payment Service
                         │
    ┌────────────────────┴───────────────────┐
    │                                        │
```

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

```json
standard
{
  "account_id": "acc_OU2H3nkLn9jDVo",
  "contains": [
    "payment_link",
    "order",
    "payment"
  ],
  "created_at": 1749618314,
  "entity": "event",
  "event": "payment_link.paid",
  "payload": {
    "order": {
      "entity": {
        "account_number": null,
        "amount": 1000,
        "amount_due": 0,
        "amount_paid": 1000,
        "app_offer": false,
        "attempts": 1,
        "authorized": true,
        "bank": null,
        "bank_account": null,
        "checkout_config_id": null,
        "created_at": 1749618325,
        "currency": "INR",
        "customer_id": null,
        "discount": false,
        "first_payment_min_amount": null,
        "force_offer": null,
        "id": "QflczVVaNJciLq",
        "late_auth_config_id": null,
        "merchant_id": "OU2H3nkLn9jDVo",
        "method": null,
        "notes": [],
        "offers": {},
        "order_metas": [],
        "order_relationships": [],
        "partial_payment": false,
        "payer_name": null,
        "payment_capture": true,
        "product_id": "QflcnnZqCekuvL",
        "product_type": "payment_link_v2",
        "provider_context": null,
        "public_key": "rzp_live_XXXXXXXXXXXXXX",
        "public_response": null,
        "receipt": "23",
        "reference2": null,
        "reference3": null,
        "reference4": null,
        "reference5": null,
        "reference6": null,
        "reference7": null,
        "reference8": null,
        "source": null,
        "status": "paid",
        "transfers": null,
        "updated_at": 1749618372
      }
    },
    "payment": {
      "entity": {
        "acquirer_data": {
          "rrn": "103608848276"
        },
        "amount": 1000,
        "amount_refunded": 0,
        "amount_transferred": 0,
        "bank": null,
        "base_amount": 1000,
        "captured": true,
        "card": null,
        "card_id": null,
        "contact": "+919876543210",
        "created_at": 1749618371,
        "currency": "INR",
        "description": "#QflcnnZqCekuvL",
        "email": null,
        "entity": "payment",
        "error_code": null,
        "error_description": null,
        "error_reason": null,
        "error_source": null,
        "error_step": null,
        "fee": 24,
        "fee_bearer": "platform",
        "id": "pay_Qfldmt5StKZFCB",
        "international": false,
        "invoice_id": null,
        "method": "upi",
        "notes": [],
        "order_id": "order_QflczVVaNJciLq",
        "refund_status": null,
        "status": "captured",
        "tax": 4,
        "upi": {
          "payer_account_type": "bank_account",
          "vpa": "gaurav.kumar@exampleupi"
        },
        "vpa": "gaurav.kumar@exampleupi",
        "wallet": null
      }
    },
    "payment_link": {
      "entity": {
        "accept_partial": false,
        "amount": 1000,
        "amount_paid": 1000,
        "cancelled_at": 0,
        "created_at": 1749618314,
        "currency": "INR",
        "customer": {
          "contact": "9000090000",
          "email": "gauravkumar@example.com"
        },
        "description": "Test Payment",
        "expire_by": 0,
        "expired_at": 0,
        "first_min_partial_amount": 0,
        "id": "plink_QflcnnZqCekuvL",
        "notes": null,
        "notify": {
          "email": false,
          "sms": false,
          "whatsapp": false
        },
        "order_id": "order_QflczVVaNJciLq",
        "reference_id": "23",
        "reminder_enable": false,
        "reminders": {
          "status": "failed"
        },
        "short_url": "https://rzp.io/rzp/twH5w1Y",
        "status": "paid",
        "updated_at": 1749618371,
        "upi_link": false,
        "user_id": "HWPf1AudnADDV5",
        "whatsapp_link": false
      }
    }
  }
}

upi
{
  "account_id": "acc_MWWeS9Ico5tQBk",
  "contains": [
    "payment_link",
    "order",
    "payment"
  ],
  "created_at": 1748586679,
  "entity": "event",
  "event": "payment_link.paid",
  "payload": {
    "order": {
      "entity": {
        "account_number": null,
        "amount": 100,
        "amount_due": 0,
        "amount_paid": 100,
        "app_offer": false,
        "attempts": 1,
        "authorized": true,
        "bank": null,
        "bank_account": null,
        "checkout_config_id": null,
        "created_at": 1748586685,
        "currency": "INR",
        "customer_id": null,
        "discount": false,
        "first_payment_min_amount": null,
        "force_offer": null,
        "id": "Qb2gOAUzSm5zpv",
        "late_auth_config_id": null,
        "merchant_id": "MWWeS9Ico5tQBk",
        "method": "upi",
        "notes": {
          "policy_name": "Jeevan Bima"
        },
        "offers": {},
        "order_metas": [],
        "order_relationships": [],
        "partial_payment": false,
        "payer_name": null,
        "payment_capture": true,
        "product_id": "Qb2gHrKr01Maky",
        "product_type": "payment_link_v2",
        "provider_context": null,
        "public_key": "rzp_live_XXXXXXXXXXXXXX",
        "public_response": null,
        "receipt": "UPItest2",
        "reference2": null,
        "reference3": null,
        "reference4": null,
        "reference5": null,
        "reference6": null,
        "reference7": null,
        "reference8": null,
        "source": null,
        "status": "paid",
        "transfers": null,
        "updated_at": 1748586717
      }
    },
    "payment": {
      "entity": {
        "acquirer_data": {
          "rrn": "551652213711"
        },
        "amount": 100,
        "amount_captured": null,
        "amount_refunded": 0,
        "amount_transferred": 0,
        "bank": null,
        "base_amount": 100,
        "captured": true,
        "card": null,
        "card_id": null,
        "contact": "+918840152270",
        "created_at": 1748586694,
        "currency": "INR",
        "description": "#Qb2gHrKr01Maky",
        "email": "gaurav.kumar@example.com",
        "entity": "payment",
        "error_code": null,
        "error_description": null,
        "error_reason": null,
        "error_source": null,
        "error_step": null,
        "fee": 2,
        "fee_bearer": "platform",
        "id": "pay_Qb2gYRc7dxedX8",
        "international": false,
        "invoice_id": null,
        "method": "upi",
        "notes": {
          "policy_name": "Jeevan Bima"
        },
        "order_id": "order_Qb2gOAUzSm5zpv",
        "provider": null,
        "refund_status": null,
        "reward": null,
        "status": "captured",
        "tax": 0,
        "upi": {
          "payer_account_type": "bank_account",
          "vpa": "gaurav.kumar@okexample"
        },
        "vpa": "gaurav.kumar@okexample",
        "wallet": null
      }
    },
    "payment_link": {
      "entity": {
        "accept_partial": false,
        "amount": 100,
        "amount_paid": 100,
        "callback_method": "get",
        "callback_url": "https://example-callback-url.com/",
        "cancelled_at": 0,
        "created_at": 1748586679,
        "currency": "INR",
        "customer": {
          "contact": "+919000090000",
          "email": "gaurav.kumar@example.com",
          "name": "Gaurav Kumar"
        },
        "description": "Payment for policy no #23456",
        "expire_by": 1748587814,
        "expired_at": 0,
        "first_min_partial_amount": 0,
        "id": "plink_Qb2gHrKr01Maky",
        "notes": {
          "policy_name": "Jeevan Bima"
        },
        "notify": {
          "email": true,
          "sms": true,
          "whatsapp": false
        },
        "order_id": "order_Qb2gOAUzSm5zpv",
        "reference_id": "UPItest2",
        "reminder_enable": true,
        "reminders": {
          "status": "failed"
        },
        "short_url": "https://rzp.io/rzp/S12PdyW",
        "status": "paid",
        "updated_at": 1748586718,
        "upi_link": true,
        "user_id": "",
        "whatsapp_link": false
      }
    }
  }
}
```



---



## Pending Features / TODOs



### 1. `DispenseMedicine` ~~in prescription~~ ✅ DONE

- Moved to `internal/billing/controllers.go` as `Checkout`
- `Checkout` correctly calls `InvoiceServ.CreatePaymentLink`



### 2. Fix `Checkout` controller silently ignoring error

- **File:** `internal/billing/controllers.go` line 67
- `IB.BillingServ.CreatePaymentLink(checkoutReq)` — error return is discarded
- Client always gets `200 "stored"` even on DB failure or Razorpay error
- Fix:
  ```go
  _, err = IB.BillingServ.CreatePaymentLink(checkoutReq)
  if err != nil {
      return wrapErrors.Wrap(err, c, 500)
  }
  ```



### 3. ~~Update `BalanceAfterDispense` on prescription items after payment confirmed~~ ✅ DONE
- `UpdateDispenseItemQty` called per item in `PaymentLinkPaid` loop in `webhook_service.go`
- `prescription_item_id` added to JOIN query and `MedInvoiceItemResponse`



### 4. ~~Dispense quantity validation before checkout~~ ✅ DONE
- Validation loop in `addInvoiceItems` (`invoice-item.services.go`) checks:
  - `dispensed_qty <= (prescribed_qty - balance_after_dispense)` — returns error if exceeded
  - `dispensed_qty <= current_stock_units` — returns error if insufficient inventory stock
- `GetqtyByMedicine` now also fetches `balance_after_dispense` from the DB



### 5. ~~Prescription status update after dispense~~ ✅ DONE
- Item-level status (`fully_dispensed` / `partially_dispensed`) updated per item via `UpdateIPrescriptionStatus`
- Prescription-level status set via XOR flag + `UpdateExtPrescriptionStatus`
- `StatusFullyDispensed`, `StatusPartiallyDispensed` added to `prescription/constants.go`



### 6. ~~Handle `PaymentLinkCancelled` in webhook~~ ✅ DONE
- Updates `PaymentAttempts.PaymentLinkStatus` and `PaymentStatus` to `cancelled`
- Invoice status kept as `unpaid` via `UpdateInvoiceStatus`



### 7. ~~Handle `PaymentLinkExpired` in webhook~~ ✅ DONE
- Updates `PaymentAttempts.PaymentLinkStatus` and `PaymentStatus` to `constants.StatusExpired`
- Invoice stays `unpaid` — cashier can generate a new payment link
- `StatusExpired` already present in `pkg/constants/constant.go`



### 8. ~~Guard against partial payments in `PaymentLinkPaid`~~ ✅ DONE
- Guard added at top of `PaymentLinkPaid` case in `webhook_service.go`
- If `AcceptPartial && AmountPaid < invoiceInfo.Amount` → sets attempt status to `partially_paid`, commits, and returns without dispensing
- `StatusPartiallyPaid = "partially_paid"` added to `pkg/constants/constant.go`



### 9. ~~Update invoice status to `paid` on `PaymentLinkPaid`~~ ✅ DONE
- `UpdateInvoiceStatus` called with `constants.InvoicePaid` after successful dispense in `webhook_service.go`



### 10. ~~Fix `FindMany` pagination count always returning 0~~ ✅ DONE
- Uncommented `prescriptionRepo.Count(organisationID)` in `prescriptions.services.go` — `totalInt` now returns correct count



### 11. ~~Fix `batch_id` vs `medicine_inventory_id` naming mismatch~~ ✅ DONE
- `tomapDispense` with `batch_id` was removed from `prescription/controllers.go` when `DispenseMedicine` moved to billing
- `billing/controllers.go` `toDispenseItems` correctly uses `medicine_inventory_id` throughout



### 12. ~~Fix `GetInvoiceItemsByInvoiceID` using `Find` instead of `Scan` on raw query~~ ✅ DONE
- Changed `d.db.Raw(query, args...).Find(&invoiceItems)` → `.Scan(&invoiceItems)` in `inovice-item.repository.go`


*Payment Expired*
```json
{
  "account_id": "acc_MWWeS9Ico5tQBk",
  "contains": [
    "payment_link"
  ],
  "created_at": 1748424975,
  "entity": "event",
  "event": "payment_link.expired",
  "payload": {
    "payment_link": {
      "entity": {
        "accept_partial": true,
        "amount": 1000,
        "amount_paid": 0,
        "callback_method": "get",
        "callback_url": "https://example-callback-url.com/",
        "cancelled_at": 0,
        "created_at": 1748424975,
        "currency": "INR",
        "customer": {
          "contact": "+919000090000",
          "email": "gaurav.kumar@example.com",
          "name": "Gaurav Kumar"
        },
        "description": "Payment for policy no #23456",
        "expire_by": 1748426427,
        "expired_at": 1748426438,
        "first_min_partial_amount": 100,
        "id": "plink_QaIlOGFf8KZNF8",
        "notes": {
          "Test": "True"
        },
        "notify": {
          "email": true,
          "sms": true,
          "whatsapp": false
        },
        "reference_id": "NewTestPayment",
        "reminder_enable": true,
        "reminders": {
          "status": "failed"
        },
        "short_url": "https://rzp.io/rzp/REnL57hV",
        "status": "expired",
        "updated_at": 1748424975,
        "upi_link": false,
        "user_id": "",
        "whatsapp_link": false
      }
    }
  }
}
```

