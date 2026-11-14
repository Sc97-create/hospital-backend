CREATE TABLE IF NOT EXISTS organisation_departments (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    department_id uuid NOT NULL,
    status text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organisation_departments_status_check
        CHECK (status IN ('active', 'inactive')),
    CONSTRAINT organisation_departments_org_dept_key
        UNIQUE (organisation_id, department_id),
    CONSTRAINT fk_organisation_departments_organisation
        FOREIGN KEY (organisation_id) REFERENCES organisations(id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_organisation_departments_department
        FOREIGN KEY (department_id) REFERENCES departments(id)
        ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_organisation_departments_organisation
    ON organisation_departments (organisation_id);

CREATE INDEX IF NOT EXISTS idx_organisation_departments_department
    ON organisation_departments (department_id);
