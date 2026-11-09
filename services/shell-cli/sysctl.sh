#!/usr/bin/env bash
set -euo pipefail

CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://localhost:5000}"

usage() {
	cat <<'EOF'
Usage: sysctl <command> [arguments]

Commands:
  status
  events [limit]
  scale <deployment> <replicas>
  self-heal [deployment]
  load-balancing <algorithm>
EOF
}

request() {
	local method="$1"
	local path="$2"
	local action="$3"
	local body="${4:-}"
	local url="${CONTROL_PLANE_URL%/}${path}"
	local response
	local curl_status=0
	local http_status
	local response_body

	if [[ -n "$body" ]]; then
		response=$(curl --silent --show-error --connect-timeout 5 \
			--request "$method" \
			--header "Content-Type: application/json" \
			--data "$body" \
			--write-out $'\n%{http_code}' "$url" 2>&1) || curl_status=$?
	else
		response=$(curl --silent --show-error --connect-timeout 5 \
			--request "$method" "$url" \
			--write-out $'\n%{http_code}' 2>&1) || curl_status=$?
	fi

	if [[ "$curl_status" -ne 0 ]]; then
		printf 'ERROR: Could not reach the control plane at %s.\n' "$CONTROL_PLANE_URL" >&2
		printf '       Start the port-forward and check CONTROL_PLANE_URL, then retry.\n' >&2
		return "$curl_status"
	fi

	http_status="${response##*$'\n'}"
	response_body="${response%$'\n'*}"
	if [[ "$http_status" -lt 200 || "$http_status" -ge 300 ]]; then
		printf 'ERROR: %s failed (HTTP %s).\n' "$action" "$http_status" >&2
		[[ -n "$response_body" ]] && printf '%s\n' "$response_body" >&2
		return 1
	fi

	printf 'SUCCESS: %s.\n' "$action"
	[[ -n "$response_body" ]] && printf '%s\n' "$response_body"
}

command="${1:-}"
case "$command" in
	status)
		[[ "$#" -eq 1 ]] || { usage >&2; exit 2; }
		request GET "/api/v1/status" "Retrieved cluster status"
		;;
	events)
		[[ "$#" -le 2 ]] || { usage >&2; exit 2; }
		limit="${2:-20}"
		request GET "/api/v1/events?limit=${limit}" "Retrieved recent events"
		;;
	scale)
		[[ "$#" -eq 3 ]] || { usage >&2; exit 2; }
		request POST "/api/v1/scale" \
			"Scaled deployment $2 to $3 replicas" \
			"{\"deployment\":\"$2\",\"replicas\":$3}"
		;;
	self-heal)
		[[ "$#" -le 2 ]] || { usage >&2; exit 2; }
		deployment="${2:-wordle-api}"
		request POST "/api/v1/self-heal" \
			"Triggered self-healing for $deployment" \
			"{\"deployment\":\"$deployment\"}"
		;;
	load-balancing)
		[[ "$#" -eq 2 ]] || { usage >&2; exit 2; }
		request POST "/api/v1/load-balancing" \
			"Requested load-balancing algorithm $2" \
			"{\"algorithm\":\"$2\"}"
		;;
	-h|--help)
		usage
		;;
	*)
		usage >&2
		exit 2
		;;
esac
