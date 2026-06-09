CREATE TABLE courses (
    id UUID PRIMARY KEY,
    owner_user_id UUID NOT NULL REFERENCES users (id),
    slug VARCHAR(255) NOT NULL,
    current_published_version_id UUID NULL,
    current_draft_version_id UUID NULL,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT NULL
);

CREATE UNIQUE INDEX uix_courses_slug_active
    ON courses (slug)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_courses_owner_active
    ON courses (owner_user_id)
    WHERE deleted_at IS NULL;

CREATE TABLE course_versions (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    version_no INTEGER NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'DRAFT',
    based_on_version_id UUID NULL REFERENCES course_versions (id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    short_description VARCHAR(500) NOT NULL DEFAULT '',
    about_course TEXT NOT NULL DEFAULT '',
    thumbnail_file_id UUID NULL REFERENCES media_files (id) ON DELETE SET NULL,
    preview_video_file_id UUID NULL REFERENCES media_files (id) ON DELETE SET NULL,
    course_level_id UUID NULL REFERENCES course_levels (id) ON DELETE SET NULL,
    course_topic_id UUID NULL REFERENCES course_topics (id) ON DELETE SET NULL,
    row_version BIGINT NOT NULL DEFAULT 1,
    submitted_by_user_id UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    submitted_at BIGINT NULL,
    approved_by_user_id UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    approved_at BIGINT NULL,
    rejected_by_user_id UUID NULL REFERENCES users (id) ON DELETE SET NULL,
    rejected_at BIGINT NULL,
    rejection_reason TEXT NOT NULL DEFAULT '',
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT NULL
);

CREATE UNIQUE INDEX uix_course_versions_course_version_no_active
    ON course_versions (course_id, version_no)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_course_versions_course_status_active
    ON course_versions (course_id, status)
    WHERE deleted_at IS NULL;

ALTER TABLE courses
    ADD CONSTRAINT fk_courses_current_published_version
    FOREIGN KEY (current_published_version_id) REFERENCES course_versions (id) ON DELETE SET NULL;

ALTER TABLE courses
    ADD CONSTRAINT fk_courses_current_draft_version
    FOREIGN KEY (current_draft_version_id) REFERENCES course_versions (id) ON DELETE SET NULL;

CREATE TABLE course_version_tags (
    course_version_id UUID NOT NULL REFERENCES course_versions (id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (course_version_id, tag_id)
);

CREATE TABLE course_version_skills (
    course_version_id UUID NOT NULL REFERENCES course_versions (id) ON DELETE CASCADE,
    skill_id UUID NOT NULL REFERENCES course_skills (id) ON DELETE CASCADE,
    PRIMARY KEY (course_version_id, skill_id)
);

CREATE TABLE course_version_outcomes (
    course_version_id UUID NOT NULL REFERENCES course_versions (id) ON DELETE CASCADE,
    outcome_id UUID NOT NULL REFERENCES course_outcomes (id) ON DELETE CASCADE,
    PRIMARY KEY (course_version_id, outcome_id)
);

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

CREATE TABLE course_sections (
    id UUID PRIMARY KEY,
    stable_id UUID NOT NULL,
    course_version_id UUID NOT NULL REFERENCES course_versions (id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    order_index INTEGER NOT NULL DEFAULT 0,
    row_version BIGINT NOT NULL DEFAULT 1,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT NULL
);

CREATE UNIQUE INDEX uix_course_sections_stable_per_version_active
    ON course_sections (course_version_id, stable_id)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uix_course_sections_order_per_version_active
    ON course_sections (course_version_id, order_index)
    WHERE deleted_at IS NULL;

CREATE TABLE course_lessons (
    id UUID PRIMARY KEY,
    stable_id UUID NOT NULL,
    course_version_id UUID NOT NULL REFERENCES course_versions (id) ON DELETE CASCADE,
    section_id UUID NOT NULL REFERENCES course_sections (id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    order_index INTEGER NOT NULL DEFAULT 0,
    row_version BIGINT NOT NULL DEFAULT 1,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT NULL
);

CREATE UNIQUE INDEX uix_course_lessons_stable_per_version_active
    ON course_lessons (course_version_id, stable_id)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uix_course_lessons_order_per_section_active
    ON course_lessons (section_id, order_index)
    WHERE deleted_at IS NULL;

CREATE TABLE course_sub_lessons (
    id UUID PRIMARY KEY,
    stable_id UUID NOT NULL,
    course_version_id UUID NOT NULL REFERENCES course_versions (id) ON DELETE CASCADE,
    lesson_id UUID NOT NULL REFERENCES course_lessons (id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL DEFAULT '',
    kind VARCHAR(16) NOT NULL,
    is_preview BOOLEAN NOT NULL DEFAULT FALSE,
    order_index INTEGER NOT NULL DEFAULT 0,
    row_version BIGINT NOT NULL DEFAULT 1,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT NULL
);

CREATE UNIQUE INDEX uix_course_sub_lessons_stable_per_version_active
    ON course_sub_lessons (course_version_id, stable_id)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX uix_course_sub_lessons_order_per_lesson_active
    ON course_sub_lessons (lesson_id, order_index)
    WHERE deleted_at IS NULL;

CREATE TABLE course_sub_lesson_videos (
    sub_lesson_id UUID PRIMARY KEY REFERENCES course_sub_lessons (id) ON DELETE CASCADE,
    media_file_id UUID NOT NULL REFERENCES media_files (id) ON DELETE RESTRICT,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT)
);

CREATE TABLE course_sub_lesson_texts (
    sub_lesson_id UUID PRIMARY KEY REFERENCES course_sub_lessons (id) ON DELETE CASCADE,
    content_delta JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT)
);

CREATE TABLE course_sub_lesson_quizzes (
    sub_lesson_id UUID PRIMARY KEY REFERENCES course_sub_lessons (id) ON DELETE CASCADE,
    prompt TEXT NOT NULL DEFAULT '',
    allow_multiple BOOLEAN NOT NULL DEFAULT FALSE,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT)
);

CREATE TABLE course_sub_lesson_quiz_options (
    id UUID PRIMARY KEY,
    sub_lesson_id UUID NOT NULL REFERENCES course_sub_lesson_quizzes (sub_lesson_id) ON DELETE CASCADE,
    option_key UUID NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    is_correct BOOLEAN NOT NULL DEFAULT FALSE,
    order_index INTEGER NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT)
);

CREATE UNIQUE INDEX uix_course_sub_lesson_quiz_options_order
    ON course_sub_lesson_quiz_options (sub_lesson_id, order_index);

CREATE TABLE course_edit_leases (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    course_version_id UUID NOT NULL REFERENCES course_versions (id) ON DELETE CASCADE,
    resource_type VARCHAR(32) NOT NULL,
    resource_stable_id UUID NOT NULL,
    holder_user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    lease_token UUID NOT NULL,
    expires_at BIGINT NOT NULL,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT)
);

