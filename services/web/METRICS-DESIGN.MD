# Metrics Dashboard Design

## Goal

The dashboard gives students a readable view of how replica count and traffic
relate to system behavior. It is served by the existing web service at
`/metrics`, so students use the same entry point as the Wordle application.

The first version reports recent Kubernetes state, Nginx aggregate counters,
and a sampled request latency. It does not claim that the sampled latency is a
complete measurement of every student's traffic.

## Request Flow

```text
Browser /metrics
       |
       v
Nginx proxy
   |       |
   |       +--> /api/v1/* --> Control Plane --> Kubernetes API
   |
   +--> /metrics --> Web service (static HTML and JavaScript)
```

The Nginx configuration has two important routes:

- `/metrics` serves `metrics.html` from the web service.
- `/api/` forwards requests to `control-plane-service`.

This means the browser can request `/api/v1/metrics` without knowing the
control plane's internal Kubernetes DNS name.

## Same-Origin Requests and CORS

A browser applies a same-origin policy. An origin is the combination of
protocol, host, and port. For example, these are different origins:

```text
http://localhost:8080
http://localhost:5000
```

If the dashboard at port 8080 requested the API directly at port 5000, the
browser would treat that as a cross-origin request. The server would then need
to send CORS headers explicitly.

CORS, or Cross-Origin Resource Sharing, is a browser permission mechanism. It
can be configured, but it creates extra configuration and failure modes for a
student lab. Routing `/api` through the same Nginx origin avoids the problem:

```text
Dashboard: http://localhost:8080/metrics
API call:  http://localhost:8080/api/v1/metrics
```

The browser sees both as the same origin. This is simpler than exposing a
second port and adding CORS rules to FastAPI.

## Data Collection

The control plane owns collection because it already has Kubernetes API access.
A single background task samples data every five seconds. The browser does not
contact Kubernetes directly, and multiple browser tabs do not create multiple
Kubernetes watchers.

Each sample contains:

- Deployment desired, available, ready, and updated replica counts.
- Wordle API pod count and running pod count.
- Wordle API container restart count.
- Nginx active connections, accepted connections, handled connections, total
  requests, and reading/writing/waiting connection counts.
- `probe_latency_ms`, which is the duration of one control-plane request through
  Nginx to the Wordle API.

The probe is useful as a consistent baseline, but it is synthetic traffic. It
must not be interpreted as the average latency of all user requests.

## Ring Buffer

The control plane stores samples in a fixed-size ring buffer with capacity 300.
At one sample every five seconds, this retains approximately 25 minutes of
history. When the buffer is full, the oldest sample is discarded automatically.

This bounds memory use and keeps the dashboard response small. It also means
this is a recent-observation dashboard, not a permanent monitoring system.
Restarting the control-plane pod clears the buffer.

The endpoint is:

```text
GET /api/v1/metrics?limit=60
```

The response contains the samples, the sampling interval, buffer capacity, and
any current collector warning. A warning is returned as data so the dashboard
can show a stale or degraded state instead of silently displaying old values.

## Nginx Status

Nginx's `stub_status` module provides aggregate counters and connection state.
It does not provide per-request latency, CPU usage, or a breakdown by backend
pod. The control plane reads the internal `/nginx_status` endpoint from inside
the cluster.

This is intentionally lightweight, but it is not a replacement for a full
metrics stack such as Prometheus. A future version could add request timing to
Nginx access logs or have the load generator publish structured measurements.

## Dashboard UI

The dashboard is static HTML and JavaScript served by Nginx through the web
service. Chart.js renders:

- Sampled latency over time.
- Ready versus desired Wordle API replicas over time.

The summary cards show the latest latency, replica state, Nginx request count,
and pod restart count. Recent control-plane events appear below the charts.
The page polls every five seconds and shows explicit waiting, stale, and error
states.

Chart.js is loaded from a pinned CDN URL for now. This keeps the repository
small and avoids adding a frontend build system. The tradeoff is that a browser
without access to the CDN cannot render charts. The dashboard detects that case
and displays a warning. Vendoring the Chart.js browser bundle is a good future
improvement for offline student environments.

## Why Polling Instead of WebSockets?

Polling is sufficient for a five-second dashboard and is easy to reason about:

- No persistent connection management is required.
- Nginx needs no WebSocket configuration.
- Refreshing the page naturally starts a fresh client.
- The control plane remains the single collector.

WebSockets or Server-Sent Events could make updates feel more immediate, but
would add lifecycle, proxy, and reconnect behavior that is not needed yet.

## Load-Test Measurement

The load generator now runs as an internal Kubernetes Service. The control
plane starts bounded tests through `/api/v1/load-tests` and exposes their
results through the same API. Each result includes request counts, status-code
counts, errors, minimum and average latency, and p50/p95 latency.

The load generator sends requests through Nginx, so these results represent a
realistic client path rather than only the control plane's health probe. Tests
are intentionally limited to one active run, 1-1000 requests per second, and
1-300 seconds to keep the student cluster responsive.

The dashboard also displays the latest completed test's status, p50, p95, and
success rate, plus a history chart of completed-test percentiles.

The current synthetic probe should remain labeled as sampled latency. It is a
health-oriented signal and avoids presenting incomplete data as if it were a
full request trace.
