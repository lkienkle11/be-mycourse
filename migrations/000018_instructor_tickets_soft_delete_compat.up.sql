-- Backfill compatibility for instructor ticket tables on drifted DBs:
-- ensure deleted_at exists for soft-delete scope (activeScope in repos)

ALTER TABLE instructor_tickets
    ADD COLUMN IF NOT EXISTS deleted_at BIGINT;

ALTER TABLE instructor_ticket_messages
    ADD COLUMN IF NOT EXISTS deleted_at BIGINT;

DROP INDEX IF EXISTS idx_instructor_tickets_status;
CREATE INDEX IF NOT EXISTS idx_instructor_tickets_status
    ON instructor_tickets (status)
    WHERE deleted_at IS NULL;