CREATE UNIQUE INDEX uix_course_edit_leases_resource
    ON course_edit_leases (course_version_id, resource_type, resource_stable_id);

CREATE INDEX idx_course_edit_leases_holder
    ON course_edit_leases (holder_user_id, expires_at);

CREATE TABLE course_enrollments (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL REFERENCES courses (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    current_version_id UUID NOT NULL REFERENCES course_versions (id) ON DELETE RESTRICT,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT NULL
);

CREATE UNIQUE INDEX uix_course_enrollments_active
    ON course_enrollments (course_id, user_id)
    WHERE deleted_at IS NULL;

CREATE TABLE course_progress_items (
    id UUID PRIMARY KEY,
    enrollment_id UUID NOT NULL REFERENCES course_enrollments (id) ON DELETE CASCADE,
    stable_content_id UUID NOT NULL,
    content_type VARCHAR(24) NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'NOT_STARTED',
    score NUMERIC(5,2) NOT NULL DEFAULT 0,
    quiz_attempt JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_interacted_at BIGINT NULL,
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    updated_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT),
    deleted_at BIGINT NULL
);

CREATE UNIQUE INDEX uix_course_progress_items_active
    ON course_progress_items (enrollment_id, stable_content_id)
    WHERE deleted_at IS NULL;
