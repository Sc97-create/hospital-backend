DELETE FROM facility_types
WHERE code IN ('hospital', 'clinic')
   OR name IN ('Hospital', 'Clinic');

UPDATE facility_types
SET code = '#' || upper(substr(md5(random()::text || name), 1, 6)),
    updated_at = now()
WHERE code NOT LIKE '#______';

INSERT INTO facility_types (code, name)
VALUES
    ('#' || upper(substr(md5(random()::text || 'dentistry'), 1, 6)), 'Dentistry'),
    ('#' || upper(substr(md5(random()::text || 'general_clinic'), 1, 6)), 'General Clinic')
ON CONFLICT (name) DO NOTHING;
