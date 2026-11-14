ALTER TABLE departments
    ADD COLUMN IF NOT EXISTS facility_type_id uuid;

-- Match a department to a facility type only when its name already is that
-- facility type. Departments that do not name a facility type stay unset.
UPDATE departments AS d
SET facility_type_id = matched.facility_type_id,
    updated_at = now()
FROM (
    SELECT d2.id AS department_id, min(ft.id::text)::uuid AS facility_type_id
    FROM departments AS d2
    JOIN facility_types AS ft
      ON lower(btrim(d2.name)) = lower(btrim(ft.name))
      OR lower(regexp_replace(btrim(d2.name), '[^a-zA-Z0-9]+', '_', 'g')) = lower(btrim(ft.code))
    GROUP BY d2.id
    HAVING count(DISTINCT ft.id) = 1
) AS matched
WHERE d.id = matched.department_id
  AND d.facility_type_id IS NULL;

ALTER TABLE departments DROP CONSTRAINT IF EXISTS fk_departments_facility_type;

ALTER TABLE departments
    ADD CONSTRAINT fk_departments_facility_type
    FOREIGN KEY (facility_type_id) REFERENCES facility_types(id)
    ON UPDATE CASCADE ON DELETE RESTRICT;

-- Existing rows are one copy per organisation. Keep those links, fold
-- duplicate names into one master department, then drop organisation_id.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'departments'
          AND column_name = 'organisation_id'
    ) THEN
        CREATE TEMP TABLE department_canonical ON COMMIT DROP AS
        SELECT DISTINCT ON (lower(btrim(name)))
            id AS canonical_id,
            lower(btrim(name)) AS name_key
        FROM departments
        ORDER BY lower(btrim(name)), (deleted_at IS NOT NULL), created_at ASC NULLS LAST, id ASC;

        UPDATE departments AS d
        SET facility_type_id = shared.facility_type_id,
            updated_at = now()
        FROM (
            SELECT c.canonical_id, min(src.facility_type_id::text)::uuid AS facility_type_id
            FROM departments AS src
            JOIN department_canonical AS c
              ON c.name_key = lower(btrim(src.name))
            WHERE src.facility_type_id IS NOT NULL
            GROUP BY c.canonical_id
            HAVING count(DISTINCT src.facility_type_id) = 1
        ) AS shared
        WHERE d.id = shared.canonical_id
          AND d.facility_type_id IS NULL;

        INSERT INTO organisation_departments (
            organisation_id, department_id, status, created_at, updated_at
        )
        SELECT
            d.organisation_id,
            c.canonical_id,
            CASE WHEN d.is_active IS FALSE THEN 'inactive' ELSE 'active' END,
            COALESCE(d.created_at, now()),
            COALESCE(d.updated_at, now())
        FROM departments AS d
        JOIN department_canonical AS c
          ON c.name_key = lower(btrim(d.name))
        WHERE d.organisation_id IS NOT NULL
        ON CONFLICT (organisation_id, department_id) DO NOTHING;

        DELETE FROM organisation_departments AS od
        USING departments AS d
        JOIN department_canonical AS c
          ON c.name_key = lower(btrim(d.name))
        WHERE od.department_id = d.id
          AND d.id <> c.canonical_id
          AND EXISTS (
              SELECT 1
              FROM organisation_departments AS keep
              WHERE keep.organisation_id = od.organisation_id
                AND keep.department_id = c.canonical_id
          );

        UPDATE organisation_departments AS od
        SET department_id = c.canonical_id,
            updated_at = now()
        FROM departments AS d
        JOIN department_canonical AS c
          ON c.name_key = lower(btrim(d.name))
        WHERE od.department_id = d.id
          AND od.department_id IS DISTINCT FROM c.canonical_id;

        UPDATE users AS u
        SET department_id = c.canonical_id
        FROM departments AS d
        JOIN department_canonical AS c
          ON c.name_key = lower(btrim(d.name))
        WHERE u.department_id = d.id
          AND u.department_id IS DISTINCT FROM c.canonical_id;

        DELETE FROM departments AS d
        USING department_canonical AS c
        WHERE lower(btrim(d.name)) = c.name_key
          AND d.id <> c.canonical_id;

        ALTER TABLE departments DROP CONSTRAINT IF EXISTS fk_departments_organisation;
        DROP INDEX IF EXISTS idx_org_department_name;
        ALTER TABLE departments DROP COLUMN organisation_id;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_name ON departments (name);
