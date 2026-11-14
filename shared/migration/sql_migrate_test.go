package migration

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestApplyVersionRunsOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := ensureSchemaMigrations(db); err != nil {
		t.Fatalf("schema_migrations: %v", err)
	}
	script := `
		CREATE TABLE facility_types (code text PRIMARY KEY, name text NOT NULL);
		INSERT INTO facility_types (code, name) VALUES ('hospital', 'Hospital');
	`
	if err := applyVersion(db, "000001_test", script); err != nil {
		t.Fatalf("apply: %v", err)
	}
	applied, err := appliedVersions(db)
	if err != nil {
		t.Fatalf("applied: %v", err)
	}
	if _, ok := applied["000001_test"]; !ok {
		t.Fatal("version was not recorded")
	}
	var count int64
	if err := db.Table("facility_types").Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("rows: got %d want 1", count)
	}
}

func TestUpFileIncludesFacilityTypes(t *testing.T) {
	names, err := upFileNames()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(names) != 5 || names[4] != "000005_seed_facility_departments.up.sql" {
		t.Fatalf("files: %#v", names)
	}
	if migrationVersion(names[0]) != "000001_create_facility_types" {
		t.Fatalf("version: %s", migrationVersion(names[0]))
	}
}

func TestFacilityDepartmentSeed(t *testing.T) {
	body, err := upSQL.ReadFile("sql/000005_seed_facility_departments.up.sql")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	script := string(body)
	for _, name := range []string{
		"General Clinic",
		"General Medicine",
		"Vaccination",
		"Dentistry",
		"General Dentistry",
		"Endodontics",
		"Orthodontics",
		"Periodontics",
		"Prosthodontics",
		"Oral Surgery",
		"Paediatric Dentistry",
	} {
		if !strings.Contains(script, name) {
			t.Fatalf("seed missing %q", name)
		}
	}
}

func TestDepartmentMigrationKeepsDoBlock(t *testing.T) {
	body, err := upSQL.ReadFile("sql/000004_departments_facility_type.up.sql")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	parts := splitStatements(string(body))
	if len(parts) != 6 {
		t.Fatalf("statements: %d", len(parts))
	}
	if !strings.Contains(parts[4], "DROP COLUMN organisation_id") {
		t.Fatalf("organisation column drop was split out of the DO block")
	}
}

func TestSplitStatementsKeepsDollarBody(t *testing.T) {
	parts := splitStatements("SELECT 1;\nDO $$\nBEGIN\n  PERFORM 1;\nEND $$;")
	if len(parts) != 2 {
		t.Fatalf("statements: %#v", parts)
	}
	if !strings.Contains(parts[1], "PERFORM 1;") {
		t.Fatalf("dollar body was split: %s", parts[1])
	}
}
