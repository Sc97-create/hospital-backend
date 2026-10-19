package labservice

import (
	"strconv"
	"strings"
)

// DefaultLabTests is the built-in catalog keyed by test code.
var DefaultLabTests = map[string]LabTemplate{
	"BIO-04": {
		TestCode:   "BIO-04",
		TestName:   "Fasting Blood Sugar (FBS)",
		Specimens:  []string{"Fluoride Plasma (Gray Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("2 hrs TAT"),
	},
	"BIO-12": {
		TestCode:   "BIO-12",
		TestName:   "Glycated Hemoglobin (HbA1c)",
		Specimens:  []string{"Whole Blood HPLC (Lavender)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("4 hrs TAT"),
	},
	"END-08": {
		TestCode:   "END-08",
		TestName:   "Fasting Serum Insulin",
		Specimens:  []string{"Serum (Red / Yellow Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("24 hrs TAT"),
	},
	"END-01": {
		TestCode:   "END-01",
		TestName:   "Thyroid Stimulating Hormone (TSH)",
		Specimens:  []string{"Serum (Red / Yellow Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("6 hrs TAT"),
	},
	"HEM-01": {
		TestCode:   "HEM-01",
		TestName:   "Complete Blood Count (CBC)",
		Specimens:  []string{"EDTA Whole Blood (Lavender)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("2 hrs TAT"),
	},
	"BIO-21": {
		TestCode:   "BIO-21",
		TestName:   "Lipid Profile (Full / Comprehensive)",
		Specimens:  []string{"Serum (Red / Yellow Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("4 hrs TAT"),
	},
	"BIO-15": {
		TestCode:   "BIO-15",
		TestName:   "Kidney Function Test (KFT)",
		Specimens:  []string{"Serum (Red / Yellow Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("4 hrs TAT"),
	},
	"BIO-16": {
		TestCode:   "BIO-16",
		TestName:   "Liver Function Test (LFT)",
		Specimens:  []string{"Serum (Red / Yellow Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("4 hrs TAT"),
	},
	"URI-01": {
		TestCode:   "URI-01",
		TestName:   "Urine Routine & Microscopy",
		Specimens:  []string{"Urine (Sterile Container)"},
		Type:       "urine_test",
		TATMinutes: parseTATMinutes("3 hrs TAT"),
	},
	"BIO-05": {
		TestCode:   "BIO-05",
		TestName:   "Postprandial Blood Sugar (PPBS)",
		Specimens:  []string{"Fluoride Plasma (Gray Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("2 hrs TAT"),
	},
	"BIO-06": {
		TestCode:   "BIO-06",
		TestName:   "Random Blood Sugar (RBS)",
		Specimens:  []string{"Fluoride Plasma (Gray Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("2 hrs TAT"),
	},
	"BIO-18": {
		TestCode:   "BIO-18",
		TestName:   "Serum Uric Acid",
		Specimens:  []string{"Serum (Red / Yellow Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("4 hrs TAT"),
	},
	"BIO-19": {
		TestCode:   "BIO-19",
		TestName:   "Serum Calcium",
		Specimens:  []string{"Serum (Red / Yellow Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("4 hrs TAT"),
	},
	"BIO-22": {
		TestCode:   "BIO-22",
		TestName:   "Electrolytes (Na / K / Cl)",
		Specimens:  []string{"Serum (Red / Yellow Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("3 hrs TAT"),
	},
	"BIO-25": {
		TestCode:   "BIO-25",
		TestName:   "Serum Amylase",
		Specimens:  []string{"Serum (Red / Yellow Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("6 hrs TAT"),
	},
	"BIO-26": {
		TestCode:   "BIO-26",
		TestName:   "Serum Lipase",
		Specimens:  []string{"Serum (Red / Yellow Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("6 hrs TAT"),
	},
	"HEM-02": {
		TestCode:   "HEM-02",
		TestName:   "Erythrocyte Sedimentation Rate (ESR)",
		Specimens:  []string{"EDTA Whole Blood (Lavender)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("2 hrs TAT"),
	},
	"HEM-03": {
		TestCode:   "HEM-03",
		TestName:   "Peripheral Smear",
		Specimens:  []string{"EDTA Whole Blood (Lavender)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("6 hrs TAT"),
	},
	"END-02": {
		TestCode:   "END-02",
		TestName:   "Free T3",
		Specimens:  []string{"Serum (Red / Yellow Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("8 hrs TAT"),
	},
	"END-03": {
		TestCode:   "END-03",
		TestName:   "Free T4",
		Specimens:  []string{"Serum (Red / Yellow Top)"},
		Type:       "blood_test",
		TATMinutes: parseTATMinutes("8 hrs TAT"),
	},
	"URI-02": {
		TestCode:   "URI-02",
		TestName:   "Urine Culture & Sensitivity",
		Specimens:  []string{"Urine (Sterile Container)"},
		Type:       "urine_test",
		TATMinutes: parseTATMinutes("48 hrs TAT"),
	},
}

// DefaultLabTestList returns the default catalog as a stable ordered slice.
func DefaultLabTestList() []LabTemplate {
	order := []string{
		"BIO-04", "BIO-12", "END-08", "END-01", "HEM-01", "BIO-21", "BIO-15", "BIO-16",
		"URI-01", "BIO-05", "BIO-06", "BIO-18", "BIO-19", "BIO-22", "BIO-25", "BIO-26",
		"HEM-02", "HEM-03", "END-02", "END-03", "URI-02",
	}
	out := make([]LabTemplate, 0, len(order))
	for _, code := range order {
		if t, ok := DefaultLabTests[code]; ok {
			out = append(out, t)
		}
	}
	return out
}

func parseTATMinutes(tat string) int {
	fields := strings.Fields(strings.TrimSpace(tat))
	if len(fields) == 0 {
		return 0
	}
	hours, err := strconv.Atoi(fields[0])
	if err != nil || hours < 0 {
		return 0
	}
	return hours * 60
}
