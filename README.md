# GoGym
![CI status](https://github.com/ctsdm/gogym/actions/workflows/ci.yml/badge.svg)

## Description

**GoGym** is a REST API for tracking workout sessions, exercises, sets, and reps. Built with Go and PostgreSQL, it provides a complete backend solution for fitness tracking applications with JWT authentication, role-based access control, and Prometheus metrics integration.

## Motivation

I built GoGym to create a flexible API that I can use to only track my workout sessions, but also to obtain metrics and statistics about my progression. The goal was to demonstrate proficiency in:

- Building RESTful APIs with **Go's standard library**.
- Implementing **JWT-based authentication** with refresh tokens.
- Designing a **PostgreSQL database** with proper relationships and migrations using **goose** and **sqlc**.
- Writing **testable, maintainable code** with dependency injection using **testcontainers**.
- Integrating **observability** through Prometheus metrics and with Grafana dashboards.
- Containerizing services with **Docker Compose**.

## Quick Start

### Prerequisites
- **Docker** and **Docker Compose**
- **Go 1.25+** (for development)

### 1. Configure Environment

These are the default values for `example.env` file in the project root:

```bash
# Server Configuration
SERVER_PORT=8080
SERVER_PORT_METRICS=9090

# Database Configuration
POSTGRES_USER=postgres
POSTGRES_PASSWORD=your_secure_password
POSTGRES_HOST_PORT=localhost:5433
POSTGRES_DB=gogym
POSTGRES_EXTERNAL_PORT=5433
DEV=1

# Authentication
JWT_SECRET=your_jwt_secret_key
JWT_DURATION=3600
REFRESH_TOKEN_DURATION=604800

# Admin User
ADMIN_USERNAME=admin
ADMIN_PASSWORD=your_admin_password

# Grafana (optional)
GRAFANA_ADMIN_PASSWORD=admin
```

### 2. Start Services

```bash
docker-compose up -d
```

This starts:
- **PostgreSQL** on port 5433
- **Prometheus** on port 9090
- **Grafana** on port 3000

### 3. Run the API

```bash
go run cmd/server/main.go
```

The API will be available at **http://localhost:8080**

Health check: **http://localhost:8080/health**

## Usage

### Tech Stack

- **Go 1.25.3** - Backend language
- **PostgreSQL** - Database with pgx/v5 driver
- **SQLC** - Type-safe SQL query generation
- **Goose** - Database migrations
- **JWT** - Authentication with refresh tokens
- **Prometheus** - Metrics and monitoring
- **Docker Compose** - Local development environment

### API Endpoints

#### Authentication
- `POST /api/v1/login` - Authenticate user and receive JWT tokens

#### Users
- `POST /api/v1/users` - Register a new user
- `GET /api/v1/users` - List all users *(admin only)*
- `GET /api/v1/users/{id}` - Get user details *(admin only)*

#### Workout Sessions
- `POST /api/v1/sessions` - Create a workout session
- `GET /api/v1/sessions` - List your sessions
- `GET /api/v1/sessions/{id}` - Get session details
- `PUT /api/v1/sessions/{id}` - Update session
- `DELETE /api/v1/sessions/{id}` - Delete session

#### Sets
- `POST /api/v1/sessions/{sessionID}/sets` - Add a set to a session
- `GET /api/v1/sets/{id}` - Get set details
- `PUT /api/v1/sets/{id}` - Update set
- `DELETE /api/v1/sets/{id}` - Delete set

#### Logs
- `POST /api/v1/sessions/{sessionID}/sets/{setID}/logs` - Log an exercise (weight, reps)
- `GET /api/v1/logs` - List your logs
- `PUT /api/v1/logs/{id}` - Update log
- `DELETE /api/v1/logs/{id}` - Delete log

#### Exercises
- `GET /api/v1/exercises` - Browse available exercises
- `GET /api/v1/exercises/{id}` - Get exercise details

#### Monitoring
- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics

### Project Structure

```
.
├── cmd/server/          # Application entry point
├── internal/
│   ├── api/            # HTTP handlers and routing
│   ├── auth/           # JWT and password hashing
│   ├── database/       # SQLC generated queries
│   └── metrics/        # Prometheus instrumentation
├── sql/
│   ├── schema/         # Goose migrations
│   └── queries/        # SQL queries for SQLC
├── docker-compose.yml  # Local development services
└── openapi.yaml        # API specification
```

### Testing

```bash
go test ./...
```

Tests use **testcontainers** to spin up real PostgreSQL instances for integration testing.

### Monitoring

Access monitoring dashboards:

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 *(default: admin/admin)*
