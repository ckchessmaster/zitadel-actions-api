# AGENTS.md: Developer & Agent Guide for zitadel-actions-api

Welcome, Agent! This document provides architectural context, development workflows, and coding conventions for working within the `zitadel-actions-api` repository.

---

## 1. System Overview

`zitadel-actions-api` is an HTTP microservice designed to integrate with **ZITADEL Actions** (specifically **Actions V2 Targets & Executions**).

Its primary capability is to intercept token creation flows in ZITADEL, extract user project roles, and flatten them into a standard `groups` claim (e.g. `groups: ["frigate-admin", "viewer"]`) so that downstream proxies like `oauth2-proxy` protecting [Frigate NVR](https://frigate.video/) (or other services) can evaluate RBAC rules and forward `X-Forwarded-Groups` without custom claim transformations.

---

## 2. Directory Layout

```
zitadel-actions-api/
├── .github/
│   └── workflows/
│       ├── ci.yml                 # CI: formatting, vet, unit tests with -race
│       └── build-and-publish.yml  # CD: multi-arch build (amd64, arm64) & GHCR publish
├── cmd/
│   └── server/
│       └── main.go                # Server entrypoint, graceful shutdown, router wiring
├── internal/
│   ├── config/                    # Environment variable configuration loading
│   │   ├── config.go
│   │   └── config_test.go
│   ├── handler/                   # HTTP handlers
│   │   ├── health.go              # Liveness (/healthz) & readiness (/readyz)
│   │   ├── health_test.go
│   │   ├── flatten_roles.go       # Role-flattening action endpoint
│   │   └── flatten_roles_test.go
│   ├── middleware/                # HTTP middlewares
│   │   ├── logger.go              # Structured logging (log/slog)
│   │   ├── recover.go             # Panic recovery
│   │   ├── signature.go           # ZITADEL HMAC signature verification (zitadel-go)
│   │   ├── signature_test.go
│   │   └── recover_test.go
│   ├── model/                     # Request and response models
│   │   └── zitadel.go
│   └── service/                   # Pure business logic
│       ├── flattener.go           # Role extraction, filtering, deduplication
│       └── flattener_test.go
├── deploy/
│   └── kubernetes/                # Kubernetes deployment manifests & Kustomize
├── docs/                          # Architecture & setup documentation
├── examples/                      # Config examples (ZITADEL Targets, oauth2-proxy)
├── Dockerfile                     # Multi-stage distroless build (<20 MB)
├── Makefile                       # Common tasks
├── go.mod
└── README.md
```

---

## 3. Key Design Rules & Patterns

### 3.1 ZITADEL Actions V2 Compatibility
- ZITADEL expects webhook responses to return `append_claims: [{ key: "groups", value: ["..."] }]`.
- Always maintain dual-compatibility: return `append_claims` **and** flat convenience keys (`groups`, `claims`) so direct API callers, testing utilities, and legacy V1 JavaScript scripts work out of the box.
- Check both `Zitadel-Signature` and `X-Zitadel-Signature` headers for HMAC verification when `ZITADEL_SIGNING_KEY` is configured.

### 3.2 Adding New Action Handlers
To add a new ZITADEL action handler:
1. Define any new request/response models in `internal/model/`.
2. Implement core business logic in an isolated service inside `internal/service/` with 100% test coverage.
3. Add an HTTP handler in `internal/handler/`.
4. Wire the handler into `actionsMux` in `cmd/server/main.go` under `/v1/actions/<action-name>`.

### 3.3 Coding Standards
- **Language**: Go 1.22+ (using standard library HTTP routing and `log/slog`).
- **Dependencies**: Keep external dependencies minimal. Currently only depends on `github.com/zitadel/zitadel-go/v3` for signature validation.
- **Concurrency**: Handlers must be stateless and safe for concurrent execution.
- **Determinism**: Slices returned in claims must be sorted for reproducible, deterministic output.

---

## 4. Development Workflows

### Run Tests:
```bash
make test-race
```

### Run Linter & Formatting:
```bash
make lint
```

### Run Locally:
```bash
make run
```

### Build Container:
```bash
make docker-build
```
