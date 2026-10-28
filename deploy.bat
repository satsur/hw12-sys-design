docker build -t wordle-api:latest ./services/wordle-api

docker build -t nginx-proxy:latest ./services/nginx

docker build -t web:latest ./services/web

kubectl apply -f k8s/

kubectl rollout restart deployment/wordle-api
kubectl rollout restart deployment/nginx-proxy
kubectl rollout restart deployment/web