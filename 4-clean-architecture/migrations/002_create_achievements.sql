CREATE TABLE IF NOT EXISTS achievements (
    id_achievement SERIAL PRIMARY KEY,
    nim_students VARCHAR(20) NOT NULL REFERENCES students(nim) ON DELETE CASCADE,
    nama_prestasi VARCHAR(150) NOT NULL,
    juara VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS achievements_nim_students_idx ON achievements (nim_students);