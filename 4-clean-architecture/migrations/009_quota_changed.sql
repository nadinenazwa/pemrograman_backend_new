-- Migration 009: composite unique constraint for fan-out quota changed events
-- psql -U postgres -d praktikum_backend -f migrations/009_quota_changed.sql

ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS notifications_outbox_unique;

ALTER TABLE notifications
    DROP CONSTRAINT IF EXISTS notifications_outbox_user_unique;

ALTER TABLE notifications
    ADD CONSTRAINT notifications_outbox_user_unique UNIQUE (outbox_id, user_id);
