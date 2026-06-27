#### Step 1: The Prescribing Interface (Doctor Module)
* **Action:** The clinician inputs the medication name, strength, and intended volume using a highly resilient autocomplete system or clean text entry.
* **Payload:** The system creates a static prescription ledger containing the text definitions. 
* **Output:** A cryptographic, short-form reference ID (e.g., `TX-8942`) or standard QR code is issued to the patient's mobile interface or paper checkout pass.

#### Step 2: The Retrieval Interface (Pharmacy Module)
* **Action:** The internal pharmacist scans or types the reference ID into the Billing Interface.
* **Hydration:** The interface pulls down the unpriced text list of prescribed medicines, dynamically structuring them as empty form rows requiring immediate real-world validation.

#### Step 3: Real-Time Checkout Compilation (Pharmacist Action)
* **Action:** The pharmacist reaches onto the physical shelf, reads the external labeling on the box/package, and enters the active constraints into the system. The transaction finishes instantly upon payment capture, firing an event to clear the prescription token.

---

# V1 Billing Feature: Simple Screen Design Specification[cite: 1]

This document outlines how the pharmacy checkout screens look and behave for the human operator, completely removing the need for a complex inventory setup[cite: 1].

---

## The Two Billing Screens[cite: 1]

The app automatically changes the checkout screen layout based on what the doctor prescribed[cite: 1]. The pharmacist will only ever see one of these two simple entry forms[cite: 1]:

### 1. The "Tablet" Screen (For Pills, Capsules, and Strips)[cite: 1]
* **When it appears:** Used when a doctor prescribes a specific number of individual pills (e.g., 14 tablets), but the price tag is printed on the whole box or strip[cite: 1].
* **What the pharmacist fills in:** Two blank fields[cite: 1].
    1. **Box/Strip Price:** `[ ₹ ______ ]`[cite: 1]
    2. **Total Tablets in Box/Strip:** `[ ______ ]`[cite: 1]
* **How it works:** If a patient needs 14 tablets, the pharmacist looks at the box, types **₹350** for the price, and **10** for the total tablet count[cite: 1]. The app automatically calculates that one tablet costs ₹35, and updates the line item total to **₹490**[cite: 1].

### 2. The "Whole Pack" Screen (For Syrups, Creams, Ointments, and Drops)[cite: 1]
* **When it appears:** Used when a patient always buys the complete container (e.g., a bottle of pediatric syrup or a tube of skin cream)[cite: 1]. 
* **What the pharmacist fills in:** One single blank field[cite: 1].
    1. **Price per Bottle/Tube:** `[ ₹ ______ ]`[cite: 1]
* **How it works:** If a doctor prescribes a bottle of syrup, the pharmacist looks at the bottle label, types the container price directly into the box, and the app instantly updates the bill[cite: 1]. There is no division or extra counting required[cite: 1].

---

## Behind-the-Scenes Logic Summary[cite: 1]
* **Smart Filtering:** The system checks the formulation type directly from the doctor's prescription list to decide which input screen to show[cite: 1].
* **No Manual Math:** The pharmacist never has to use a manual calculator or do mental division for cut strips at the counter[cite: 1].
* **Instant Saves (The V2 Transition):** The first time a price is entered for any medicine string, the app locks it into a local clinic memory cache[cite: 1]. The next time that exact medicine is prescribed, the price boxes auto-fill automatically, making checkout faster over time without manual inventory entry[cite: 1].

+-------------------------------------------------------------------------------+
| PRESCRIBED: Tab. Amoxicillin 500mg  |  QTY REQUIRED: 14 Tablets               |
+-------------------------------------------------------------------------------+
|  [Step 1: Check Box]                 [Step 2: Enter Count]                    |
|  Enter Total Box/Strip MRP:          Enter Total Tablets inside that Package:  |
|  ₹ [  350.00  ]                      [  10  ] Tablets                         |
+-------------------------------------------------------------------------------+
|  MATH ENGINE OUTPUT (Auto-Calculated Real-Time):                             |
|  Unit Price: ₹35.00/Tab  |  Total Line Item Charge (14 * 35): ₹490.00        |
+-------------------------------------------------------------------------------+
|  [ ] Override Total Directly (Manual Override Input: ₹ [________])            |
+-------------------------------------------------------------------------------+

### 4. Mathematical Model & Database Architecture

To ensure the backend service remains completely independent of an inventory master list, all calculations must be derived purely from variables captured over the network request during the checkout invocation.

