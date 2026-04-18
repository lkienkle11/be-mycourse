
# E-Learning Platform Folder Structure (DDD + Role-Based Architecture)

This document defines the project folder structure for the E-learning platform, based on the database design, API layer, and data flow architecture.

The structure follows a **Domain-Driven Design (DDD)** combined with **Role-Based modularization**. This approach (Modular Monolith) allows a large-scale system to be decomposed into independent, manageable modules, making it easier to scale, maintain, and distribute work across teams.

---

## 📂 Project Structure Overview

```text
📦 e-learning-platform
 ┣ 📂 src
 ┃ ┣ 📂 core
 ┃ ┃ ┗ Shared core components across the system
 ┃ ┃
 ┃ ┣ 📂 infrastructure
 ┃ ┃ ┗ External integrations (DB, Cache, Storage, Payment)
 ┃ ┃
 ┃ ┗ 📂 modules
 ┃
 ┃   ┣ 📂 IAM (Identity & Access Management)
 ┃   ┃ ┣ 📂 auth
 ┃   ┃ ┃ ┗ Login, Register, Forgot Password, Refresh Token
 ┃   ┃ ┣ 📂 users
 ┃   ┃ ┃ ┗ Profile, Change Password, Avatar
 ┃   ┃ ┗ 📂 rbac
 ┃   ┃   ┗ Dynamic Role & Permission management (Admin/SysAdmin)
 ┃   ┃
 ┃   ┣ 📂 course_management (Core Domain)
 ┃   ┃ ┣ 📂 instructor [ROLE: INSTRUCTOR]
 ┃   ┃ ┃ ┣ 📜 course.controller.ts
 ┃   ┃ ┃ ┣ 📜 section.controller.ts
 ┃   ┃ ┃ ┗ 📜 lesson.controller.ts
 ┃   ┃ ┣ 📂 admin [ROLE: ADMIN]
 ┃   ┃ ┃ ┗ 📜 approval.controller.ts
 ┃   ┃ ┗ 📂 public [ROLE: GUEST/ALL]
 ┃   ┃   ┗ 📜 catalog.controller.ts
 ┃   ┃
 ┃   ┣ 📂 learning_workspace
 ┃   ┃ ┣ 📂 enrollments
 ┃   ┃ ┣ 📂 progress
 ┃   ┃ ┣ 📂 quizzes
 ┃   ┃ ┗ 📂 certificates
 ┃   ┃
 ┃   ┣ 📂 interactions
 ┃   ┃ ┣ 📂 reviews
 ┃   ┃ ┗ 📂 comments
 ┃   ┃
 ┃   ┣ 📂 commerce
 ┃   ┃ ┣ 📂 promotions
 ┃   ┃ ┣ 📂 cart
 ┃   ┃ ┗ 📂 orders
 ┃   ┃
 ┃   ┣ 📂 taxonomy
 ┃   ┃ ┗ 📂 admin
 ┃   ┃
 ┃   ┗ 📂 system_settings
 ┃     ┗ 📂 sysadmin
 ┃
 ┣ 📜 package.json
 ┗ 📜 README.md
```

---

## 🔍 Module Breakdown

### 1. course_management (Core Domain - Most Complex)

#### instructor
- Draft creation  
- Course cloning  
- Lesson authoring (JSONB with XSS protection)  
- Section/Lesson structuring  
- Quiz configuration (timing, scoring weight)  

#### admin
- Retrieve pending course versions  
- Transition states: `PENDING → APPROVED / REJECTED`  
- Store approval notes  

#### public
- Browse default course versions  
- Lightweight course outline (optimized bandwidth)  

---

### 2. learning_workspace (Student Lifecycle Domain)

#### progress
- Receives real-time tracking signals:
  - Text scroll  
  - Video watch time  
- Stores JSONB tracking data  
- Handles synchronization via `sync_uid`  

#### quizzes
- Instant grading  
- Attempt history  
- Supports single/multiple choice logic  

#### certificates
- Dynamic certificate generation  
- Template binding  

---

### 3. interactions (Community Layer)

#### comments
- Threaded replies (self-referencing)  
- User tagging (`tagged_user_ids`)  
- Notification triggers  

#### reviews
- Course rating system  
- Instructor reputation scoring  

---

### 4. commerce (Monetization Layer)

#### promotions
- Percentage or fixed discount  
- Applicable to Course or Series  

#### orders
- Order lifecycle:
  - Create order  
  - Await payment  
  - Handle payment webhooks (VNPay, PayPal, Stripe)  
  - Unlock courses → enrollments  

---

### 5. IAM & system_settings

#### IAM / RBAC
- Fine-grained permission system  
- Role upgrades:
  - Student → Instructor → Admin  

#### system_settings
- Restricted to SYS_ADMIN  
- Runtime configuration:
  - Commission rate  
  - API keys (Storage, SMTP, Payment)  
- No server restart required  

---

## 💡 Architectural Advantages

### Maintainability & Debugging
- Clear module boundaries  
- Example: `learning_workspace/progress` for tracking issues  

### Role-Based Security by Design
- Separate folders: `instructor/`, `admin/`, `public/`  
- Clean middleware isolation  
- Prevents exposure of privileged APIs  

### Microservices-Ready
- Modules can be extracted independently  
- Example: `commerce` can scale separately  

---

## 🚀 Summary

- Strong domain separation (DDD)  
- Clear role-based boundaries  
- High scalability potential  
- Clean migration path to microservices  

Suitable for large-scale, enterprise-grade E-learning platforms.
