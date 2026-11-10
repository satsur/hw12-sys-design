import asyncio
import json
import re
import time
from collections import deque
from contextlib import asynccontextmanager, suppress
from datetime import datetime, timezone
from threading import Lock
from typing import Any
from urllib.parse import urlencode
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

from fastapi import FastAPI, HTTPException
from kubernetes import client, config
from pydantic import BaseModel, Field

DEFAULT_DEPLOYMENTS = ("wordle-api", "nginx-proxy", "web")
METRICS_INTERVAL_SECONDS = 5
METRICS_BUFFER_SIZE = 300
NGINX_SERVICE_URL = "http://nginx-service:80"
LOADGEN_SERVICE_URL = "http://loadgen-service:8080"
NGINX_CONFIGMAP_NAME = "nginx-proxy-config"
SUPPORTED_LOAD_BALANCERS = {"round_robin", "least_conn"}


class ScaleRequest(BaseModel):
    deployment: str = "wordle-api"
    replicas: int = Field(ge=0, le=50)


class SelfHealRequest(BaseModel):
    deployment: str = "wordle-api"


class LoadBalancingRequest(BaseModel):
    algorithm: str


class LoadTestRequest(BaseModel):
    rps: int = Field(ge=1, le=1000)
    duration: int = Field(ge=1, le=300)
    word: str = Field(default="hello", min_length=1, max_length=20)


_events: list[dict[str, Any]] = []
_metrics: deque[dict[str, Any]] = deque(maxlen=METRICS_BUFFER_SIZE)
_metrics_lock = Lock()
_collector_error: str | None = None


@asynccontextmanager
async def lifespan(_: FastAPI):
    collector = asyncio.create_task(collect_metrics())
    try:
        yield
    finally:
        collector.cancel()
        with suppress(asyncio.CancelledError):
            await collector


app = FastAPI(title="Control Plane", version="1.0.0", lifespan=lifespan)


def record_event(event_type: str, message: str, **details: Any) -> None:
    _events.append(
        {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "type": event_type,
            "message": message,
            **details,
        }
    )
    del _events[:-100]


def kubernetes_clients() -> tuple[client.AppsV1Api, client.CoreV1Api]:
    try:
        config.load_incluster_config()
    except config.ConfigException:
        config.load_kube_config()
    return client.AppsV1Api(), client.CoreV1Api()


def deployment_summary(apps: client.AppsV1Api, name: str) -> dict[str, Any]:
    deployment = apps.read_namespaced_deployment(name, "default")
    status = deployment.status
    return {
        "name": name,
        "desired_replicas": deployment.spec.replicas or 0,
        "available_replicas": status.available_replicas or 0,
        "ready_replicas": status.ready_replicas or 0,
        "updated_replicas": status.updated_replicas or 0,
    }


def pod_restart_count(pod: client.V1Pod) -> int:
    return sum(
        container.restart_count or 0
        for container in (pod.status.container_statuses or [])
    )


def nginx_status() -> dict[str, int]:
    with urlopen(f"{NGINX_SERVICE_URL}/nginx_status", timeout=2) as response:
        lines = [line.decode().strip() for line in response.readlines()]

    request_counts = [int(value) for value in lines[2].split()]
    connection_counts: dict[str, int] = {}
    connection_tokens = lines[3].split()
    for index, token in enumerate(connection_tokens[:-1]):
        if token.endswith(":"):
            connection_counts[token[:-1].lower()] = int(connection_tokens[index + 1])
    return {
        "active_connections": int(lines[0].split(":", 1)[1].strip()),
        "accepts": request_counts[0],
        "handled": request_counts[1],
        "requests": request_counts[2],
        "reading": connection_counts["reading"],
        "writing": connection_counts["writing"],
        "waiting": connection_counts["waiting"],
    }


def probe_latency_ms() -> float:
    query = urlencode({"word": "hello"})
    started = time.perf_counter()
    with urlopen(f"{NGINX_SERVICE_URL}/wordle?{query}", timeout=10):
        pass
    return round((time.perf_counter() - started) * 1000, 2)


def loadgen_request(method: str, path: str, payload: dict[str, Any] | None = None) -> Any:
    body = None if payload is None else json.dumps(payload).encode()
    request = Request(
        f"{LOADGEN_SERVICE_URL}{path}",
        data=body,
        headers={"Content-Type": "application/json"} if body else {},
        method=method,
    )
    try:
        with urlopen(request, timeout=5) as response:
            return json.loads(response.read())
    except HTTPError as error:
        detail = error.read().decode() or "load generator returned an error"
        raise HTTPException(status_code=error.code, detail=detail) from error
    except URLError as error:
        raise HTTPException(status_code=503, detail=f"load generator unavailable: {error.reason}") from error


def collect_metric_sample() -> None:
    global _collector_error
    try:
        apps, core = kubernetes_clients()
        deployments = [
            deployment_summary(apps, name) for name in DEFAULT_DEPLOYMENTS
        ]
        pods = core.list_namespaced_pod(
            "default", label_selector="app=wordle-api"
        ).items
        sample: dict[str, Any] = {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "deployments": deployments,
            "wordle_api": {
                "pod_count": len(pods),
                "running_pods": sum(pod.status.phase == "Running" for pod in pods),
                "restart_count": sum(pod_restart_count(pod) for pod in pods),
            },
            "probe_latency_ms": probe_latency_ms(),
        }
        try:
            sample["nginx"] = nginx_status()
        except Exception as error:
            sample["nginx_error"] = str(error)

        with _metrics_lock:
            _metrics.append(sample)
            _collector_error = None
    except Exception as error:
        _collector_error = str(error)


