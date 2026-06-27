On-Ground Onboarding: Developer Guide for Invoice/ERP Data Scraping

This document outlines the engineering workflow for our "On-Ground Bootstrapping" strategy. Instead of loading a bloated, generic database of 400,000+ Indian medicines, we will dynamically build our medicine catalog by collecting physical/digital invoices or local ERP exports during clinic onboarding.

1. The Strategy: Hyper-Localized Data Loading

We want to keep our global database lean and regionalized.

A pediatric clinic in Mumbai uses a vastly different subset of medicines than an orthopedic clinic in Delhi.

Loading all data on Day 1 causes "search fatigue" for users.

Our Solution: We seed each new clinic's account with only their actual active inventory, extracted from their onboarding invoices or legacy systems.

2. Onboarding Data Input Formats

Our operations team will collect data in two formats. Developers must design pipelines to ingest both:

Format A: Legacy ERP Exports (Preferred)

Most pharmacies use legacy desktop billing software (Marg ERP, Vyapar, Tally, or Busy).

The File: A .csv, .xlsx, or .txt file representing their current "Product Master List" or "Stock Ledger".

The Good: Zero privacy concerns (it contains drug names, pack sizes, and MRPs, but usually no distributor purchase margins). 100% digital.

Developer Ingestion Plan:

Build a basic CSV/Excel importer script.

Map incoming columns dynamically (e.g., User selects which column is Medicine Name, Formulation, MRP, and Pack Size).

Run data cleaning and strip out local weird shorthand (e.g., convert TAB. DOLO 650MG to Dolo 650mg [TABLET]).

Format B: Physical Invoice Photos

If the clinic does not have an ERP or refuses to share the digital file, they will provide paper invoices from distributors.

The File: High-resolution smartphone photos or scans of bills.

The Trap: Pharmacists are protective of their wholesale cost prices. They will scratch out, blur, or fold the "Purchase Rate" column.

Developer Ingestion Plan (V1 vs V2):

V1 (Manual Data Entry Assist): We upload the raw images directly to an internal admin panel. A cheap data-entry contractor (or our ops team) types the medicine names, formulation, and pack sizes from the images directly into our clean admin UI.

V2 (AI Pipeline): Send the image to an OCR/Document-Intelligence API (like Google Cloud Document AI or Gemini Flash Vision). Instruct the prompt to completely ignore the price columns and only extract: [Medicine Name, Formulation, Pack Size, Brand/Manufacturer].

3. Data Cleansing & Normalization Rules (Critical)

To prevent our database from turning into a chaotic mess, all incoming data must pass through a strict cleaning filter before hit our database tables:

[ Raw Imported String ] ──> [ Normalization Filter ] ──> [ Database Insert/Upsert ]
"TAB. DOLO 650 MG"           1. Strip Prefix (TAB., INJ., CAP.)     "Dolo 650mg" (Formulation: TABLET)
"DOLO 650 STRIP OF 15"       2. Parse & Extract Pack Size           "Dolo 650mg" (Units per box: 15)
"dolo 650"                   3. Normalize Case & Spacing            "Dolo 650mg"


The Developer Ruleset:

Strip Common Prefixes/Suffixes: Remove text like Tab, Cap, Syp, Inj, Oint, Sachet from the name field and map them directly to the formulation enum field (TABLET, CAPSULE, SYRUP, INJECTION, CREAM).

De-duplicate using Salt/Fuzzy Matching: Check if Dolo 650mg already exists in the medicines table for that clinic before inserting.

Preserve Pack Size: If a row says Dolo 650 Strip of 15, strip the pack size (15) and save it to the database schema's units_per_box field, keeping the core medicine name clean as Dolo 650mg.

4. The "Network Effect" Master DB Consolidation

As we onboard clinics, our global dictionary gets progressively smarter.

Local Dynamic Insert: When Clinic A uploads their list, it writes directly to medicines with their specific clinic_id.

Global Verification (The "Master List"):

If a medicine name has been entered and verified across multiple clinics, our system flags it as a "Verified Global Drug".

Future clinics onboarding in the same geographic region can automatically search and match against this verified global list, reducing the need to ask for their invoices at all!

5. UI Requirements for Developers (Emergency Escape Hatch)

Even with invoices pre-loaded, a doctor or pharmacist will eventually need to prescribe/bill a brand-new medicine that wasn't on their invoices.

We must build an "On-the-Fly Creator" directly in the search bar:

If a search query returns 0 results, show a button: "+" Create custom medicine "{Input Name}".

Clicking it opens a 2-second dialog box: "What is the type? [Tablet, Syrup, Cream, Drops]".

Once selected, it instantly saves to their local catalog and lets them continue billing without breaking their work momentum.