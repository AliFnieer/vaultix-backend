# vaultix-backend

Vaultix is a production-grade backend for a secrets / password vault application,
written in Go.

## Tech stack
- **Language:** Go 1.27.2
- **Web framework:** Gin
- **Database:** MongoDB (mongo-driver v2)
- **Validation:** go-playground/validator v10
- **JSON:** gopkg/sonic
- **JWT:** golang-jwt/jwt v5
- **Password hashing:** bcrypt (golang.org/x/crypto)
- **Field encryption:** AES-256-GCM (stdlib)

## Architecture
Clean, layered architecture:

```
cmd/server/              - application entry point
internal/app/            - dependency wiring + HTTP server
internal/config/         - environment configuration
internal/domain/         - entities, repository interfaces, sentinel errors
internal/infrastructure/ - MongoDB, JWT, bcrypt, AES-256-GCM field encryption
internal/usecase/        - business logic (auth, user, secret, vault, audit)
internal/dto/            - request/response DTOs
internal/handler/        - Gin HTTP handlers
internal/middleware/     - error handling, auth (JWT), CORS, structured logger
pkg/response/            - standardized JSON response envelopes
pkg/errors/              - domain error type + HTTP status mapping
```

## Security features
- JWT access tokens (short-lived) + refresh tokens with rotation
- bcrypt password hashing (configurable cost)
- AES-256-GCM field-level encryption for secret passwords/notes at rest
- SHA-256 hashed refresh tokens stored in DB (raw tokens never persisted)
- RBAC (user/admin) with route-level role enforcement
- Input validation + standardized error responses
- Audit logging for security-relevant events

## Getting started

### Prerequisites
- Go 1.27.2+
- MongoDB 7+

### 1. Clone & configure
```bash
cp .env.example .env
# Edit .env with your secrets (FIELD_ENCRYPTION_KEY, JWT secrets, etc.)
```

Generate a field-encryption key:
```bash
head -c 32 /dev/urandom | base64
```

### 2. Run locally
```bash
go mod tidy
go run ./cmd/server
```

Server listens on `:8080` (configurable via `APP_PORT`).

### 3. Run with Docker Compose
```bash
cp .env.example .env
docker compose up -d --build
```

## API overview

### Auth (`/api/v1/auth`)
- `POST /register` — create account
- `POST /login` — credentials → access + refresh token
- `POST /refresh` — rotate refresh token → new access token
- `POST /logout` — revoke all refresh tokens (Bearer)
- `GET /me` — current user profile (Bearer)

### Users (`/api/v1/users`)
- `GET /me`, `PUT /me`, `PUT /me/password`
- `PUT /:id/role` (admin), `PUT /:id/active` (admin)

### Secrets (`/api/v1/secrets`)
- `POST`, `GET` (list, paginated), `GET /:id`, `PUT /:id`, `DELETE /:id`

### Vaults (`/api/v1/vaults`)
- `POST`, `GET`, `PUT /:id`, `DELETE /:id`

### Health
- `GET /health`

## License
MIT