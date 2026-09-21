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

### Step 1.2: Configure Secrets and API URL

#### A. ZITADEL Signing Key (Required for HMAC Verification)
When creating a ZITADEL Target, ZITADEL generates a `signingKey`. This key is used to compute and verify the `Zitadel-Signature` HMAC header on every webhook request.

#### B. Service User Personal Access Token (Recommended for `preuserinfo`)
In ZITADEL Actions V2, `user_grants` are included in `preaccesstoken` payloads, but **omitted** by ZITADEL in `preuserinfo` payloads. If your application or proxy (like `oauth2-proxy`) queries `/userinfo`, `zitadel-actions-api` needs to query ZITADEL's Management API to fetch the user's project roles dynamically.

To configure this:
1. In ZITADEL Console, create a Service User under **Users -> Service Users -> New** (e.g. `actions-api-service-user`).
2. Assign the Service User the **Org Viewer** role (or Org Project Permission Editor) so it can view user grants.
3. Under the Service User's **Personal Access Tokens**, create a new token (PAT) and copy it.
4. Set `ZITADEL_API_URL` in `deploy/kubernetes/configmap.yaml` (e.g., `https://auth.example.com` or internal URL `http://zitadel:8080`).
5. Copy `deploy/kubernetes/secret.example.yaml` to `secret.yaml` and provide both keys:
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: zitadel-actions-api-secret
     namespace: zitadel
   type: Opaque
   stringData:
     ZITADEL_SIGNING_KEY: "<your-target-signing-key>"
     ZITADEL_API_TOKEN: "<your-service-user-pat>"
   ```
6. Apply the secret:
   ```bash
   kubectl apply -f deploy/kubernetes/secret.yaml
   ```

### Step 1.3: Verify Deployment


```bash
kubectl get pods -l app.kubernetes.io/name=zitadel-actions-api
kubectl get svc zitadel-actions-api
```

The service is now reachable within your cluster at:
`http://zitadel-actions-api.zitadel.svc.cluster.local/actions`

---

## 2. Configuring ZITADEL Actions V2

With `zitadel-actions-api`, you only need to configure **one single Target** in ZITADEL. All executions (`preaccesstoken`, `preuserinfo`, user creation hooks, etc.) point to this same target. The internal Action Dispatcher automatically coordinates all registered processors.

### Step 2.1: Create the Single Target

Make a `POST` request to ZITADEL API using a service user token:

```bash
curl -X POST "https://<your-zitadel-domain>/v2/actions/targets" \
  -H "Authorization: Bearer <SERVICE_USER_TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "zitadel-actions-target",
    "endpoint": "http://zitadel-actions-api.zitadel.svc.cluster.local/actions",
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

---

## 5. Troubleshooting

### `Errors.Target.DeniedURL (COMMAND-NcJUKo)`
**Cause**: ZITADEL has built-in SSRF protection. By default, ZITADEL's `HTTPClient.DenyList` blocks RFC1918 private IP ranges (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `127.0.0.0/8`, `localhost`). When you give ZITADEL an internal Kubernetes service DNS (e.g. `*.cluster.local`), it resolves to an internal ClusterIP and rejects the target.

**Solution**:
Adjust ZITADEL's `HTTPClient.DenyList` to permit in-cluster requests:

1. **Via Environment Variable** (in your ZITADEL Deployment/StatefulSet):
   ```yaml
   env:
     - name: ZITADEL_HTTPCLIENT_DENYLIST
       value: "169.254.169.254/32,127.0.0.1/32,localhost"
   ```
   *(Or set `value: ""` to disable the denylist entirely in a trusted homelab).*

2. **Via Helm `values.yaml`**:
   ```yaml
   config:
     HTTPClient:
       DenyList:
         - "169.254.169.254/32"
         - "127.0.0.1/32"
         - "localhost"
   ```
Restart ZITADEL after applying this change, and target creation will succeed.