#### The Mathematical Engine
Let:
* $Q_p$ = Total individual units prescribed by the clinician.
* $M_b$ = Maximum Retail Price (MRP) stamped onto the active physical box container.
* $U_b$ = Total individual countable units contained within that specific container package box.
* $C_l$ = Computed Line-Item Price to be charged to the patient.
* $O_m$ = Optional manual pricing override value entered by the operator.

The logic engine resolves $C_l$ using the following functional cascade:

$$C_l = \begin{cases} 
O_m & \text{if } O_m \neq \text{null} \\
\left( \frac{M_b}{U_b} \right) \times Q_p & \text{if } O_m = \text{null} 
\end{cases}$$

#### Minimalist Database Schema (PostgreSQL DDL)
The architecture explicitly discards tables such as `inventory_stocks`, `batch_ledgers`, or `product_sku_variants`. It preserves only transaction records:

```sql
-- 1. Prescriptions Table (Captures intent from Doctor)
CREATE TABLE prescriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id UUID NOT NULL,
    token_code VARCHAR(12) UNIQUE NOT NULL, -- e.g., 'TX-8942'
    doctor_id UUID NOT NULL,
    patient_id UUID NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING',  -- PENDING, DISPENSED, EXPIRED
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Prescription Line Items (Text entries, zero relational constraints to an inventory table)
CREATE TABLE prescription_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prescription_id UUID REFERENCES prescriptions(id) ON DELETE CASCADE,
    medicine_name_string TEXT NOT NULL,     -- Free-text/autocomplete capture
    prescribed_quantity INT NOT NULL,       -- e.g., 30 (Tablets)
    dosage_instruction TEXT
);

-- 3. Invoices Table (Captures final transaction parameters)
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prescription_id UUID REFERENCES prescriptions(id),
    clinic_id UUID NOT NULL,
    pharmacist_user_id UUID NOT NULL,
    subtotal_amount NUMERIC(10, 2) NOT NULL,
    tax_amount NUMERIC(10, 2) NOT NULL,
    discount_amount NUMERIC(10, 2) DEFAULT 0.00,
    final_paid_amount NUMERIC(10, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 4. Invoice Line Items (Captures the localized variables typed at counter)
CREATE TABLE invoice_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID REFERENCES invoices(id) ON DELETE CASCADE,
    medicine_name_string TEXT NOT NULL,
    units_prescribed INT NOT NULL,
    entered_box_mrp NUMERIC(10, 2),        -- Raw variable from physical package
    entered_pack_size INT,                 -- Raw variable from physical package
    is_manually_overridden BOOLEAN DEFAULT FALSE,
    computed_line_total NUMERIC(10, 2) NOT NULL -- Permanent audit record of final charge
);

Real-World Edge Case Handlers & FallbacksEdge 
Case A: The "Mixed Batch / Asymmetric Pricing" ConflictScenario: A patient requires 30 tablets. The pharmacist extracts an older open strip of 10 tablets (stamped with an older batch price of ₹100 per 10-pack) and takes a newer fresh box of 20 tablets from the shelf (stamped with an updated price inflation of ₹120 per 10-pack).System Resolution: The UI provides an instantaneous inline split-action option. The pharmacist can click "Add Split Batch" on that specific item row, creating two identical processing sub-rows:Sub-row 1: Qty 10 $\rightarrow$ Enter MRP: 100, Pack Size: 10Sub-row 2: Qty 20 $\rightarrow$ Enter MRP: 120, Pack Size: 10The system aggregates both calculations transparently into a singular final transaction charge.Edge 
Case B: Extreme Custom / Fragmented Compound MathScenario: Complex compounding mixtures, fractional dermatological liquids, or loose unbranded surgical consumables where calculating via a rigid unit formula introduces rounding errors.System Resolution: The "Dispensed Value Override" checkbox completely uncouples the automatic mathematical parser. When clicked, all fields disappear except for a single entry field: [ Final Line Charge Amount ]. The pharmacist computes the total offline via their standard mental/calculator baseline, inputs the definitive value, and the engine bypasses internal multiplication logic entirely.

6. The V2 Evolution: Turning Invoices into Passive Master Assets
By deploying this specific system layout, the app does not remain permanently feature-restricted. Instead, it systematically crowdsources its own underlying master data layers passively during day-to-day use.

[ Invoice Completed ] ──> Capture string: "Tab. Augmentin 625"
                      ──> Save structural data profile: { MRP: 201.20, Pack_Size: 10 }
                      ──> Write to 'Clinic Local Cache Index'
                                    │
                                    ▼
[ Next Prescribed Order ] ──> Doctor types "Aug... " ──> Auto-suggests name
                          ──> Pharmacist UI looks up Local Cache 
                          ──> Auto-fills [201.20] and [10] instantly 
                              (Pharmacist only clicks "Confirm" without typing)