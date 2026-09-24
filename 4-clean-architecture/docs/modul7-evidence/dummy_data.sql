DO $$
DECLARE
    i INT;
    v_owner_id INT;
BEGIN
    SELECT id INTO v_owner_id FROM users LIMIT 1;
    IF v_owner_id IS NULL THEN
        INSERT INTO users (username, email, password, role, is_active) VALUES ('dummyowner', 'dummy@dummy.com', 'dummy', 'admin', true) RETURNING id INTO v_owner_id;
    END IF;

    FOR i IN 1..5000 LOOP
        INSERT INTO students (nim, name, grade, is_active, owner_id, created_at)
        VALUES (
            '1000' || LPAD(i::text, 4, '0'),
            'Student ' || i,
            (random() * 100)::numeric(5,2),
            true,
            v_owner_id,
            NOW() - (i || ' minutes')::interval
        );
    END LOOP;
END $$;
