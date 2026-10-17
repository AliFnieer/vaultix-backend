# =============================================================================
# Vaultix Backend - Multi-stage Docker build
# =============================================================================

# Build stage
FROM golang:1.27.2-alpine AS builder
WORKDIR /app

# Install build dependencies (ca-certificates for HTTPS)
RUN apk add --no-cache ca-certificates git

# Cache dependencies first
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /app/server ./cmd/api

# Run stage
FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/server /app/server
COPY .env.example .env.example

EXPOSE 8080

ENV APP_ENV=production
ENV APP_PORT=8080

ENTRYPOINT ["/app/server"]
