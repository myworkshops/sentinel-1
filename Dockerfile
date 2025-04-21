# Builder stage
FROM golang:1.23.4-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o sentinel-1 ./cmd/webapp

# Runtime stage
FROM alpine:3.18
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy application binary
COPY --from=builder /app/sentinel-1 .

# Copy HTML templates
COPY --from=builder /app/internal/web/templates ./internal/web/templates

# Copy static assets (logo.svg, CSS, etc.)
COPY --from=builder /app/static ./static

# Setup directory for mounted secrets
RUN mkdir -p /etc/secrets/keycloak

EXPOSE 8080

ENV SECRET_WATCH_PATH=/etc/secrets/keycloak

ENTRYPOINT ["/app/sentinel-1"]