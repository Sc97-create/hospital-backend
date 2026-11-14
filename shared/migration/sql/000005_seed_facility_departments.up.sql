INSERT INTO departments (name, description, facility_type_id, is_active)
SELECT seed.name, seed.description, ft.id, true
FROM (
    VALUES
        ('General Clinic', 'General Medicine', 'General consultations and common illnesses'),
        ('General Clinic', 'Vaccination', 'Immunisation services'),
        ('Dentistry', 'General Dentistry', 'Routine dental consultations'),
        ('Dentistry', 'Endodontics', 'Root canal treatments'),
        ('Dentistry', 'Orthodontics', 'Braces and teeth alignment'),
        ('Dentistry', 'Periodontics', 'Gum treatments'),
        ('Dentistry', 'Prosthodontics', 'Crowns, bridges and dentures'),
        ('Dentistry', 'Oral Surgery', 'Extractions and minor surgeries'),
        ('Dentistry', 'Paediatric Dentistry', 'Children''s dental care')
) AS seed(facility_name, name, description)
JOIN facility_types AS ft ON ft.name = seed.facility_name
ON CONFLICT (name) DO UPDATE
SET description = EXCLUDED.description,
    facility_type_id = EXCLUDED.facility_type_id,
    updated_at = now();
