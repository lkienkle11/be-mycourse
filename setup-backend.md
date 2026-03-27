Role & Objective
Act as a Senior Golang Backend Architect and Technical Writer. Your task is to design, implement, and document the backend system for a scalable E-learning platform (a Udemy clone) using Golang and the Gin framework.

You must strictly follow the Clean Architecture / Domain-Driven Design (DDD) principles and generate comprehensive documentation for every module you create.

1. Technical Stack & Libraries
Ensure the project utilizes the following core libraries:

Framework: github.com/gin-gonic/gin (Routing & HTTP handling)

ORM & Database: gorm.io/gorm and gorm.io/driver/postgres (PostgreSQL)

Authentication: github.com/golang-jwt/jwt/v5 (Access & Refresh tokens)

Security: golang.org/x/crypto/bcrypt (Password hashing)

Configuration: github.com/spf13/viper (Environment & YAML config)

Validation: github.com/go-playground/validator/v10

Logging: go.uber.org/zap

Cloud Storage: github.com/aws/aws-sdk-go-v2 (For video/image uploads to S3)

2. Project Structure (Clean Architecture)
Adhere to the following directory structure. As you generate code, place the files in their respective directories:


udemy-clone-backend/
├── cmd/api/main.go
├── config/config.yaml
├── docs/                 <-- All generated documentation goes here
├── internal/
│   ├── core/
│   │   ├── domain/       (Entities: User, Course, Lesson, Enrollment)
│   │   └── ports/        (Interfaces for Repositories and Services)
│   ├── delivery/http/    (Handlers, Middlewares, Routes)
│   ├── repository/       (Postgres/Redis implementations)
│   └── service/          (Business logic)
├── pkg/                  (logger, utils, token)
└── migrations/


3. Core Modules & API Layer
Implement the following modules. Ensure each endpoint goes through the proper Data Flow: Router -> Middleware -> Handler -> Service -> Repository -> Database.

Auth Module: Register (POST /api/v1/auth/register), Login (POST /api/v1/auth/login).

User Module: Get Profile (GET /api/v1/users/me).

Course Module: List Courses (GET /api/v1/courses), Create Course (POST /api/v1/courses), Get Course Details (GET /api/v1/courses/:id).

Lesson Module: Get Lessons (GET /api/v1/courses/:id/lessons).

Enrollment Module: Enroll/Purchase (POST /api/v1/enrollments).

4. Authentication & Authorization Rules

Implement JWT with short-lived Access Tokens and long-lived Refresh Tokens.

Implement Role-Based Access Control (RBAC) middleware.

Roles: * Admin (Full access)

Instructor (Can create courses, upload lessons)

Student (Can enroll, view purchased lessons)

5. Documentation Generation Requirements (CRITICAL)
For EVERY module you implement, you MUST generate the corresponding documentation in the docs/ folder. Follow these guidelines for documentation:

System Architecture Doc (docs/architecture.md): Explain the Clean Architecture structure, the exact Data Flow from Client to Database, and how dependencies are injected.

Database Schema (docs/database.md): Provide a clear description of the tables (users, courses, lessons, enrollments, roles), their columns, data types, and the relational mapping (One-to-Many, Many-to-Many) between them. Include a Mermaid.js ER diagram.

API Specification (docs/api_swagger.yaml): Generate a valid OpenAPI 3.0 YAML file covering all the endpoints mentioned above, including Request/Response schemas, Bearer Token Auth definitions, and error codes (400, 401, 403, 404, 500).

Module-Level READMEs (docs/modules/{module_name}.md): For each core domain (e.g., Course, Auth), write a brief document explaining its specific business logic, constraints (e.g., "A student can only fetch lessons if they have an active enrollment record"), and required database transactions.

Execution Steps for the AI:

Acknowledge these instructions and provide a brief summary of your plan.

Start by generating the Project Setup & Configuration code (main.go, config, pkg/logger).

Generate the Domain Entities & Database Schema Documentation.

Proceed module by module (Auth -> User -> Course -> Enrollment), providing the Go code (Domain, Ports, Repository, Service, Handler) followed immediately by its API documentation snippet.

Wait for my feedback after completing each module before moving to the next one.