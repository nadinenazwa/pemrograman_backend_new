-- Migration 006: Index untuk cursor-based pagination students
-- Index ini cocok dengan ORDER BY created_at DESC, id DESC yang digunakan pada cursor pagination.
CREATE INDEX IF NOT EXISTS idx_students_cursor_pagination
    ON students (created_at DESC, id DESC);
