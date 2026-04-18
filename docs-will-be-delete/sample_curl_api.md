# E-Learning Platform API (cURL Collection)

This document consolidates all APIs of the enterprise-grade E-Learning platform into cURL commands.

You can directly import these commands into tools like:
- Postman (File → Import → Raw Text)
- Apifox / Apidog

---

## 1. Authentication & IAM

### 1.1 Login (Get Access Token)

```bash
curl --location --request POST 'http://localhost:3000/api/v1/auth/login' \
--header 'Content-Type: application/json' \
--data-raw '{
    "email": "student@example.com",
    "password": "Password123!"
}'
````

**Response:**

```json
{
  "accessToken": "eyJhbG...",
  "user": { "id": "uuid", "role": "STUDENT" }
}
```

---

### 1.2 Update User Role (Admin Only)

```bash
curl --location --request PUT 'http://localhost:3000/api/v1/admin/users/{user_id}/role' \
--header 'Authorization: Bearer <ADMIN_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "new_role": "INSTRUCTOR"
}'
```

---

## 2. Course Management (Instructor)

### 2.1 Create Course Draft

```bash
curl --location --request POST 'http://localhost:3000/api/v1/instructor/courses' \
--header 'Authorization: Bearer <INSTRUCTOR_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "title": "Enterprise Node.js Course",
    "category_id": "uuid-category",
    "price": 499000
}'
```

**Response:**

```json
{
  "course_id": "uuid",
  "edit_id": "uuid"
}
```

---

### 2.2 Create Section

```bash
curl --location --request POST 'http://localhost:3000/api/v1/instructor/edits/{edit_id}/sections' \
--header 'Authorization: Bearer <INSTRUCTOR_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "title": "Chapter 1: Backend Architecture",
    "order_index": 1
}'
```

---

### 2.3 Create Lesson (Text - JSONB)

```bash
curl --location --request POST 'http://localhost:3000/api/v1/instructor/sections/{section_id}/lessons' \
--header 'Authorization: Bearer <INSTRUCTOR_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "title": "Lesson 1: Fundamentals",
    "lesson_type": "TEXT",
    "order_index": 1,
    "content_data": {
        "blocks": [
            { "type": "header", "data": { "text": "Introduction", "level": 2 } },
            { "type": "paragraph", "data": { "text": "Lesson content..." } }
        ]
    },
    "estimated_duration_seconds": 300
}'
```

---

### 2.4 Create Lesson (Quiz)

```bash
curl --location --request POST 'http://localhost:3000/api/v1/instructor/sections/{section_id}/lessons' \
--header 'Authorization: Bearer <INSTRUCTOR_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "title": "Midterm Quiz",
    "lesson_type": "QUIZ",
    "order_index": 2,
    "quiz_settings": {
        "passing_score": 80,
        "time_limit_seconds": 900
    },
    "questions": [
        {
            "question_text": "What is Node.js?",
            "question_type": "SINGLE",
            "options": [
                { "text": "A programming language", "is_correct": false },
                { "text": "A runtime environment", "is_correct": true }
            ],
            "weight": 10
        }
    ]
}'
```

---

### 2.5 Submit Course for Review

```bash
curl --location --request POST 'http://localhost:3000/api/v1/instructor/edits/{edit_id}/submit' \
--header 'Authorization: Bearer <INSTRUCTOR_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "note_to_admin": "Updated quiz content"
}'
```

---

### 2.6 Create Coupon

```bash
curl --location --request POST 'http://localhost:3000/api/v1/instructor/coupons' \
--header 'Authorization: Bearer <INSTRUCTOR_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "code": "BLACKFRIDAY",
    "discount_type": "PERCENTAGE",
    "discount_value": 30,
    "usage_limit": 100,
    "target_course_ids": ["uuid-course-1"]
}'
```

---

## 3. Admin (Approval System)

### 3.1 Get Pending Courses

```bash
curl --location --request GET 'http://localhost:3000/api/v1/admin/approvals?status=PENDING' \
--header 'Authorization: Bearer <ADMIN_TOKEN>'
```

---

### 3.2 Approve Course

```bash
curl --location --request POST 'http://localhost:3000/api/v1/admin/approvals/{edit_id}/approve' \
--header 'Authorization: Bearer <ADMIN_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "approval_note": "High-quality content, approved."
}'
```

---

## 4. Public APIs

### 4.1 Get Course Outline (Optimized)

```bash
curl --location --request GET 'http://localhost:3000/api/v1/public/courses/{course_id}/outline' \
--header 'Accept: application/json'
```

---

## 5. Learning & Tracking (Student)

### 5.1 Create Order (Purchase Course)

```bash
curl --location --request POST 'http://localhost:3000/api/v1/orders' \
--header 'Authorization: Bearer <STUDENT_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "course_id": "uuid-course",
    "payment_method": "VNPAY",
    "coupon_code": "BLACKFRIDAY"
}'
```

---

### 5.2 Sync Progress (Version Migration)

```bash
curl --location --request POST 'http://localhost:3000/api/v1/learning/enrollments/{enrollment_id}/sync' \
--header 'Authorization: Bearer <STUDENT_TOKEN>'
```

---

### 5.3 Track Lesson Progress

```bash
curl --location --request PUT 'http://localhost:3000/api/v1/learning/lessons/{lesson_id}/progress' \
--header 'Authorization: Bearer <STUDENT_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "tracking_data": {
        "played_seconds": 120,
        "scrolled_percentage": 0
    },
    "is_completed": false
}'
```

---

### 5.4 Submit Quiz

```bash
curl --location --request POST 'http://localhost:3000/api/v1/learning/lessons/{lesson_id}/quiz-submit' \
--header 'Authorization: Bearer <STUDENT_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "answers": [
        { "question_id": "uuid-q1", "selected_option_ids": ["uuid-opt-1"] },
        { "question_id": "uuid-q2", "selected_option_ids": ["uuid-opt-3", "uuid-opt-4"] }
    ]
}'
```

**Response:**

```json
{
  "score": 85,
  "passed": true
}
```

---

## 6. Community Interactions

### 6.1 Create Course Review

```bash
curl --location --request POST 'http://localhost:3000/api/v1/courses/{course_id}/reviews' \
--header 'Authorization: Bearer <STUDENT_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "rating": 5,
    "content": "Great course!"
}'
```

---

### 6.2 Reply to Review (Threaded Comment)

```bash
curl --location --request POST 'http://localhost:3000/api/v1/reviews/{review_id}/replies' \
--header 'Authorization: Bearer <TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "parent_reply_id": null,
    "content": "Thanks for your feedback @User",
    "tagged_user_ids": ["uuid-user-A"]
}'
```

---

## 7. System Settings (SysAdmin)

### 7.1 Update Commission Rate

```bash
curl --location --request PUT 'http://localhost:3000/api/v1/sysadmin/settings/COMMISSION_RATE' \
--header 'Authorization: Bearer <SYSADMIN_TOKEN>' \
--header 'Content-Type: application/json' \
--data-raw '{
    "value": { "rate": 0.3 }
}'
```

---

## Notes

* Replace all `{uuid}` placeholders with actual database values when testing.
* This API set covers the **full critical data flow** of the platform:

  * Course creation → approval → publishing
  * Purchase → enrollment → learning → tracking
  * Community interactions
  * System configuration

