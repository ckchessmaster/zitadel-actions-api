# Architecture: zitadel-actions-api

## Overview

`zitadel-actions-api` is a lightweight, stateless microservice written in Go designed to act as a backend endpoint for **ZITADEL Actions**.

Its primary initial purpose is solving the role representation mismatch between ZITADEL's native nested role maps and OIDC consumers like `oauth2-proxy`, which expect a simple string array claim (`groups: ["admin", "viewer"]`).

## Data Flow

```mermaid
sequenceDiagram
    autonumber
    actor User as User Browser
    participant Proxy as oauth2-proxy
    participant Zitadel as ZITADEL IdP
    participant ActionsAPI as zitadel-actions-api
    participant Frigate as Frigate NVR

    User->>Proxy: Access https://frigate.example.com
    Proxy->>Zitadel: Redirect to OIDC Login
    User->>Zitadel: Enter Credentials & MFA
    Note over Zitadel: Authentication successful.<br/>Trigger: preaccesstoken / preuserinfo
    Zitadel->>ActionsAPI: POST /v1/actions/flatten-roles<br/>Header: Zitadel-Signature<br/>Body: { user_grants: [...] }
    ActionsAPI->>ActionsAPI: 1. Validate signature (if key set)<br/>2. Extract roles from grants/claims<br/>3. Deduplicate & lowercase<br/>4. Build append_claims
    ActionsAPI-->>Zitadel: 200 OK<br/>{ append_claims: [{ key: "groups", value: ["frigate-admin"] }] }
    Note over Zitadel: Injects "groups" claim into token
    Zitadel-->>Proxy: Redirect with Auth Code -> Exchange for Tokens
    Note over Proxy: Extracts "groups" claim.<br/>Matches --allowed-group=frigate-admin
    Proxy->>Frigate: Reverse proxy request<br/>Headers: X-Forwarded-User, X-Forwarded-Groups
    Frigate-->>User: 200 OK (Frigate Web Interface)
```

## Input Normalization

ZITADEL passes user roles in different formats depending on the API version and configuration:

1. **Actions V2 Target Payload (`user_grants`)**:
   ```json
   {
     "user_grants": [
       {
         "projectId": "2638491028301",
         "roles": ["frigate-admin", "viewer"]
       }
     ]
   }
   ```
2. **Actions V1 / Custom JS Payload (`grants`)**:
   ```json
   {
     "grants": [
       {
         "projectId": "2638491028301",
         "roles": ["frigate-admin"]
       }
     ]
   }
   ```
3. **Legacy Claims Map (`urn:zitadel:iam:org:project:roles`)**:
   ```json
   {
     "claims": {
       "urn:zitadel:iam:org:project:roles": {
         "frigate-admin": { "2638491028301": "My Organization" }
       }
     }
   }
   ```

`zitadel-actions-api` parses all of the above structures simultaneously, deduplicates the role names, and returns a sorted array.

## Output Specification

The service returns a payload designed to satisfy both ZITADEL Actions V2 and direct HTTP consumers:

```json
{
  "append_claims": [
    {
      "key": "groups",
      "value": ["frigate-admin", "viewer"]
    }
  ],
  "groups": ["frigate-admin", "viewer"],
  "claims": {
    "groups": ["frigate-admin", "viewer"]
  }
}
```

- **`append_claims`**: Read natively by ZITADEL Actions V2 when the target is configured with `restCall`.
- **`groups` & `claims`**: Can be used by Actions V1 JavaScript scripts or tested via `curl`.

## Extensibility

The codebase is structured to facilitate adding more action handlers:
- `internal/handler/`: Add new handlers (e.g. `validate_user.go`, `sync_ldap.go`) without touching core routing.
- `internal/service/`: House independent business logic components.
- `internal/middleware/`: Reusable middleware for signature validation, structured logging, and panic recovery.
