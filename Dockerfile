# ---- Builder Stage ----
FROM golang:1.23.4-alpine AS builder
# (…igual que antes…)

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o sentinel-1 ./cmd/webapp

# ---- Runtime Stage ----
FROM alpine:3.18

RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copiamos sólo el binario
COPY --from=builder /app/sentinel-1 .

# **Y ahora copiamos también los templates**  
COPY --from=builder /app/internal/web/templates ./internal/web/templates

# Directorio donde montaremos el Secret
RUN mkdir -p /etc/secrets/keycloak

EXPOSE 8080

ENV SECRET_WATCH_PATH=/etc/secrets/keycloak

ENTRYPOINT ["/app/sentinel-1"]