-- =============================================================================
-- Migration 005: RBAC tables + owner_id on students
-- =============================================================================
-- Modul 6: Authorization & Role Based Access Control
--
-- Strategi backfill owner_id:
--   - Jika ada students dengan owner_id NULL tetapi tidak ada users → RAISE EXCEPTION
--   - Jika ada students dengan owner_id NULL dan ada users → backfill ke user pertama
--   - Jika tidak ada students → tambahkan kolom NOT NULL langsung
--
-- Migration ini idempotent: menggunakan IF NOT EXISTS dan ON CONFLICT DO NOTHING.
-- =============================================================================

-- === Step 1: Tabel roles ===
CREATE TABLE IF NOT EXISTS roles (
    name VARCHAR(50) PRIMARY KEY
);

INSERT INTO roles (name) VALUES ('admin'), ('staff'), ('user')
ON CONFLICT DO NOTHING;

-- === Step 2: Tabel permissions ===
CREATE TABLE IF NOT EXISTS permissions (
    name VARCHAR(100) PRIMARY KEY
);

INSERT INTO permissions (name) VALUES
    ('student:list'),
    ('student:read:any'),
    ('student:create'),
    ('student:update:any'),
    ('student:delete')
ON CONFLICT DO NOTHING;

-- === Step 3: Tabel role_permissions ===
CREATE TABLE IF NOT EXISTS role_permissions (
    role_name VARCHAR(50) NOT NULL REFERENCES roles(name),
    permission_name VARCHAR(100) NOT NULL REFERENCES permissions(name),
    PRIMARY KEY (role_name, permission_name)
);

-- Mapping permission ke role:
--   admin → semua 5 permission
--   staff → student:list, student:read:any, student:create
--   user  → tidak ada permission
INSERT INTO role_permissions (role_name, permission_name) VALUES
    ('admin', 'student:list'),
    ('admin', 'student:read:any'),
    ('admin', 'student:create'),
    ('admin', 'student:update:any'),
    ('admin', 'student:delete'),
    ('staff', 'student:list'),
    ('staff', 'student:read:any'),
    ('staff', 'student:create')
ON CONFLICT DO NOTHING;

-- === Step 4: Normalisasi users.role dan tambahkan FK ===
-- Normalisasi role yang tidak dikenal menjadi 'user' sebelum menambahkan FK.
-- Data users tidak dihapus, hanya role yang dinormalisasi.
UPDATE users SET role = 'user'
WHERE role NOT IN ('admin', 'staff', 'user');

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_role_fk;

ALTER TABLE users
    ADD CONSTRAINT users_role_fk FOREIGN KEY (role) REFERENCES roles(name);

-- === Step 5: Tambahkan owner_id ke students dengan fail-fast backfill ===
DO $$
DECLARE
    v_student_count INTEGER;
    v_first_user_id INTEGER;
BEGIN
    -- Tambahkan kolom owner_id jika belum ada (nullable sementara)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'students' AND column_name = 'owner_id'
    ) THEN
        ALTER TABLE students ADD COLUMN owner_id INTEGER REFERENCES users(id) ON DELETE RESTRICT;
    END IF;

    -- Hitung students yang belum punya owner_id
    SELECT COUNT(*) INTO v_student_count FROM students WHERE owner_id IS NULL;

    IF v_student_count > 0 THEN
        -- Cari user pertama sebagai owner data lama
        SELECT id INTO v_first_user_id FROM users ORDER BY id LIMIT 1;

        IF v_first_user_id IS NULL THEN
            RAISE EXCEPTION
                'Migration 005 ABORTED: % students have NULL owner_id but no users exist in the database. '
                'Create at least one user first, then re-run this migration.',
                v_student_count;
        END IF;

        -- Backfill: data students lama diasumsikan dibuat oleh user pertama di sistem.
        -- Keputusan ini didokumentasikan karena data historis tidak memiliki informasi owner.
        UPDATE students SET owner_id = v_first_user_id WHERE owner_id IS NULL;

        RAISE NOTICE 'Backfilled % students with owner_id = %', v_student_count, v_first_user_id;
    END IF;

    -- Sekarang aman untuk memaksa NOT NULL
    ALTER TABLE students ALTER COLUMN owner_id SET NOT NULL;
END $$;
