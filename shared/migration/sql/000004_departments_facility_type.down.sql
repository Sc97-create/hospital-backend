DROP INDEX IF EXISTS idx_departments_name;

ALTER TABLE departments DROP CONSTRAINT IF EXISTS fk_departments_facility_type;

ALTER TABLE departments DROP COLUMN IF EXISTS facility_type_id;
