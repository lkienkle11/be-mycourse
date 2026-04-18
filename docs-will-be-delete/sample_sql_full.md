# PostgreSQL Schema Generation Prompt for AI Agents

## Role

You are a senior database architect specializing in PostgreSQL schema design for large-scale e-learning platforms.

---

## Task

Design a complete, production-grade PostgreSQL database schema that models an online learning system.

---

## Core Requirements

### 1. PostgreSQL Best Practices

Use advanced PostgreSQL features where appropriate:

- ENUM types for controlled vocabularies
- JSONB for flexible structured data
- UUID as primary keys (auto-generated)
- Constraints:
  - CHECK
  - UNIQUE
  - FOREIGN KEY with ON DELETE behavior
- Indexing strategies:
  - B-Tree
  - GIN (for JSONB)

---

### 2. Schema Domains

Organize the schema into logical domains:

#### User Management
- Users with roles: STUDENT, INSTRUCTOR, ADMIN

#### Course System
- Courses
- Course levels
- Categories
- Tags

#### Course Content
- Sections
- Lessons
- Lesson types (text, video, quiz)

#### Versioning System
- Course revisions (draft, published, readonly)

#### Course Relationships
- Instructors
- Prerequisites
- Metadata (skills, tools, outcomes)

#### Course Series
- Learning paths / bundles of courses

#### Lesson Content Types
- Text content (JSONB)
- Video content (with subtitles)
- Quiz content

#### Quiz System
- Questions
- Answers
- Attempts
- Video-based quizzes

#### Enrollment & Progress Tracking
- Course enrollments
- Series enrollments
- Section progress
- Lesson progress

#### Payments & Orders
- Orders
- Order items
- Payment methods and statuses

#### Coupons / Discounts
- Coupon system
- Applicable scopes (courses, categories, series)

#### Reviews & Ratings
- Course reviews
- Series reviews
- Instructor ratings
- Threaded replies

---

### 3. Data Integrity & Design

- Normalize data appropriately
- Use junction tables for many-to-many relationships
- Apply composite unique constraints where necessary
- Enforce referential integrity via foreign keys
- Avoid redundancy unless justified

---

### 4. Performance Optimization

- Add indexes for frequently queried columns:
  - user_id
  - course_id
  - status
  - slug
- Consider:
  - Partial indexes
  - Composite indexes
  - GIN indexes for JSONB
- Optimize query patterns for:
  - filtering
  - joins
  - sorting

---

### 5. Real-World Constraints

- Prices must be non-negative
- Ratings must be within a valid range (e.g., 1–5)
- Ensure logical constraints:
  - At least one of `course_id` or `series_id` in order items
- Prevent duplicate relationships

---

### 6. Extensibility & Maintainability

- Use clear and consistent naming conventions
- Separate concerns across tables
- Allow future expansion:
  - Localization
  - Analytics
  - Recommendation systems

---

## Output Requirements

- Output **pure PostgreSQL SQL DDL only**
- Start with:

