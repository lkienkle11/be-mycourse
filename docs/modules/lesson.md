# Lesson Module

_Last audited: 2026-06-15._

There is still no standalone `internal/lesson/` package.

Lesson behavior now lives inside `internal/course/` as part of the versioned outline model:

- `course_lessons` stores first-level lessons inside a section
- `course_sub_lessons` stores second-level lesson items inside a lesson
- `course_sub_lesson_videos`, `course_sub_lesson_texts`, and `course_sub_lesson_quizzes` store type-specific content

## Outline hierarchy

```text
Course version
└── Section
    └── Lesson
        └── Sub-lesson
            ├── VIDEO
            ├── QUIZ
            └── TEXT
```

## Current behavior

- lessons are version-scoped, not shared across drafts and published versions
- lesson and sub-lesson ordering is explicit through `order_index`
- every section / lesson / sub-lesson has a stable UUID (`stable_id`) for progress migration across approved versions
- lesson edits use optimistic locking (`row_version`)
- lesson and sub-lesson edits can be coordinated through course edit leases
- outline API responses include `estimated_duration_ms` on sections, lessons, and sub-lessons (see **`docs/modules/course.md`** → Estimated duration)

## Learner visibility

- learners read lessons only from the currently published course version
- draft-only lesson changes are invisible to learners until admin/sysadmin approval
- `is_preview` exists on sub-lessons for preview-capable lesson items

## Route surface

The lesson APIs are exposed through the course route group, not through a separate lesson router:

- `POST /api/v1/courses/:courseId/lessons`
- `PATCH /api/v1/courses/:courseId/lessons/:lessonId`
- `DELETE /api/v1/courses/:courseId/lessons/:lessonId`
- `POST /api/v1/courses/:courseId/sections/:sectionId/lessons/reorder`
- `POST /api/v1/courses/:courseId/sub-lessons`
- `PATCH /api/v1/courses/:courseId/sub-lessons/:subLessonId`
- `DELETE /api/v1/courses/:courseId/sub-lessons/:subLessonId`
- `POST /api/v1/courses/:courseId/lessons/:lessonId/sub-lessons/reorder`

If the system later introduces a separate lesson bounded context, this document should be updated. Today the source of truth is `internal/course/`.
