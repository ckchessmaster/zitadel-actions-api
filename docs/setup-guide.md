# Setup Guide: ZITADEL, Kubernetes & oauth2-proxy for Frigate

This guide walks through configuring **`zitadel-actions-api`** in your on-premises Kubernetes cluster and linking it to ZITADEL and `oauth2-proxy`.

---

## 1. Deploying to Kubernetes

### Step 1.1: Deploy using Kustomize

From the repository root:

```bash
# Review or modify configuration in deploy/kubernetes/configmap.yaml
kubectl apply -k deploy/kubernetes/
```

### Step 1.2: (Optional) Configure ZITADEL Signing Key

If you want ZITADEL to verify request integrity using an HMAC signature:

1. Copy `deploy/kubernetes/secret.example.yaml` to `secret.yaml`.
2. Add the `signingKey` provided when creating the ZITADEL Target.
3. Apply the secret:
   ```bash
   kubectl apply -f deploy/kubernetes/secret.yaml
   ```

### Step 1.3: Verify Deployment

```bash
kubectl get pods -l app.kubernetes.io/name=zitadel-actions-api
kubectl get svc zitadel-actions-api
```

The service is now reachable within your cluster at:
`http://zitadel-actions-api.default.svc.cluster.local/v1/actions/flatten-roles`

---

## 2. Configuring ZITADEL Actions V2

ZITADEL Actions V2 uses **Targets** and **Executions** to call external webhooks.

### Step 2.1: Create Target

Make a `POST` request to ZITADEL API using a service user token:

```bash
curl -X POST "https://<your-zitadel-domain>/v2/actions/targets" \
  -H "Authorization: Bearer <SERVICE_USER_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "flatten-roles-target",
    "endpoint": "http://zitadel-actions-api.default.svc.cluster.local/v1/actions/flatten-roles",
    "timeout": "5s",
    "payloadType": "PAYLOAD_TYPE_JSON",
    "restCall": {
      "interruptOnError": false
    }
  }'
```

> **Note:** The response contains an `id` and a `signingKey`. Save the `id` for Step 2.2.

### Step 2.2: Create Executions

Bind the Target to run before token creation:

#### A. Pre-Access Token Creation
```bash
curl -X PUT "https://<your-zitadel-domain>/v2/actions/executions" \
  -H "Authorization: Bearer <SERVICE_USER_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "condition": {
      "function": {
        "name": "preaccesstoken"
      }
    },
    "targets": ["<TARGET_ID>"]
  }'
```

#### B. Pre-Userinfo Creation (for ID tokens and userinfo endpoints)
```bash
curl -X PUT "https://<your-zitadel-domain>/v2/actions/executions" \
  -H "Authorization: Bearer <SERVICE_USER_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "condition": {
      "function": {
        "name": "preuserinfo"
      }
    },
    "targets": ["<TARGET_ID>"]
  }'
```

---

## 3. Configuring oauth2-proxy for Frigate

In your `oauth2-proxy` deployment or Helm values:

1. **OIDC Provider**:
   ```yaml
   args:
     - --provider=oidc
     - --oidc-issuer-url=https://<your-zitadel-domain>
     - --client-id=<client-id>
     - --client-secret=<client-secret>
     - --cookie-secret=<cookie-secret>
   ```

2. **Groups Claim Configuration**:
   ```yaml
   args:
     # Tell oauth2-proxy to read the flattened 'groups' claim
     - --oidc-groups-claim=groups
     - --scope=openid profile email

     # Restrict access to users having the role 'frigate-admin'
     - --allowed-group=frigate-admin
   ```

3. **Pass Headers to Frigate**:
   ```yaml
   args:
     - --upstream=http://frigate:5000
     - --set-xauthrequest=true
     - --pass-user-headers=true
   ```

Frigate will now receive the authenticated headers:
- `X-Forwarded-User: <username>`
- `X-Forwarded-Groups: frigate-admin`

---

## 4. Query Parameter Options

You can override defaults per request URL:
- `?claim_name=roles`: Injects into `roles` instead of `groups`.
- `?format=prefixed`: Generates `projectId:role` format (e.g. `frigate:admin`).
- `?project_id=<id>`: Only includes roles belonging to the specified project.
- `?lowercase=false`: Preserves uppercase casing of role names.
