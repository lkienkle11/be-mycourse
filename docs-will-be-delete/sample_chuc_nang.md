# E-Learning Platform Feature Specification by Role

This document provides a detailed functional specification of the E-learning system based on user roles, including: Instructor, Student, Admin, and SysAdmin.

---

## 1. Instructor Role

The Instructor is responsible for creating, managing content, and selling courses:

* **Multi-version Course Management:** Instructors can clone courses to update content without affecting currently enrolled students.
* **Internal Versioning Workflow:** Manage course lifecycle through "Draft -> Published -> Readonly".
* **Independent Editing:** Instructors edit drafts independently and can reuse older versions to create entirely new courses.
* **Default Versioning:** Set a default version displayed to users who have not yet enrolled.
* **Multi-format Content:** Create text-based lessons stored as JSONB (supporting chunking, LZ4 compression, and XSS protection).
* **Video Management:** Video lessons are managed via URLs with automatic duration extraction (or manual input for text/quiz).
* **Advanced Quiz System:** Create single-choice or multiple-choice quizzes. Configure timers, passing scores, and per-question weights. Supports Video Quizzes that display popup questions at specific timestamps.
* **Metadata Management:** Define course level, acquired skills, tools used, and learning outcomes.
* **Prerequisites:** Require students to complete specific courses before enrolling.
* **Co-instructors:** Allow multiple instructors to co-manage and teach a course.
* **Languages & Subtitles:** Declare supported languages and upload physical subtitle files (.srt, .vtt) for videos.
* **Coupon System:** Create percentage-based or fixed-amount discounts with usage limits and expiration dates. Coupons can apply to specific courses, categories, or course series.
* **Certificate Issuance:** Configure multiple certificate types based on completion criteria using dynamic HTML templates.
* **Dashboard & Analytics:** Access dashboards with charts for revenue, active students, and quiz pass rates.

---

## 2. Student Role

Students consume content, interact with the system, and track learning progress:

* **Real-time Progress Tracking:** Track scroll position for text, video watch time, and quiz interactions using debounce mechanisms to prevent excessive updates.
* **Resume Learning:** Automatically save and restore the last learning position for future sessions.
* **Progress Synchronization:** When a new course version is released, students can opt to migrate. The system maps and resumes progress using `sync_uid` (UUID) without data loss.
* **Progress Preservation:** If synchronization is declined, the system preserves progress on the old version.
* **Course Series (Learning Paths):** Students can enroll in grouped courses structured as long-term learning paths or specializations.
* **Reviews & Feedback:** Provide star ratings and comments for courses. Additionally, students can rate instructors (Instructor Ratings).
* **Threaded Discussions:** Support nested replies with self-referencing structure and user tagging.

---

## 3. Admin Role

Acts as a Business Administrator responsible for operations and content moderation:

* **Course Approval Queue:** Manage pending course edits awaiting approval.
* **Manual Approval Workflow:** Each course version must be manually approved with an approval note before becoming the default version.
* **Taxonomy Management:** Create and manage Categories, Tags, and Course Levels across the system.
* **Role Management:** Upgrade user roles (e.g., Student → Instructor → Admin).
* **Global Analytics:** Access system-wide statistics such as total revenue and active users.

---

## 4. SysAdmin Role

The highest level of authority, responsible for technical system control and security:

* **Secure Configuration Management:** Only SysAdmins can access and modify sensitive configurations such as API keys (Stripe, VNPay, PayPal), storage services (AWS S3, Cloudinary), and email systems (SMTP).
* **Admin Management:** Create and manage Admin accounts, assign responsibilities, and revoke Admin privileges when necessary.
* **Troubleshooting & Monitoring:** Access system logs (error logs), monitor webhook statuses, and handle failed data synchronization.
* **Global System Settings:** Configure platform-wide parameters such as instructor commission rates, video upload limits, and other operational constraints.