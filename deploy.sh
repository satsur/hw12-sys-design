#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

docker build -t wordle-api:latest ./services/wordle-api
docker build -t nginx-proxy:latest ./services/nginx
docker build -t web:latest ./services/web
docker build -t control-plane:latest ./services/control-plane
docker build -t loadgen:latest ./services/loadgen


if [[ "$(kubectl config current-context)" != "docker-desktop" ]]; then
	printf 'ERROR: This deployment script requires the docker-desktop Kubernetes context.\n' >&2
	kubectl config current-context >&2
	exit 1
fi

KUBE_NODE="$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')"
if ! docker inspect "$KUBE_NODE" >/dev/null 2>&1; then
	printf 'ERROR: Docker Desktop Kubernetes node "%s" is not available to Docker.\n' "$KUBE_NODE" >&2
	exit 1
fi
printf 'Loading locally built images into Docker Desktop Kubernetes node %s.\n' "$KUBE_NODE"
for image in wordle-api:latest nginx-proxy:latest web:latest control-plane:latest loadgen:latest; do
	docker save "$image" | docker exec -i "$KUBE_NODE" ctr -n k8s.io images import -
done

kubectl apply -f k8s/
kubectl delete configmap nginx-proxy-config --ignore-not-found

for deployment in wordle-api nginx-proxy web control-plane loadgen; do
	kubectl rollout restart "deployment/$deployment"
	kubectl rollout status "deployment/$deployment"
done