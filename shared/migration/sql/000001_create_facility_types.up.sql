CREATE TABLE IF NOT EXISTS facility_types (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    code text NOT NULL,
    name text NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT facility_types_code_key UNIQUE (code),
    CONSTRAINT facility_types_name_key UNIQUE (name)
);

INSERT INTO facility_types (code, name)
VALUES
    ('#' || upper(substr(md5(random()::text || 'dentistry'), 1, 6)), 'Dentistry'),
    ('#' || upper(substr(md5(random()::text || 'general_clinic'), 1, 6)), 'General Clinic')
ON CONFLICT (name) DO NOTHING;
