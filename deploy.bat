docker build -t wordle-api:latest ./services/wordle-api
docker build -t nginx-proxy:latest ./services/nginx
docker build -t web:latest ./services/web
docker build -t control-plane:latest ./services/control-plane
kind load docker-image wordle-api:latest nginx-proxy:latest web:latest control-plane:latest
kubectl apply -f k8s/
kubectl rollout status deployment/wordle-api
kubectl rollout status deployment/control-plane