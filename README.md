![Banner](img/sentinel-banner.png)

## Sentinel‑1 API – Quick Start (Local Docker Run)

A minimal Go web application that demonstrates **Kubernetes‑style secret**

**rotation** and **OIDC login with Keycloak**.

It provisions:

- /         → login page (button redirects to Keycloak)
    
- /callback → exchanges code, verifies ID Token, stores token JSON
    
- /token    → pretty‑printed token + decoded access‑token claims
    
The code is container‑ready and requires **only a Keycloak realm** running locally (or port‑forwarded) plus a client with client_id / client_secret.

### 1 · Prerequisites

| **Requirement** | **Notes** |
| --- | --- |
| Docker 20+ | Build & run image |
| Keycloak | Up and listening on localhost:8080 (native, container or kubectl port‑forward) |
| /etc/hosts | Map your realm host if you use a vanity name (keycloak.test → 127.0.0.1) |

### 2 · Build (or pull) the image

```
# build locally
docker build -t sentinel-1-api:latest .
```
— or —
```
# pull if you pushed it to a registry
docker pull ghcr.io/your‑org/sentinel-1-api:1.0.0
```

### 3 · Required environment variables

| **Variable** | **Purpose** | **Example** |
| --- | --- | --- |
| KEYCLOAK_ISSUER_URL | Public URL shown to the browser – must match the issuer claim Keycloak issues | http://keycloak.test/realms/iam |
| KEYCLOAK_INTERNAL_URL | URL reachable by the container for JWKS / token calls | http://127.0.0.1:8080/realms/iam |
| REDIRECT_URL | Where Keycloak sends the user back | http://localhost:8081/callback |
| **either** |     |     |
| KEYCLOAK_CLIENT_ID | Client ID (provided by Keycloak) | sentinel-api-client |
| KEYCLOAK_CLIENT_SECRET | Client secret | top‑secret |
| **or** |     |     |
| SECRET_WATCH_PATH | Path where two files client_id / client_secret are mounted (K8s style) | /etc/secrets/keycloak |

> If SECRET_WATCH_PATH is provided, the application reads the initial credentials from the files and watches the directory for updates.


### 4 · Run the container locally

```
docker run --rm -p 8081:8080 \
  -e KEYCLOAK_ISSUER_URL="http://keycloak.test/realms/iam" \
  -e KEYCLOAK_INTERNAL_URL="http://127.0.0.1:8080/realms/iam" \
  -e REDIRECT_URL="http://localhost:8081/callback" \
  -e KEYCLOAK_CLIENT_ID="sentinel-api-client" \
  -e KEYCLOAK_CLIENT_SECRET="super-secret" \
  sentinel-1-api:latest
```

- Visit [**http://localhost:8081/**](http://localhost:8081/) → press **Login**
    
- Authenticate in Keycloak → you’ll be redirected to **/token**
    
- The page shows the full token JSON and the decoded access‑token claims
    
#### Using file‑based secrets (optional)

```
docker run --rm -p 8081:8080 \
  -e KEYCLOAK_ISSUER_URL="http://keycloak.test/realms/iam" \
  -e KEYCLOAK_INTERNAL_URL="http://127.0.0.1:8080/realms/iam" \
  -e REDIRECT_URL="http://localhost:8081/callback" \
  -e SECRET_WATCH_PATH="/etc/secrets/keycloak" \
  -v "$PWD/test-secrets":/etc/secrets/keycloak:ro \
  sentinel-1-api:latest
```

Any change to client_id or client_secret inside the mounted

directory is detected on‑the‑fly – no need to restart the container.

### 5 · Troubleshooting

| **Symptom** | **Likely cause / fix** |
| --- | --- |
| invalid_grant “Code not valid” | Page */callback* refreshed – use the provided flow with redirect to */token* |
| context deadline exceeded when fetching /certs | Keycloak URL unreachable from container – check host mapping or KEYCLOAK_INTERNAL_URL |
| Browser cannot reach Keycloak | Add host entry 127.0.0.1 keycloak.test or use localhost URLs |


### 6 · Project layout (recap)

```
cmd/webapp           – main entrypoint
internal/config      – env / secret loader
internal/keycloak    – OIDC client
internal/server      – Gin handlers + middleware
internal/web         – HTML templates
pkg/watcher          – filesystem secret hot‑reload
Dockerfile
```

### 7 · License

This project is licensed under the **MIT License** – see LICENSE for details.