async def collect_metrics() -> None:
    while True:
        await asyncio.to_thread(collect_metric_sample)
        await asyncio.sleep(METRICS_INTERVAL_SECONDS)


@app.get("/api/v1/health")
def health() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/api/v1/status")
def status() -> dict[str, Any]:
    apps, core = kubernetes_clients()
    deployments = []
    for name in DEFAULT_DEPLOYMENTS:
        try:
            deployments.append(deployment_summary(apps, name))
        except client.exceptions.ApiException as error:
            if error.status != 404:
                raise

    pods = core.list_namespaced_pod("default", label_selector="app=wordle-api").items
    return {
        "deployments": deployments,
        "wordle_api_pods": [
            {"name": pod.metadata.name, "phase": pod.status.phase}
            for pod in pods
        ],
        "events": _events[-20:],
    }


@app.post("/api/v1/scale")
def scale(request: ScaleRequest) -> dict[str, Any]:
    apps, _ = kubernetes_clients()
    try:
        deployment = apps.patch_namespaced_deployment_scale(
            request.deployment,
            "default",
            {"spec": {"replicas": request.replicas}},
        )
    except client.exceptions.ApiException as error:
        raise HTTPException(status_code=error.status or 500, detail=error.body) from error

    record_event(
        "scale",
        f"Scaled {request.deployment} to {request.replicas} replicas",
        deployment=request.deployment,
        replicas=request.replicas,
    )
    return {
        "deployment": request.deployment,
        "replicas": deployment.spec.replicas,
        "status": "accepted",
    }


@app.post("/api/v1/self-heal")
def self_heal(request: SelfHealRequest) -> dict[str, Any]:
    _, core = kubernetes_clients()
    try:
        pods = core.list_namespaced_pod(
            "default", label_selector=f"app={request.deployment}"
        ).items
        running = [pod for pod in pods if pod.status.phase == "Running"]
        if not running:
            raise HTTPException(status_code=409, detail="No running pod is available")
        victim = sorted(running, key=lambda pod: pod.metadata.name)[0]
        core.delete_namespaced_pod(victim.metadata.name, "default")
    except client.exceptions.ApiException as error:
        raise HTTPException(status_code=error.status or 500, detail=error.body) from error

    record_event(
        "self-heal",
        f"Deleted pod {victim.metadata.name}",
        deployment=request.deployment,
        pod=victim.metadata.name,
    )
    return {
        "deployment": request.deployment,
        "deleted_pod": victim.metadata.name,
        "status": "accepted",
    }


@app.post("/api/v1/load-balancing")
def load_balancing(request: LoadBalancingRequest) -> dict[str, str]:
    if request.algorithm not in SUPPORTED_LOAD_BALANCERS:
        raise HTTPException(
            status_code=400,
            detail=f"algorithm must be one of: {', '.join(sorted(SUPPORTED_LOAD_BALANCERS))}",
        )

    apps, core = kubernetes_clients()
    try:
        config_map = core.read_namespaced_config_map(NGINX_CONFIGMAP_NAME, "default")
        config_text = (config_map.data or {}).get("nginx_proxy.conf", "")
        directive = "    least_conn;\n" if request.algorithm == "least_conn" else ""
        updated_config, replacements = re.subn(
            r"(upstream wordle-api \{\n)(?:\s+least_conn;\n)?(\s+server wordle-api-service:8080;)",
            rf"\1{directive}\2",
            config_text,
            count=1,
        )
        if replacements != 1:
            raise HTTPException(status_code=500, detail="Nginx upstream configuration was not recognized")
        core.patch_namespaced_config_map(
            NGINX_CONFIGMAP_NAME,
            "default",
            {"data": {"nginx_proxy.conf": updated_config}},
        )
        apps.patch_namespaced_deployment(
            "nginx-proxy",
            "default",
            {
                "spec": {
                    "template": {
                        "metadata": {
                            "annotations": {
                                "control-plane/algorithm-updated": datetime.now(timezone.utc).isoformat()
                            }
                        }
                    }
                }
            },
        )
    except client.exceptions.ApiException as error:
        raise HTTPException(status_code=error.status or 500, detail=error.body) from error

    record_event(
        "load-balancing",
        f"Set Nginx load-balancing algorithm to {request.algorithm}",
        algorithm=request.algorithm,
    )
    return {
        "status": "accepted",
        "algorithm": request.algorithm,
        "message": "Nginx rollout started with the new algorithm.",
    }


@app.post("/api/v1/load-tests")
def start_load_test(request: LoadTestRequest) -> Any:
    result = loadgen_request("POST", "/start", request.model_dump())
    record_event(
        "load-test",
        f"Started load test {result['id']} at {request.rps} RPS for {request.duration} seconds",
        test_id=result["id"],
        rps=request.rps,
        duration=request.duration,
    )
    return result


@app.get("/api/v1/load-tests")
def list_load_tests() -> Any:
    return loadgen_request("GET", "/tests")


@app.get("/api/v1/load-tests/{test_id}")
def get_load_test(test_id: str) -> Any:
    return loadgen_request("GET", f"/tests/{test_id}")


@app.get("/api/v1/events")
def events(limit: int = 20) -> dict[str, Any]:
    return {"events": _events[-max(1, min(limit, 100)) :]}


@app.get("/api/v1/metrics")
def metrics(limit: int = 60) -> dict[str, Any]:
    with _metrics_lock:
        samples = list(_metrics)[-max(1, min(limit, METRICS_BUFFER_SIZE)) :]
        collector_error = _collector_error
    return {
        "samples": samples,
        "sample_interval_seconds": METRICS_INTERVAL_SECONDS,
        "buffer_capacity": METRICS_BUFFER_SIZE,
        "collector_error": collector_error,
    }
