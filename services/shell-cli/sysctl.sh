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
	local body="${3:-}"
	local url="${CONTROL_PLANE_URL%/}${path}"

	if [[ -n "$body" ]]; then
		curl --fail-with-body --silent --show-error \
			--request "$method" \
			--header "Content-Type: application/json" \
			--data "$body" "$url"
	else
		curl --fail-with-body --silent --show-error \
			--request "$method" "$url"
	fi
	printf '\n'
}

command="${1:-}"
case "$command" in
	status)
		[[ "$#" -eq 1 ]] || { usage >&2; exit 2; }
		request GET "/api/v1/status"
		;;
	events)
		[[ "$#" -le 2 ]] || { usage >&2; exit 2; }
		limit="${2:-20}"
		request GET "/api/v1/events?limit=${limit}"
		;;
	scale)
		[[ "$#" -eq 3 ]] || { usage >&2; exit 2; }
		request POST "/api/v1/scale" \
			"{\"deployment\":\"$2\",\"replicas\":$3}"
		;;
	self-heal)
		[[ "$#" -le 2 ]] || { usage >&2; exit 2; }
		deployment="${2:-wordle-api}"
		request POST "/api/v1/self-heal" \
			"{\"deployment\":\"$deployment\"}"
		;;
	load-balancing)
		[[ "$#" -eq 2 ]] || { usage >&2; exit 2; }
		request POST "/api/v1/load-balancing" \
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