```sql
SET search_path TO public;

-- ============================================================
-- 1. ENUMS (Chuẩn hóa Trạng thái & Phân loại)
-- ============================================================
CREATE TYPE user_role AS ENUM ('STUDENT', 'INSTRUCTOR', 'ADMIN');
CREATE TYPE course_status AS ENUM ('DRAFT', 'PENDING', 'APPROVED', 'REJECTED');
CREATE TYPE edit_status AS ENUM ('DRAFT', 'PUBLISHED', 'READONLY');
CREATE TYPE publish_status AS ENUM ('DRAFT', 'PREVIEW', 'PUBLISHED');
CREATE TYPE lesson_type AS ENUM ('TEXT', 'VIDEO', 'QUIZ');
CREATE TYPE question_type AS ENUM ('SINGLE', 'MULTIPLE');
CREATE TYPE discount_type AS ENUM ('PERCENTAGE', 'FIXED_AMOUNT');
CREATE TYPE coupon_status AS ENUM ('ACTIVE', 'INACTIVE');
CREATE TYPE enrollment_status AS ENUM ('ACTIVE', 'COMPLETED');
CREATE TYPE progress_status AS ENUM ('PENDING', 'IN_PROGRESS', 'COMPLETED');
CREATE TYPE payment_method AS ENUM ('CREDIT_CARD', 'PAYPAL', 'MOMO', 'VNPAY', 'BANK_TRANSFER', 'FREE');
CREATE TYPE payment_status AS ENUM ('PENDING', 'COMPLETED', 'FAILED', 'REFUNDED');

-- ============================================================
-- 2. NHÓM NGƯỜI DÙNG & PHÂN LOẠI CƠ BẢN
-- ============================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    avatar VARCHAR(500),
    role user_role NOT NULL DEFAULT 'STUDENT',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Bảng trình độ riêng biệt (Admin quản lý)
CREATE TABLE course_levels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    image_url VARCHAR(512) NOT NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- ============================================================
-- 3. NHÓM KHÓA HỌC LẺ (COURSES)
-- ============================================================
CREATE TABLE courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instructor_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    level_id UUID REFERENCES course_levels(id) ON DELETE RESTRICT,
    title VARCHAR(500) NOT NULL,
    slug VARCHAR(500) NOT NULL UNIQUE,
    description TEXT,
    thumbnail_url VARCHAR(1000),
    preview_video_url VARCHAR(1000),
    price DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (price >= 0),
    status course_status NOT NULL DEFAULT 'DRAFT',
    total_sections INT DEFAULT 0,
    total_lessons INT DEFAULT 0,
    estimated_duration_seconds INT DEFAULT 0,
    spoken_languages JSONB DEFAULT '[]',
    subtitle_languages JSONB DEFAULT '[]',
    published_edit_id UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

-- Metadata bổ trợ
CREATE TABLE course_skills (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    skill_title VARCHAR(200) NOT NULL,
    skill_description TEXT,
    UNIQUE (course_id, skill_title)
);

CREATE TABLE course_tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    tool_title VARCHAR(200) NOT NULL,
    UNIQUE (course_id, tool_title)
);

CREATE TABLE course_outcomes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    outcome_title TEXT NOT NULL,
    outcome_subtitle JSONB NOT NULL DEFAULT '[]'
);

-- Quản lý phiên bản (Self-Edit Revisions)
CREATE TABLE course_edits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    revision_number INT NOT NULL,
    status edit_status NOT NULL DEFAULT 'DRAFT',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    UNIQUE (course_id, revision_number)
);

ALTER TABLE courses ADD CONSTRAINT fk_courses_published_edit 
    FOREIGN KEY (published_edit_id) REFERENCES course_edits(id) ON DELETE SET NULL;

-- Các bảng liên kết trung gian
CREATE TABLE course_instructors (
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    instructor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (course_id, instructor_id)
);

CREATE TABLE course_prerequisites (
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    prerequisite_course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    PRIMARY KEY (course_id, prerequisite_course_id)
);

CREATE TABLE course_certificates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    template_data JSONB NOT NULL,
    criteria JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE course_categories (
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (course_id, category_id)
);

CREATE TABLE course_tags (
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (course_id, tag_id)
);

-- ============================================================
-- 4. NHÓM COURSE SERIES (LỘ TRÌNH/CHUYÊN NGÀNH)
-- ============================================================
CREATE TABLE course_series (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(500) NOT NULL,
    slug VARCHAR(500) NOT NULL UNIQUE,
    description TEXT,
    thumbnail_url VARCHAR(1000),
    level_id UUID REFERENCES course_levels(id) ON DELETE RESTRICT,
    price DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (price >= 0),
    status course_status NOT NULL DEFAULT 'DRAFT',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE course_series_courses (
    series_id UUID NOT NULL REFERENCES course_series(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
    order_index INT NOT NULL DEFAULT 0,
    PRIMARY KEY (series_id, course_id)
);

CREATE TABLE series_outcomes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id UUID NOT NULL REFERENCES course_series(id) ON DELETE CASCADE,
    outcome_title TEXT NOT NULL,
    outcome_subtitle JSONB NOT NULL DEFAULT '[]'
);

-- ============================================================
-- 5. CẤU TRÚC NỘI DUNG (SECTIONS & LESSONS)
-- ============================================================
CREATE TABLE sections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    edit_id UUID NOT NULL REFERENCES course_edits(id) ON DELETE CASCADE,
    sync_uid UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    order_index INT NOT NULL DEFAULT 0,
    estimated_duration_seconds INT DEFAULT 0,
    status publish_status NOT NULL DEFAULT 'PUBLISHED'
);

CREATE TABLE lessons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    section_id UUID NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
    sync_uid UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    type lesson_type NOT NULL,
    order_index INT NOT NULL DEFAULT 0,
    estimated_duration_seconds INT DEFAULT 0,
    status publish_status NOT NULL DEFAULT 'PUBLISHED'
);

-- Chi tiết nội dung bài học
CREATE TABLE lesson_texts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    content_data JSONB NOT NULL,
    page_order INT NOT NULL DEFAULT 1,
    UNIQUE (lesson_id, page_order)
);

ALTER TABLE lesson_texts ALTER COLUMN content_data SET COMPRESSION lz4;

CREATE TABLE lesson_videos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL UNIQUE REFERENCES lessons(id) ON DELETE CASCADE,
    video_url VARCHAR(1000) NOT NULL,
    duration_seconds INT NOT NULL CHECK (duration_seconds > 0)
);

CREATE TABLE lesson_video_subtitles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_video_id UUID NOT NULL REFERENCES lesson_videos(id) ON DELETE CASCADE,
    language_code VARCHAR(10) NOT NULL,
    language_name VARCHAR(100) NOT NULL,
    file_url VARCHAR(1000) NOT NULL,
    UNIQUE (lesson_video_id, language_code)
);

-- ============================================================
-- 6. HỆ THỐNG KIỂM TRA (QUIZ)
-- ============================================================
CREATE TABLE video_quizzes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_video_id UUID NOT NULL REFERENCES lesson_videos(id) ON DELETE CASCADE,
    appear_at_second INT NOT NULL CHECK (appear_at_second >= 0),
    passing_correct_answers INT NOT NULL DEFAULT 1 CHECK (passing_correct_answers >= 0)
);

CREATE TABLE video_quiz_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_quiz_id UUID NOT NULL REFERENCES video_quizzes(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    type question_type NOT NULL,
    order_index INT NOT NULL DEFAULT 0
);

CREATE TABLE video_quiz_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES video_quiz_questions(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE lesson_quizzes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lesson_id UUID NOT NULL UNIQUE REFERENCES lessons(id) ON DELETE CASCADE,
    background_image_url VARCHAR(1000),
    background_audio_url VARCHAR(1000),
    time_limit_seconds INT,
    passing_correct_answers INT NOT NULL DEFAULT 1 CHECK (passing_correct_answers >= 0)
);

CREATE TABLE quiz_questions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_id UUID NOT NULL REFERENCES lesson_quizzes(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    type question_type NOT NULL,
    time_limit_seconds INT,
    order_index INT NOT NULL DEFAULT 0,
    weight DECIMAL(5, 2) NOT NULL DEFAULT 1.00 CHECK (weight > 0)
);

CREATE TABLE quiz_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES quiz_questions(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT false
);

-- ============================================================
-- 7. HỆ THỐNG KHUYẾN MÃI (COUPONS)
-- ============================================================
CREATE TABLE coupons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instructor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code VARCHAR(50) NOT NULL UNIQUE,
    discount_type discount_type NOT NULL,
    discount_value DECIMAL(10, 2) NOT NULL CHECK (discount_value > 0),
    valid_from TIMESTAMP WITH TIME ZONE,
    valid_until TIMESTAMP WITH TIME ZONE,
    usage_limit INT,
    used_count INT DEFAULT 0,
    status coupon_status NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE coupon_courses (
    coupon_id UUID NOT NULL REFERENCES coupons(id) ON DELETE CASCADE,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    PRIMARY KEY (coupon_id, course_id)
);

CREATE TABLE coupon_categories (
    coupon_id UUID NOT NULL REFERENCES coupons(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (coupon_id, category_id)
);

CREATE TABLE coupon_series (
    coupon_id UUID NOT NULL REFERENCES coupons(id) ON DELETE CASCADE,
    series_id UUID NOT NULL REFERENCES course_series(id) ON DELETE CASCADE,
    PRIMARY KEY (coupon_id, series_id)
);

-- ============================================================
-- 8. HỆ THỐNG THANH TOÁN (PAYMENTS & ORDERS)
-- ============================================================
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    total_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (total_amount >= 0),
    discount_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (discount_amount >= 0),
    final_amount DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (final_amount >= 0),
    payment_method payment_method NOT NULL,
    payment_status payment_status NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    course_id UUID REFERENCES courses(id) ON DELETE RESTRICT,
    series_id UUID REFERENCES course_series(id) ON DELETE RESTRICT,
    price DECIMAL(10, 2) NOT NULL DEFAULT 0.00 CHECK (price >= 0),
    CHECK (course_id IS NOT NULL OR series_id IS NOT NULL)
);

-- ============================================================
-- 9. THEO DÕI TIẾN TRÌNH & HỌC TẬP (ENROLLMENTS)
-- ============================================================
CREATE TABLE series_enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    series_id UUID NOT NULL REFERENCES course_series(id) ON DELETE RESTRICT,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    status enrollment_status NOT NULL DEFAULT 'ACTIVE',
    enrolled_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    UNIQUE (user_id, series_id)
);

CREATE TABLE enrollments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE RESTRICT,
    current_edit_id UUID NOT NULL REFERENCES course_edits(id) ON DELETE RESTRICT,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    last_accessed_lesson_id UUID REFERENCES lessons(id) ON DELETE SET NULL,
    completed_sections_count INT NOT NULL DEFAULT 0,
    total_sections_count INT NOT NULL DEFAULT 0,
    completed_lessons_count INT NOT NULL DEFAULT 0,
    total_lessons_count INT NOT NULL DEFAULT 0,
    progress_percentage DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    status enrollment_status NOT NULL DEFAULT 'ACTIVE',
    enrolled_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    UNIQUE (user_id, course_id)
);

CREATE TABLE section_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id UUID NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
    section_id UUID NOT NULL REFERENCES sections(id) ON DELETE CASCADE,
    section_sync_uid UUID NOT NULL,
    completed_lessons_count INT NOT NULL DEFAULT 0,
    total_lessons_count INT NOT NULL DEFAULT 0,
    status progress_status NOT NULL DEFAULT 'PENDING',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    UNIQUE (enrollment_id, section_id)
);

CREATE TABLE lesson_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id UUID NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
    lesson_id UUID NOT NULL REFERENCES lessons(id) ON DELETE CASCADE,
    lesson_sync_uid UUID NOT NULL,
    status progress_status NOT NULL DEFAULT 'PENDING',
    tracking_data JSONB,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    UNIQUE (enrollment_id, lesson_id)
);

CREATE TABLE quiz_attempts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enrollment_id UUID NOT NULL REFERENCES enrollments(id) ON DELETE CASCADE,
    quiz_id UUID NOT NULL REFERENCES lesson_quizzes(id) ON DELETE CASCADE,
    correct_answers_count INT NOT NULL DEFAULT 0,
    is_passed BOOLEAN,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    completed_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE quiz_attempt_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attempt_id UUID NOT NULL REFERENCES quiz_attempts(id) ON DELETE CASCADE,
    question_id UUID NOT NULL REFERENCES quiz_questions(id) ON DELETE CASCADE,
    selected_answer_ids UUID[] NOT NULL DEFAULT '{}',
    time_taken_seconds INT NOT NULL DEFAULT 0,
    UNIQUE (attempt_id, question_id)
);

-- ============================================================
-- 10. ĐÁNH GIÁ CỘNG ĐỒNG (REVIEWS)
-- ============================================================
CREATE TABLE series_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id UUID NOT NULL REFERENCES course_series(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    UNIQUE (series_id, user_id)
);

CREATE TABLE course_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    UNIQUE (course_id, user_id)
);

CREATE TABLE review_replies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id UUID REFERENCES course_reviews(id) ON DELETE CASCADE,
    series_review_id UUID REFERENCES series_reviews(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    parent_reply_id UUID REFERENCES review_replies(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    tagged_user_ids UUID[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

CREATE TABLE instructor_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instructor_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    UNIQUE(instructor_id, student_id)
);

-- ============================================================
-- 11. CHỈ MỤC TỐI ƯU (INDEXES)
-- ============================================================

-- Khóa học, Phân loại & Series
CREATE INDEX idx_courses_instructor ON courses(instructor_id);
CREATE INDEX idx_courses_slug ON courses(slug);
CREATE INDEX idx_courses_status ON courses(status);
CREATE INDEX idx_courses_level_id ON courses(level_id);

CREATE INDEX idx_series_slug ON course_series(slug);
CREATE INDEX idx_series_level_id ON course_series(level_id);
CREATE INDEX idx_series_courses_order ON course_series_courses(series_id, order_index);

-- Cấu trúc bài học
CREATE INDEX idx_edits_course ON course_edits(course_id);
CREATE INDEX idx_sections_edit ON sections(edit_id, order_index);
CREATE INDEX idx_lessons_section ON lessons(section_id, order_index);

-- Khuyến mãi & Đơn hàng
CREATE INDEX idx_coupon_series ON coupon_series(series_id);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(payment_status);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_course_id ON order_items(course_id);
CREATE INDEX idx_order_items_series_id ON order_items(series_id);

-- Học tập & Tiến trình
CREATE INDEX idx_series_enrollments_user ON series_enrollments(user_id);
CREATE INDEX idx_enrollments_user ON enrollments(user_id);
CREATE INDEX idx_section_progress_enrollment ON section_progress(enrollment_id);
CREATE INDEX idx_lesson_progress_enrollment ON lesson_progress(enrollment_id);
CREATE INDEX idx_section_progress_sync ON section_progress(enrollment_id, section_sync_uid);
CREATE INDEX idx_lesson_progress_sync ON lesson_progress(enrollment_id, lesson_sync_uid);
CREATE INDEX idx_lesson_progress_tracking ON lesson_progress USING gin(tracking_data);
CREATE INDEX idx_quiz_attempts_enrollment ON quiz_attempts(enrollment_id);

-- Đánh giá
CREATE INDEX idx_reviews_course ON course_reviews(course_id);
CREATE INDEX idx_series_reviews_series ON series_reviews(series_id);
CREATE INDEX idx_replies_course_review ON review_replies(review_id);
CREATE INDEX idx_replies_series_review ON review_replies(series_review_id);

```
---

- Organize schema into clearly commented sections
- Use consistent formatting and indentation
- Optimization Phase

---
# After generating the schema:

- Review your own design

- Improve it by:
    Refining indexes
    Adjusting normalization/denormalization
    Improving constraints
    Leveraging advanced PostgreSQL features

- Goal
    - The final schema must be:
        Production-ready
        Scalable to millions of users
        Optimized for both read-heavy and write-heavy workloads
        Clean, maintainable, and extensible

- Notes
    Make reasonable assumptions where necessary
    Follow best practices from modern SaaS e-learning systems

---