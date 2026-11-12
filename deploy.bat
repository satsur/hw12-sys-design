docker build -t wordle-api:latest ./services/wordle-api
docker build -t nginx-proxy:latest ./services/nginx
docker build -t web:latest ./services/web
docker build -t control-plane:latest ./services/control-plane
docker build -t loadgen:latest ./services/loadgen

for /f "delims=" %%c in ('kubectl config current-context') do set "KUBE_CONTEXT=%%c"
if not "%KUBE_CONTEXT%"=="docker-desktop" (
	echo ERROR: This deployment script requires the docker-desktop Kubernetes context.
	echo Current context: %KUBE_CONTEXT%
	exit /b 1
)

for /f "delims=" %%n in ('kubectl get nodes -o jsonpath^="{.items[0].metadata.name}"') do set "KUBE_NODE=%%n"
docker inspect "%KUBE_NODE%" >nul 2>nul
if errorlevel 1 (
	echo ERROR: Docker Desktop Kubernetes node "%KUBE_NODE%" is not available to Docker.
	exit /b 1
)
echo Loading locally built images into Docker Desktop Kubernetes node "%KUBE_NODE%".
for %%i in (wordle-api:latest nginx-proxy:latest web:latest control-plane:latest loadgen:latest) do (
	docker save %%i | docker exec -i "%KUBE_NODE%" ctr -n k8s.io images import -
	if errorlevel 1 exit /b 1
)

kubectl apply -f k8s/
kubectl delete configmap nginx-proxy-config --ignore-not-found

for %%d in (wordle-api nginx-proxy web control-plane loadgen) do (
	kubectl rollout restart deployment/%%d
	if errorlevel 1 exit /b 1
	kubectl rollout status deployment/%%d
	if errorlevel 1 exit /b 1
)