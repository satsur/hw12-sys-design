# Load Generator

## Purpose

The load generator is a small Go HTTP service that creates controlled traffic
through the Nginx entry point. It is not called directly by the browser. The
control plane starts tests and retrieves results through the load generator's
internal Kubernetes Service.

```text
CLI or dashboard
      |
      v
Control Plane :5000
      |
      v
Loadgen Service :8080
      |
      v
Nginx Service :8080/wordle
      |
      v
Wordle API pods
```

This path is important for experiments: generated requests experience the
same Nginx routing path as normal users.

## Kubernetes Configuration

The load generator is deployed by `k8s/loadgen-deployment.yaml` as:

- Deployment: `loadgen`
- Service: `loadgen-service`
- Internal port: `8080`
- Target URL: `http://nginx-service:8080/wordle`

The target URL is supplied through `LOADGEN_TARGET_URL`. Keeping it in an
environment variable makes the Go service reusable without changing source
code or rebuilding the image.

## Per-Pod Capacity

Each Wordle API pod has a deterministic capacity model configured in
`k8s/wordle-deployment.yaml`:

```text
WORDLE_MAX_IN_FLIGHT=4
WORDLE_DELAY_MS=50
```

The API uses a semaphore with four slots. A valid request waits for a slot,
sleeps for the configured processing delay, and releases the slot before
returning. With four slots and a 50 ms service time, one pod has an approximate
steady-state capacity of 80 requests per second. Three replicas provide about
240 requests per second before queueing becomes significant; five replicas
provide about 400 requests per second.

This is an experimental model, not a claim about physical machine capacity.
It makes comparisons repeatable across student machines. Change one of these
values only when designing a new experiment, and keep it fixed while comparing
replica counts.

## Test Request Schema

The control plane accepts:

```http
POST /api/v1/load-tests
Content-Type: application/json
```

```json
{
  "rps": 25,
  "duration": 30,
  "word": "hello"
}
```

| Field | Type | Constraints | Meaning |
| --- | --- | --- | --- |
| `rps` | integer | 1-1000 | Target requests per second. |
| `duration` | integer | 1-300 | Test length in seconds. |
| `word` | string | 1-20 characters | Wordle query sent with every request. |

The control plane validates this request and forwards it to the internal
loadgen endpoint:

```http
POST http://loadgen-service:8080/start
```

The loadgen responds immediately with HTTP `202` and a test record. Tests run
asynchronously so the control-plane request does not remain open for the full
test duration.

## Result Schema

Start response example:

```json
{
  "id": "test-1794283375893225559",
  "status": "running",
  "requested_rps": 25,
  "duration_seconds": 30,
  "word": "hello",
  "started_at": "2026-11-10T04:02:55Z",
  "requests_sent": 0,
  "successes": 0,
  "errors": 0,
  "status_codes": {},
  "min_latency_ms": 0,
  "average_latency_ms": 0,
  "p50_latency_ms": 0,
  "p95_latency_ms": 0
}
```

When the test completes, the same record contains:

- `requests_sent`: all completed HTTP attempts.
- `successes`: responses with a 2xx status.
- `errors`: network failures and non-2xx responses.
- `status_codes`: response count grouped by status code.
- `min_latency_ms`: fastest completed request.
- `average_latency_ms`: arithmetic mean of completed request latency.
- `p50_latency_ms`: median request latency.
- `p95_latency_ms`: latency below which 95 percent of requests completed.
- `finished_at`: completion timestamp.

The control plane exposes these loadgen endpoints:

```http
POST /api/v1/load-tests
GET  /api/v1/load-tests
GET  /api/v1/load-tests/{test_id}
```

The CLI maps to them as:

```bash
sysctl load-test --rps 25 --duration 30 --word hello
sysctl load-tests
sysctl load-test-status <test-id>
```

## How Requests Are Scheduled

The Go service creates a ticker with an interval of approximately
`1 second / rps`. Each tick starts one goroutine. The goroutine:

1. Builds the Wordle URL and query string.
2. Starts a timer.
3. Sends an HTTP request through Nginx.
4. Records elapsed time and response status.
5. Closes the response body.

At the deadline, the generator stops creating requests, waits for in-flight
requests to finish, sorts the recorded latencies, and calculates the summary
percentiles.

The implementation allows concurrent requests because a slow request must not
prevent later scheduled requests from being sent. The result has locks because
the test runs in the background while the control plane may be polling it.

## Why Only One Test Runs at a Time?

The current loadgen instance rejects a second test with HTTP `409` while one is
active. This keeps experiments understandable and prevents two students or two
CLI terminals from unknowingly mixing traffic in the same result.

The in-memory result store is intentionally simple for the lab. Results vanish
when the loadgen pod restarts. A future version could persist results or run
each test as a Kubernetes Job.

## Interpreting Results

A test result describes traffic observed by the load generator, not only the
control plane's health probe. To compare configurations, keep these values
constant:

- RPS
- Duration
- Word
- Replica count before the test

Change one experimental variable at a time. Compare p95 as well as average
latency because averages can hide a slow tail of requests.

The dashboard now displays the latest completed test's status, p50, p95, and
success rate, plus a history chart of completed-test percentiles. Algorithm
switching is intentionally not part of this lab; compare replica count and
per-pod capacity instead.
