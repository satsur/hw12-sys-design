# Architecture

## Runtime Topology

```text
Host: localhost:8080
          |
          v
   nginx-service (LoadBalancer)
      |       |        |
      v       v        v
    web   wordle-api  control-plane
                         |
                         v
                       loadgen
```

All components run in Docker Desktop Kubernetes. Nginx is the public entry
point. The remaining Services are internal ClusterIP Services unless a local
fallback exposes them temporarily.

## Data Plane

The data plane handles user traffic:

1. The web browser requests the static web application from Nginx.
2. Wordle guesses go through `/wordle` to `wordle-api-service`.
3. Kubernetes distributes requests across the Wordle API pods.
4. Each pod validates the guess and applies the configured per-pod capacity
   model.

The Wordle API is stateless. A Deployment maintains the requested replica
count, and a Service selects healthy pods using the `app: wordle-api` label.

## Control Plane

The FastAPI control plane handles management traffic through Nginx `/api`:

- Kubernetes API reads provide status and metrics.
- Scale requests patch Deployment replica counts.
- Self-healing requests delete a managed pod; its Deployment replaces it.
- Load-test requests start work in the internal loadgen Service.

See [CONTROL-PLANE.md](CONTROL-PLANE.md) for permissions and endpoint details.

## Load Testing

The Go loadgen service sends concurrent requests through Nginx rather than
directly to Wordle pods. This makes its latency results represent the same
routing path used by users. It returns request counts, errors, status codes,
and latency percentiles.

See [LOADGEN.md](LOADGEN.md) for scheduling and result details.

## Routing and Same-Origin Access

Nginx maps public paths to internal Services:

```text
/         -> web-service:8080
/metrics  -> web-service:8080/metrics.html
/wordle   -> wordle-api-service:8080
/api      -> control-plane-service:5000
```

The browser uses `localhost:8080` for both pages and API calls. Nginx performs
internal routing, so the browser does not need CORS headers or direct access to
port `5000`.

## Kubernetes Concepts Demonstrated

- **Pod:** replaceable runtime unit containing one service container.
- **Deployment:** maintains replica count and replaces failed pods.
- **Service:** stable DNS name and load-balanced endpoint for selected pods.
- **Labels/selectors:** connect Services to the correct pod group.
- **Config through manifests:** environment variables define the capacity model
  and service behavior.
- **RBAC:** the control plane receives only the namespaced permissions it needs.
