# Local Runbook

This project runs in Docker Desktop Kubernetes. Nginx is the public
`LoadBalancer` on port `8080`; Docker Desktop versions that do not publish the
LoadBalancer to the host can use the port-forward fallback below.

## Prerequisites

All platforms need Docker Desktop with Kubernetes enabled and `kubectl`
configured for the Docker Desktop context.

- Windows: Docker Desktop, `kubectl`, and either PowerShell or WSL.
- WSL: Docker Desktop integration enabled for the WSL distribution and
  `kubectl` installed inside WSL.
- Linux/macOS: Docker Desktop and `kubectl` installed in the native shell.

Verify the selected cluster before deploying:

```bash
kubectl config current-context
kubectl get nodes
```

## Build and Deploy

From PowerShell on Windows, run [deploy.bat](deploy.bat):

```powershell
.\deploy.bat
```

From WSL, Linux, or macOS, run [deploy.sh](deploy.sh):

```bash
bash ./deploy.sh
```

The scripts build five local images, import them into Docker Desktop Kubernetes,
apply `k8s/`, restart the Deployments, and wait for rollout completion. The
manifests use `imagePullPolicy: Never`, so missing local images fail visibly.

## Access

After deployment, use these URLs when the LoadBalancer is host-reachable:

```text
Wordle:   http://localhost:8080/
Metrics:  http://localhost:8080/metrics
API:      http://localhost:8080/api/v1/health
CLI:      bash services/shell-cli/sysctl.sh status
```

The CLI uses the Nginx `/api` route by default. See [ENTRYPOINT.md](ENTRYPOINT.md)
for the network path and Docker Desktop troubleshooting.

## Port-Forward Fallback

Port forwarding is a local process. Keep each `kubectl port-forward` command
running in its own terminal while using the service.

### Windows PowerShell

Run these commands in separate PowerShell terminals:

```powershell
kubectl port-forward --address 127.0.0.1 service/control-plane-service 5000:5000
kubectl port-forward --address 127.0.0.1 service/nginx-service 8080:8080
```

The control plane is available at `http://localhost:5000`; the web application
is available at `http://localhost:8080`, with the dashboard at
`http://localhost:8080/metrics` and its same-origin API at
`http://localhost:8080/api/v1/metrics`.

### WSL

Run both `kubectl port-forward` commands inside WSL, not in a separate Windows
PowerShell terminal. This keeps the port-forward listener and Bash CLI in the
same network namespace:

```bash
kubectl port-forward --address 127.0.0.1 service/control-plane-service 5000:5000
kubectl port-forward --address 127.0.0.1 service/nginx-service 8080:8080
```

Use `http://localhost:5000` for the CLI and open `http://localhost:8080` or
`http://localhost:8080/metrics` in a Windows browser. If `kubectl` is only available on Windows, bind the Windows
forward to `0.0.0.0` and use the Docker Desktop host gateway from WSL instead:

```powershell
kubectl port-forward --address 0.0.0.0 service/control-plane-service 5000:5000
```

```bash
export CONTROL_PLANE_URL=http://host.docker.internal:5000
```

### Linux and macOS

Run these commands in separate terminals:

```bash
kubectl port-forward --address 127.0.0.1 service/control-plane-service 5000:5000
kubectl port-forward --address 127.0.0.1 service/nginx-service 8080:8080
```

Use `http://localhost:5000` for the CLI and `http://localhost:8080` or
`http://localhost:8080/metrics` for the web application and dashboard.

## CLI

The Bash CLI uses `curl` and optionally Python 3 for formatted summaries. The
Python CLI provides the same commands for PowerShell users. See the README for
installation and examples.

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

Create a temporary Bash alias from the repository root if desired:

```bash
alias sysctl='bash "$(pwd)/services/shell-cli/sysctl.sh"'
sysctl status
```

The CLI prints `SUCCESS` for a 2xx response and returns a nonzero exit code for
connection, HTTP, or argument errors. API details are in
[CONTROL-PLANE.md](CONTROL-PLANE.md); load-test details are in
[LOADGEN.md](LOADGEN.md).
