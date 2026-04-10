# SSL Certificate Setup with Let's Encrypt

**Note:** `make deploy` is **HTTP-only** and does **not** install cert-manager or apply `cert-manager.yaml`. Use this document when you want to add TLS manually.

This optional flow provisions SSL certificates using Let's Encrypt and cert-manager.

## Prerequisites

### 1. Nginx Ingress Controller

Still required before TLS routing works (install once per cluster):

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s
```

### 2. cert-manager

Install cert-manager on the cluster (not done by `make deploy`):

Manual install:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.5/cert-manager.yaml
```

## Deployment

Deploy the application with your domain (install ingress first; cert-manager is handled by deploy when needed):

```bash
# If you SSH into the same hostname that users use in the browser:
make deploy HOST=iammanager.it.eu-cloud.mirantis.net USER=your-linux-user

# If you SSH to a different machine than the public app URL (recommended when sharing an ingress):
make deploy HOST=k8s-bastion.internal DOMAIN=iammanager.it.eu-cloud.mirantis.net USER=ubuntu
```

**Important**:
- **HOST** is the `ssh`/`scp` target (bastion or control node with `kubectl`).
- **DOMAIN** (optional) is the DNS name on the certificate and Ingress. It defaults to **HOST** when omitted.
- **Shared port 80**: The nginx ingress controller listens on port 80 for all `Ingress` resources. Other apps use different `host:` values on the same controller. This project’s Ingress only matches your **DOMAIN**, so it does not take over traffic for unrelated hostnames.

**DNS**: Create an `A` (or `AAAA`) record for **DOMAIN** pointing at the ingress controller’s external load balancer IP (same as your other app if it already uses that ingress).

### “Connection refused” on https://your-domain/

Usually **nothing is listening on :443** at the IP your DNS resolves to.

1. **Confirm DNS** (must be the IP where HTTP(S) terminates, often the node running ingress):
   - `dig +short your.domain.example`
2. **Ingress controller** must exist and expose 80/443. On **bare metal / k0s** the default cloud `LoadBalancer` never gets an `EXTERNAL-IP`; the stock **bare-metal** ingress manifest uses **NodePort** instead, so **:443 on the node is closed** unless you use **hostNetwork** or an external LB.
3. Install nginx ingress and bind host ports (single-node k0s + DNS → that node IP):

   ```bash
   make install-ingress-nginx HOST=YOUR_NODE_IP USER=ubuntu INGRESS_HOSTNETWORK=1
   ```

4. **Firewall**: allow **80** and **443** on that host (Let’s Encrypt HTTP-01 needs **80**).

5. From your laptop: `curl -vkI --resolve your.domain.example:443:THAT_IP https://your.domain.example/` to test before relying on public DNS.

## SSL Certificate Process

`make deploy` applies an **Ingress** with `cert-manager.io/cluster-issuer`; cert-manager’s **ingress-shim** creates the **Certificate** and writes **`cloud-manager-tls`**. (The file `k8s/certificate.yaml` is optional/manual only—not applied by deploy—to avoid duplicate Certificates fighting the same secret.)

1. **Certificate**: Created by cert-manager from the Ingress `tls` section
2. **HTTP-01 Challenge**: cert-manager serves the challenge (needs **port 80** to the ingress)
3. **Let’s Encrypt** issues the cert → **Secret** `cloud-manager-tls`
4. **Nginx** terminates TLS using that secret

## Verification

Check certificate status:

```bash
# Check certificate status
kubectl get certificates -n cloud-manager

# Check certificate details (name may be cloud-manager-tls-… from ingress-shim)
kubectl describe certificate -n cloud-manager -l app.kubernetes.io/name=cloud-manager 2>/dev/null || kubectl describe certificate -n cloud-manager

# Check certificate secret
kubectl get secret cloud-manager-tls -n cloud-manager
```

## Troubleshooting

### HTTP 503 from the ingress URL

A **503** from nginx usually means **no ready backends** for that `Ingress` host (not a TLS failure).

1. **Pods and endpoints**

   ```bash
   kubectl get pods,svc,endpoints -n cloud-manager
   kubectl describe pod -n cloud-manager -l app.kubernetes.io/component=app
   kubectl logs -n cloud-manager -l app.kubernetes.io/component=app --tail=80
   ```

2. **SQLite on PVC**: The app opens `DB_PATH` (default `/data/iam-manager.db`). If the pod runs as non-root and the volume is root-owned, startup can fail and the pod never becomes Ready → **503**. The Deployment sets `runAsUser` / `fsGroup` **1000** to match the image user; rebuild/push the image after the Dockerfile uid/gid change, then redeploy.

3. **Image pull**: `ImagePullBackOff` also yields no endpoints → 503. Confirm `ghcr.io/mirantis/cloud-manager:latest` exists and is pullable from the cluster (image pull secrets if the registry is private).

### Browser shows a “wrong” or default certificate

Until **`cloud-manager-tls`** exists and is bound on the Ingress, nginx often serves the **controller default certificate** (looks self-signed or “fake”). That is expected until cert-manager marks the Certificate **Ready**.

```bash
kubectl get certificate,secret -n cloud-manager
kubectl describe certificate cloud-manager-cert -n cloud-manager
kubectl get challenges,orders -n cloud-manager
```

Fix ACME (DNS to this ingress, **port 80** open for HTTP-01, ClusterIssuer email) first. If you previously had **two** `Certificate` objects or a stuck **invalid** order, delete them and let cert-manager recreate from the Ingress:

```bash
kubectl delete certificate --all -n cloud-manager
kubectl delete secret cloud-manager-tls -n cloud-manager --ignore-not-found
kubectl delete challenge,order,certificaterequest -n cloud-manager --all --ignore-not-found
```

Then `make deploy ...` again (or only re-apply the Ingress). If Let’s Encrypt **rate-limited** you, deploy with **`CERT_ISSUER=letsencrypt-staging`** once, confirm staging cert works, then switch back to **`letsencrypt-prod`** after the limit window.

### Certificate Pending

If certificate status shows "Pending":

```bash
# Check cert-manager logs
kubectl logs -n cert-manager -l app=cert-manager

# Check certificate events
kubectl describe certificate cloud-manager-cert -n cloud-manager

# Check challenge status
kubectl get challenges -n cloud-manager
```

### Common Issues

1. **DNS not pointing to ingress**: Ensure your domain points to the ingress controller's external IP
2. **Firewall blocking port 80**: HTTP-01 challenge requires port 80 to be accessible
3. **Rate limiting**: Let's Encrypt has rate limits; use staging issuer for testing
4. **Invalid email**: Ensure the email in cert-manager.yaml is valid

### Using Staging Environment

For testing, use the staging issuer to avoid rate limits:

```bash
# Edit certificate.yaml to use staging issuer
# issuerRef:
#   name: letsencrypt-staging
#   kind: ClusterIssuer
```

## Security Notes

- Certificates auto-renew 15 days before expiry
- HTTP traffic is automatically redirected to HTTPS
- HTTPS is enforced for all routes
- Application authentication (admin username/password) applies over SSL