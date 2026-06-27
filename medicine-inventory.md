{
  "invoice_number": "GST-2026-9842",    => from invoice need to get
  "invoice_date": "2026-06-20T12:00:00Z",  => auto
  "payment_due_date": "2026-07-20T12:00:00Z",  => auto/manual
  "supplier_id": "8f8b6528-795a-4952-b88d-71b302c0b0a8",  => auto
  "organisation_id": "3c9b1234-5678-90ab-cdef-1234567890ab", => auto
  "items": [
    {
      "medicine_id": "4a5c6d7e-8f9a-0b1c-2d3e-4f5a6b7c8d9e", => auto/ find from existing and update it
      "name": "Dolo 650mg",  => if found already just update the prices if new
      "form": "TABLET",
      "strength": "650mg",
      "hsn_code": "30049011",
      "barcode": "8901082004245",
      "batch_no": "BAT-4029",
      "expires_at": "2028-12-31T00:00:00Z",
      "shelf_location": "Rack 3-B",
      "purchase_qty_boxes": 10,
      "units_per_box": 15,
      "pricing": {
        "mrp": 30.00,
        "purchase_price": 21.50,
        "discount": 5.00,
        "discount_type": "PERCENTAGE",
        "selling_price": 28.50,
        "total_price": 215.00
      }
    },
    {
      "medicine_id": "", 
      "name": "Elocon Ointment",
      "form": "CREAM",
      "strength": "5g",
      "hsn_code": "30049012",
      "barcode": "8901234567890",
      "batch_no": "CRM-8812",
      "expires_at": "2027-04-30T00:00:00Z",
      "shelf_location": "Fridge-2",
      "purchase_qty_boxes": 5,
      "units_per_box": 1,
      "pricing": {
        "mrp": 120.00,
        "purchase_price": 95.00,
        "discount": 0.00,
        "discount_type": "FLAT",
        "selling_price": 120.00,
        "total_price": 475.00
      }
    }
  ]
}

$$\text{Total Price} = (\text{Purchase Price} - \text{Distributor Discount}) \times \text{Quantity of Boxes Purchased}$$

Backend Calculations Run Automatically on Save

When the Go server receives this payload, it doesn't just save it blindly. It runs these automated background functions:

Total Individual Units Calculation:
The database tracks inventory counts at the single loose unit (e.g., individual tablet) level to support cut-strip sales.

$$\text{CurrentStockUnits} = \text{purchase\_qty\_boxes} \times \text{units\_per\_box}$$

Example: 10 boxes $\times$ 15 tablets = 150 loose units saved under current_stock_units in MedicineInventory.

Stock Movement Logging:
For every item in the invoice, the backend automatically generates an entry in the MedicineStockMovements table with:

MovementType: "GRN_STOCK_INTAKE"

QtyChanged: +150 (in individual units)

SourceType: "PURCHASE_INVOICE"



dispense_payload:
{
  "prescription_id":"",
  medicine_array:[
    {
       "medicine_id":"abcd",
       "dispensed_qty:0,
       "batch_no":"",
       "quantity:25,
       "expiry_date:"some date"
    },
    {
       "medicine_id":"abcd",
       "dispensed_qty:0,
       "batch_no":"",
       "quantity:15,
       "expiry_date:"near date"
    }

  ]
}

When the Go backend receives the dispense_payload, it performs its own authoritative math inside a SQL transaction:

Unit Price Validation: Fetches the true unit_price of the batch_id directly from inventory_batches.

Tax Compilation: Calculates local medical taxes (e.g., GST/VAT) using backend-configured rates.

True Invoice Totals: Sums up the calculated line items, subtracts authorized discounts, and derives the absolute total_amount_paid.

Audit Log Generation: Writes the correct calculated values to invoice_items and medicine_stock_movements.

[ Frontend Payload Received ]
             │
             ▼
[ Step 1: Query DB & Perform Authoritative Calculations ]
             │
             ▼
[ Step 2: Cross-Verify Backend Total vs Expected Total ]
  Formula: Backend Total == Expected Frontend Total
             │
             ├──► (Fail) ──► Revert Tx & Abort (Return Validation Error)
             └──► (Pass)
                   │
                   ▼
[ Step 3: Check Stock Availability (Select For Update Locks) ]
             │
             ├──► (Fail) ──► Revert Tx & Abort (Return Out-of-Stock Error)
             └──► (Pass)
                   │
                   ▼
[ Step 4: Initiate Payment Gateway / Generate Dynamic UPI QR ]
  (Ensures QR reflects validated, secure calculations with zero tampering)
                   │
                   ├──► (Payment Fails/Cancelled) ──► Cancel Tx & Release Locks
                   └──► (Payment Approved / Success)
                         │
                         ▼
[ Step 5: Save to Billing Tables & Decrement Inventory ]
  (Final Commit to 'invoices', 'invoice_items', 'medicine_stock_movements')

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