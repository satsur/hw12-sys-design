# Application Entry Point

## Overview

The project has one public entry point: the Nginx Kubernetes Service. Docker
Desktop Kubernetes exposes that Service as a local `LoadBalancer`, so students
can use the application without running `kubectl port-forward`.

```text
Browser or shell CLI
        |
        | http://localhost:8080
        v
Docker Desktop Kubernetes LoadBalancer
        |
        v
nginx-service:8080
        |
        +--> /              -> web-service:8080
        +--> /metrics       -> web-service:8080/metrics.html
        +--> /wordle        -> wordle-api-service:8080
        +--> /api           -> control-plane-service:5000
```

The `nginx-service` Service is the only service intended to be reached from the
host machine. The other Services remain internal Kubernetes `ClusterIP`
Services.

## What a Kubernetes Service Does

A Kubernetes Service is a stable network name and virtual address in front of
one or more pods. Pods are replaceable, so clients should not connect directly
to pod IP addresses.

For example:

```text
wordle-api-service:8080
```

always means "the healthy Wordle API pods selected by this Service," even when
those pods are replaced during scaling or self-healing.

The Nginx Service is different because its type is `LoadBalancer`:

```yaml
spec:
  type: LoadBalancer
  ports:
    - port: 8080
      targetPort: 8080
```

- `targetPort: 8080` is the port Nginx listens on inside its pod.
- `port: 8080` is the port exposed by the Kubernetes Service.
- `LoadBalancer` asks the local Kubernetes provider to make that Service
  reachable from the host.

When Docker Desktop's local load-balancer integration exposes the proxy to the
host, the resulting endpoint is `http://localhost:8080`. The exact Kubernetes
address is visible with:

```bash
kubectl get service nginx-service
```

Wait until the Service has an external address or until Docker Desktop reports
the load balancer as ready. Some Docker Desktop Kubernetes versions report an
internal Docker-network address instead of a host-reachable localhost address.
If that address cannot be reached from the host, use the port-forward fallback
described below; the Kubernetes routing itself is still correct.

## Why the Control Plane Still Uses Port 5000

The control plane process listens on port `5000` inside its pod and its
ClusterIP Service:

```text
control-plane-service:5000
```

That port is not a host port. Nginx connects to it through Kubernetes DNS:

```nginx
upstream control-plane {
    server control-plane-service:5000;
}
```

A browser or CLI request to:

```text
http://localhost:8080/api/v1/status
```

therefore follows this path:

```text
host:8080
  -> nginx-service:8080
  -> control-plane-service:5000
  -> FastAPI route /api/v1/status
```

The CLI defaults to `http://localhost:8080`, so it uses the same public path as
the browser. It does not need a separate control-plane port-forward.

## Avoiding CORS

Browsers apply the same-origin policy. A page at
`http://localhost:8080/metrics` making a request to
`http://localhost:5000/api/v1/metrics` would be a cross-origin request and
would require CORS headers.

The dashboard instead requests:

```text
http://localhost:8080/api/v1/metrics
```

Both the page and API use the same protocol, host, and port. Nginx performs the
internal routing, so the browser does not need CORS configuration.

## Relationship to the Load Generator

The load generator is not exposed through the public LoadBalancer. It remains
an internal Service:

```text
loadgen-service:8080
```

The control plane starts tests by calling that Service:

```text
POST control-plane /api/v1/load-tests
    -> POST loadgen-service:8080/start
        -> requests nginx-service:8080/wordle
            -> wordle-api-service:8080
```

This ensures generated traffic uses the same Nginx path as browser traffic. The
load generator does not call individual pods directly.

## Troubleshooting

### LoadBalancer address is not reachable from the host

Docker Desktop Kubernetes runs its node inside Docker. A LoadBalancer address
such as `172.19.0.5` may be reachable inside Docker's network but not from a
Windows or WSL browser. This is a limitation of the local provider, not an
application routing failure.

Use the stable fallback while the cluster is running:

```powershell
kubectl port-forward service/nginx-service 8080:8080
```

### External address is pending

Check Docker Desktop Kubernetes status and inspect the Service:

```bash
kubectl describe service nginx-service
kubectl get events --sort-by=.lastTimestamp
```

The pods and Service endpoints should be ready before the load balancer can
forward traffic.

### Port 8080 is already in use

Docker Desktop cannot bind the LoadBalancer to a host port already owned by
another application. Stop the conflicting process or choose a different public
Service port and update the documented URL consistently.

### The browser loads but `/api` fails

Check the Nginx and control-plane pods:

```bash
kubectl get pods -l app=nginx-proxy
kubectl get pods -l app=control-plane
kubectl logs deployment/nginx-proxy
kubectl logs deployment/control-plane
```

The control plane must be listening on `0.0.0.0:5000`, while the Nginx upstream
must point to `control-plane-service:5000`.

### Load tests return connection errors

Check the internal loadgen and Nginx Services:

```bash
kubectl get service loadgen-service nginx-service
kubectl logs deployment/loadgen
kubectl logs deployment/nginx-proxy
```

The load generator should use `http://nginx-service:8080/wordle`, not
`localhost`, because `localhost` inside a pod refers to that same pod.

## Why Not Gateway API?

Gateway API is a useful Kubernetes standard, but it requires installing and
maintaining a Gateway controller. Docker Desktop's LoadBalancer integration provides the needed host exposure
with fewer moving parts. Nginx already handles the path routing, so a Gateway
would duplicate that responsibility.
