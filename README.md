# zitadel-actions-api

A lightweight, high-performance Go microservice designed for **ZITADEL Actions** (with first-class support for **Actions V2 Targets & Executions** and fallback support for Actions V1 JavaScript scripts).

The initial goal of this service is to **flatten ZITADEL project roles into a clean `groups` claim** so that downstream proxies such as [oauth2-proxy](https://oauth2-proxy.github.io/oauth2-proxy/) protecting services like [Frigate NVR](https://frigate.video/) can perform standard RBAC and forward headers like `X-Forwarded-Groups`.

---

## Features

- **Actions V2 Ready**: Implements ZITADEL Actions V2 `restCall` webhook target contract (`append_claims`).
- **Backward & Forward Compatible**: Parses roles from Actions V2 `user_grants`, Actions V1 `grants`, and nested `urn:zitadel:iam:org:project:roles` claims.
- **HMAC Signature Security**: Validates incoming `Zitadel-Signature` headers using official `github.com/zitadel/zitadel-go/v3/pkg/actions` when `ZITADEL_SIGNING_KEY` is set.
- **Configurable**: Customize claim name (default: `groups`), role formatting (`bare` or `prefixed`), lowercase normalization, and project-specific filtering.
- **Ultra-Lightweight & Secure**: Multi-stage distroless scratch container (<20 MB), non-root execution (`UID 65532`), minimal memory footprint (<10 MB).
- **Kubernetes-Native**: Includes liveness/readiness probes (`/healthz`, `/readyz`) and production K8s manifests.
- **Multi-Architecture**: Built for `linux/amd64` and `linux/arm64` via GitHub Actions and published to GitHub Container Registry (`ghcr.io`).

---

## API Endpoints

### 1. `POST /actions` *(Primary Unified Target Endpoint)*

Unified webhook endpoint for ZITADEL Actions V2 Targets.
You only need to configure **one Target** in ZITADEL pointing to `/actions`. The internal dispatcher automatically coordinates all applicable action processors (e.g. role flattening, claims enrichment, user validation) based on the execution trigger and merges the results.

*(Aliases: `POST /v1/actions`, `POST /v1/actions/flatten-roles`, `POST /flatten-roles`)*

#### Query Parameters (Optional Overrides):
| Parameter | Default | Description | Example |
| :--- | :--- | :--- | :--- |
| `claim_name` | `groups` | Target claim key to populate in token | `?claim_name=roles` |
| `format` | `bare` | Formatting style: `bare` or `prefixed` | `?format=prefixed` |
| `project_id` | *(all)* | Limit roles to a specific project ID | `?project_id=2638491` |
| `lowercase` | `true` | Convert role strings to lowercase | `?lowercase=false` |

#### Sample Request (ZITADEL Actions V2):
```json
{
  "user_grants": [
    {
      "projectId": "frigate-proj",
      "roles": ["frigate-admin", "viewer"]
    },
    {
      "projectId": "monitoring-proj",
      "roles": ["editor"]
    }
  ]
}
```

#### Sample Response:
```json
{
  "append_claims": [
    {
      "key": "groups",
      "value": ["editor", "frigate-admin", "viewer"]
    }
  ],
  "groups": ["editor", "frigate-admin", "viewer"],
  "claims": {
    "groups": ["editor", "frigate-admin", "viewer"]
  }
}
```

### 2. Probes
- `GET /healthz` - Kubernetes liveness probe (`{"status":"ok"}`)
- `GET /readyz` - Kubernetes readiness probe (`{"status":"ready"}`)

---

## Configuration

All configuration can be provided via environment variables:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | HTTP port the server listens on |
| `LOG_LEVEL` | `info` | Logging verbosity (`debug`, `info`, `warn`, `error`) |
| `CLAIM_NAME` | `groups` | Default claim name to append |
| `ROLE_FORMAT` | `bare` | Default role format (`bare` or `prefixed`) |
| `LOWERCASE_ROLES` | `true` | Convert extracted roles to lowercase |
| `FILTER_PROJECT_ID`| `""` | Restrict extracted roles to this project ID |
| `ZITADEL_SIGNING_KEY`| `""` | Signing key for validating `Zitadel-Signature` (disabled if empty) |

---

## Quickstart

### Run locally with Go:
```bash
make run
```

### Test with cURL:
```bash
curl -X POST http://localhost:8080/v1/actions/flatten-roles \
  -H "Content-Type: application/json" \
  -d '{
    "user_grants": [
      {"projectId": "frigate", "roles": ["ADMIN", "viewer"]}
    ]
  }'
```

### Run tests:
```bash
make test-race
```

### Build container image:
```bash
make docker-build
```

---

## Kubernetes Deployment

Deploy with Kustomize:
```bash
kubectl apply -k deploy/kubernetes/
```

See [docs/setup-guide.md](docs/setup-guide.md) for full instructions on setting up ZITADEL Targets, Executions, and integrating `oauth2-proxy` with Frigate.

---

## Architecture & Integration

See [docs/architecture.md](docs/architecture.md) for sequence diagrams and payload lifecycle details.
