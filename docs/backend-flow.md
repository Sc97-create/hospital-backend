# Backend Architecture

## Overview

The application follows a layered architecture based on the Repository Pattern and Dependency Injection principles.

Primary goals:

* Separation of concerns
* Testability
* Maintainability
* Scalability
* Centralized dependency management

---

## High-Level Request Flow

```text
Client
  ↓
Controller
  ↓
Service
  ↓
Repository
  ↓
GORM
  ↓
PostgreSQL
```

### Layer Responsibilities

#### Controller Layer

Responsibilities:

* Accept HTTP requests
* Validate request payloads
* Extract path/query parameters
* Invoke appropriate service methods
* Return HTTP responses

Controllers should not contain business logic or database operations.

---

#### Service Layer

Responsibilities:

* Implement business rules
* Coordinate multiple repositories if required
* Handle validations that belong to the domain
* Manage transactional workflows

Services should not directly execute database queries.

---

#### Repository Layer

Responsibilities:

* Encapsulate all database access
* Execute CRUD operations
* Build database queries
* Abstract persistence details from services

Repositories interact with the database through GORM.

---

#### Persistence Layer

Technology: GORM

Responsibilities:

* ORM mapping
* Query execution
* Transaction handling
* Database connection management

Database:

* PostgreSQL

---

## Application Startup Flow

```text
Application Start
  ↓
Load Environment Configuration
  ↓
Initialize Database Connection
  ↓
Build Dependency Container
  ↓
Register Repositories
  ↓
Register Services
  ↓
Register Controllers
  ↓
Register Routes
  ↓
Start HTTP Server
```

---

## Dependency Management

A centralized Container is used for dependency initialization.

Responsibilities:

* Load application dependencies
* Initialize database connection
* Create repository instances
* Create service instances
* Wire dependencies together

Example dependency chain:

```text
Database
   ↓
Repository
   ↓
Service
   ↓
Controller
```

This ensures a single source of truth for dependency creation.

---

## Configuration Management

Configuration values are loaded from environment variables.

Source:

```text
.env
```

Examples:

```env
DB_HOST=
DB_PORT=
DB_USER=
DB_PASSWORD=
DB_NAME=

SERVER_PORT=
```

Configuration should never be hardcoded.

---

## Design Principles

### Repository Pattern

Database access is isolated within repositories.

Benefits:

* Easier testing
* Better separation of concerns
* Database implementation can evolve independently

---

### Dependency Injection

Dependencies are created during application startup and injected into consumers.

Benefits:

* Loose coupling
* Easier mocking
* Better maintainability

---

### Single Responsibility Principle (SRP)

Each component should have exactly one responsibility.

Examples:

| Component  | Responsibility            |
| ---------- | ------------------------- |
| Controller | HTTP handling             |
| Service    | Business logic            |
| Repository | Data access               |
| Config     | Configuration loading     |
| Container  | Dependency initialization |

---

## Typical Request Lifecycle

### Create Patient Example

```text
POST /patients
      ↓
PatientController.Create()
      ↓
PatientService.Create()
      ↓
PatientRepository.Create()
      ↓
GORM Insert
      ↓
PostgreSQL
      ↓
Response Returned
```

---

## Project Structure

```text
internal/

├── controller/
│   └── HTTP handlers
│
├── service/
│   └── Business logic
│
├── repository/
│   └── Database operations
│
├── model/
│   └── Domain models
│
├── container/
│   └── Dependency registration
│
├── config/
│   └── Environment configuration
│
└── database/
    └── GORM initialization
```

---

## Architectural Rules

1. Controllers must not access repositories directly.
2. Services must not contain HTTP-specific logic.
3. Repositories must not contain business logic.
4. All database operations must go through repositories.
5. Dependencies must be created through the container.
6. Configuration must be sourced from environment variables.
7. New features should follow the Controller → Service → Repository flow.

```
```
