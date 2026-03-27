# Backend Architecture

## Overview

The backend follows Clean Architecture with explicit layers:

- `delivery/http`: Gin handlers, route setup, and middlewares.
- `service`: business logic and application rules.
- `repository`: external adapters (Postgres, S3, cache).
- `core/domain`: enterprise entities and invariants.
- `core/ports`: contracts for repositories/services.

## Data Flow

Client request path:

1. `Router` receives HTTP request.
2. `Middleware` handles auth, RBAC, logging, validation.
3. `Handler` parses input DTOs and calls service interface.
4. `Service` applies business rules and transactional logic.
5. `Repository` executes persistence operations.
6. Database returns data back through the same chain.

## Dependency Injection

- `cmd/api/main.go` is the composition root.
- Concrete adapters (`repository`) are constructed first.
- Services receive interfaces from `core/ports`.
- Handlers depend only on service interfaces.
- This keeps domain logic testable and independent from frameworks.
