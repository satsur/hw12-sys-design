# Distributed Systems Lab

A Kubernetes-based Wordle system for studying microservices, reverse proxies,
scaling, self-healing, control planes, and load testing.

## What You Will Run

```text
Browser / CLI
     |
     v
Nginx entry point
  |      |       |
  v      v       v
Web   Wordle   Control Plane
                  |
                  v
               Loadgen
```

- **Web:** static Wordle interface and metrics dashboard.
- **Nginx:** the single HTTP entry point and reverse proxy.
- **Wordle API:** stateless Go service replicated across Kubernetes pods.
- **Control plane:** FastAPI service that reads Kubernetes state, scales pods,
  triggers self-healing tests, collects metrics, and starts load tests.
- **Loadgen:** internal Go service that sends controlled traffic through Nginx.

## Prerequisites

Install Docker Desktop with Kubernetes enabled, `kubectl`, and Python 3.12 or
newer. Python is needed for the native PowerShell CLI and for formatted Bash
summaries; the Bash CLI still works without it but prints raw JSON.

On Windows, install Python from [python.org](https://www.python.org/downloads/)
or with `winget`:

```powershell
winget install Python.Python.3.12
```

Enable the installer option that adds Python to `PATH`, then verify:

```powershell
python --version
```

Verify that Docker Desktop Kubernetes is running:

```bash
kubectl config use-context docker-desktop
kubectl get nodes
```

The deployment scripts are designed for this Docker Desktop environment and do
not require the kind CLI.

## Start the System

From the repository root, run the platform-appropriate script:

```bash
# WSL, Linux, or macOS shell
bash ./deploy.sh
```

```powershell
# Windows PowerShell or Command Prompt
.\deploy.bat
```

The script builds all local images, imports them into Docker Desktop's
Kubernetes runtime, applies the manifests, and waits for every Deployment to
roll out. Local images use `imagePullPolicy: Never`, so a missing image import doesn't silently use an older or remote image when deploying updates.

## Access the Application

Nginx is configured as the public `LoadBalancer` Service on port `8080`:

```text
Wordle:   http://localhost:8080/
Metrics:  http://localhost:8080/metrics
Health:   http://localhost:8080/api/v1/health
```

Docker Desktop versions differ in how they publish local LoadBalancer Services.
If `localhost:8080` is not reachable, use this fallback in another terminal:

```bash
kubectl port-forward service/nginx-service 8080:8080
```

The shell CLI uses the same public path by default:

```bash
bash services/shell-cli/sysctl.sh status
```

## Common CLI Commands

The Bash CLI uses `curl` directly. When Python 3 is available, it formats JSON
responses into readable summaries; without Python it falls back to the raw API
JSON. From the repository root, use [services/shell-cli/sysctl.sh](services/shell-cli/sysctl.sh):

```bash
bash services/shell-cli/sysctl.sh --help
bash services/shell-cli/sysctl.sh status
bash services/shell-cli/sysctl.sh scale wordle-api 5
bash services/shell-cli/sysctl.sh self-heal wordle-api
bash services/shell-cli/sysctl.sh events 20
bash services/shell-cli/sysctl.sh load-test --rps 25 --duration 30
bash services/shell-cli/sysctl.sh load-tests
bash services/shell-cli/sysctl.sh load-test-status <test-id>
```

Create a temporary Bash alias from the repository root:

```bash
alias sysctl='bash "$(pwd)/services/shell-cli/sysctl.sh"'
sysctl status
```

PowerShell users can run the equivalent Python CLI without Bash or WSL:

```powershell
python services/shell-cli/sysctl.py status
python services/shell-cli/sysctl.py load-test --rps 400 --duration 30
python services/shell-cli/sysctl.py load-tests
python services/shell-cli/sysctl.py load-test-status <test-id>
```

The Python and Bash CLIs use the same control-plane API, default to
`http://localhost:8080`, and share the same human-readable output formatter.
Set `CONTROL_PLANE_URL` when using a different endpoint.

The CLI controls the control plane through Nginx's `/api` route. The control
plane and load generator remain internal Kubernetes Services.

## Key Experiments

The Wordle API has a deterministic per-pod capacity model:

```text
WORDLE_MAX_IN_FLIGHT=4
WORDLE_DELAY_MS=50
```

A pod can process four requests concurrently, with each request taking about
50 ms. This creates a reproducible capacity target across student machines.
Compare tests such as:

1. Run a test with three Wordle replicas.
2. Record request count, errors, p50, and p95 latency.
3. Scale to five replicas.
4. Repeat the same RPS, duration, and word.
5. Compare the results.

The system also demonstrates Kubernetes self-healing: deleting a managed pod
causes its Deployment to create a replacement.

## Request Routing

```text
localhost:8080/
  -> nginx-service
  -> web-service

localhost:8080/wordle
  -> nginx-service
  -> wordle-api-service
  -> one of the Wordle API pods

localhost:8080/api/v1/*
  -> nginx-service
  -> control-plane-service:5000

control-plane-service
  -> loadgen-service:8080
  -> nginx-service:8080/wordle
```

Keeping the browser and API under the same host and port avoids CORS
configuration. Kubernetes Services provide stable DNS names while pods remain
replaceable.

## Documentation Map

- [OUTPUT.md](OUTPUT.md): build, deployment, access, and CLI/API reference.
- [ENTRYPOINT.md](ENTRYPOINT.md): Docker Desktop entry point and networking.
- [ARCHITECTURE.md](ARCHITECTURE.md): system architecture and Kubernetes
  concepts.
- [CONTROL-PLANE.md](CONTROL-PLANE.md): control-plane responsibilities,
  permissions, and API.
- [LOADGEN.md](LOADGEN.md): load-test request/result schemas and experiments.
- [METRICS-DESIGN.MD](METRICS-DESIGN.MD): metrics collection and dashboard
  design.
- [TODO.md](TODO.md): remaining cleanup work.

## Troubleshooting

Check the cluster and workloads:

```bash
kubectl config current-context
kubectl get nodes
kubectl get pods
kubectl get services
```

If a pod reports `ErrImageNeverPull`, rerun the deployment script so the local
images are imported into Docker Desktop Kubernetes. If the CLI cannot connect,
confirm that either `localhost:8080` is reachable or the Nginx port-forward is
running.
