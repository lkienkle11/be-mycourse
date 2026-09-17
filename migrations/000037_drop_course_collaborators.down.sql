-- Restores course_collaborators' exact shape from migration 000016_course_management.up.sql.
-- Data is NOT restored: a DROP TABLE is not reversible for data. See this change's proposal.md
-- for why that is an accepted trade-off in dev. authorization_role_bindings remains the live
-- source of truth for collaborator access regardless of whether this down migration is applied.
CREATE TABLE course_collaborators (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role VARCHAR(16) NOT NULL DEFAULT 'EDITOR',
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT NULL
);

CREATE UNIQUE INDEX uix_course_collaborators_active
    ON course_collaborators (course_id, user_id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_course_collaborators_user_active
    ON course_collaborators (user_id)
    WHERE deleted_at IS NULL;
