# Control Plane

The control plane is the Python FastAPI service in
`services/control-plane`. It manages the Kubernetes application rather than
serving Wordle guesses directly.

## Responsibilities

- Read Deployment and pod state from the Kubernetes API.
- Scale the `wordle-api` Deployment.
- Delete a selected pod for self-healing demonstrations.
- Collect recent Kubernetes and Nginx observations in a bounded ring buffer.
- Start load tests through the internal loadgen Service.
- Expose the API used by both CLIs and the metrics dashboard.

The control plane runs in Kubernetes as Deployment `control-plane` and listens
inside the cluster on port `5000`. Its Service is `control-plane-service:5000`.
It is reached externally through Nginx's `/api` route:

```text
localhost:8080/api/v1/status
    -> nginx-service:8080
    -> control-plane-service:5000
```

This keeps browser requests same-origin and avoids CORS configuration.

## API

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/health` | Process health check. |
| `GET` | `/api/v1/status` | Deployment and Wordle pod state. |
| `GET` | `/api/v1/events?limit=20` | Recent scale, recovery, and load-test events. |
| `GET` | `/api/v1/metrics?limit=60` | Recent ring-buffer samples. |
| `POST` | `/api/v1/scale` | Change a Deployment replica count. |
| `POST` | `/api/v1/self-heal` | Delete one running pod selected by deployment label. |
| `POST` | `/api/v1/load-tests` | Start a bounded load test. |
| `GET` | `/api/v1/load-tests` | List load-test results. |
| `GET` | `/api/v1/load-tests/{test_id}` | Retrieve one result. |

Example scale request:

```json
{"deployment":"wordle-api","replicas":5}
```

Example load-test request:

```json
{"rps":400,"duration":30,"word":"hello"}
```

## Kubernetes Access

Inside the cluster, the service account `control-plane` has a namespaced Role
that permits:

- Reading Deployments and pods.
- Updating Deployment scale values.
- Deleting pods for the self-healing demonstration.

The service does not need cluster-wide permissions. The control plane loads
in-cluster Kubernetes credentials when running in a pod and falls back to the
local kubeconfig for development runs.

## Metrics Collection

A background task samples every five seconds. Each sample records Deployment
readiness, Wordle pod count and restarts, Nginx connection/request counters,
and one labeled synthetic latency probe through Nginx.

Samples are stored in a 300-entry in-memory ring buffer. At the five-second
interval this retains about 25 minutes. Restarting the control-plane pod clears
it; the buffer is intended for a teaching dashboard, not durable monitoring.

The control plane also forwards load-test requests to `loadgen-service:8080`.
Loadgen results contain request totals, status codes, errors, average latency,
and p50/p95 latency. See [LOADGEN.md](LOADGEN.md) for that service's logic and
schema.
