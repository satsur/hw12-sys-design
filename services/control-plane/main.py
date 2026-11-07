from datetime import datetime, timezone
from typing import Any

from fastapi import FastAPI, HTTPException
from kubernetes import client, config
from pydantic import BaseModel, Field

app = FastAPI(title="Control Plane", version="1.0.0")

DEFAULT_DEPLOYMENTS = ("wordle-api", "nginx-proxy", "web")


class ScaleRequest(BaseModel):
    deployment: str = "wordle-api"
    replicas: int = Field(ge=0, le=50)


class SelfHealRequest(BaseModel):
    deployment: str = "wordle-api"


class LoadBalancingRequest(BaseModel):
    algorithm: str


_events: list[dict[str, Any]] = []


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
    return {
        "status": "placeholder",
        "algorithm": request.algorithm,
        "message": "Load-balancing changes are not implemented yet.",
    }


@app.get("/api/v1/events")
def events(limit: int = 20) -> dict[str, Any]:
    return {"events": _events[-max(1, min(limit, 100)) :]}
