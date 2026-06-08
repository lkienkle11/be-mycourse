DROP INDEX IF EXISTS idx_instructor_tickets_status;
CREATE INDEX IF NOT EXISTS idx_instructor_tickets_status
    ON instructor_tickets (status);

ALTER TABLE instructor_ticket_messages DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE instructor_tickets DROP COLUMN IF EXISTS deleted_at;
