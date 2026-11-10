#!/usr/bin/env bash
set -euo pipefail

CONTROL_PLANE_URL="${CONTROL_PLANE_URL:-http://localhost:5000}"
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

usage() {
	cat <<'EOF'
Usage: sysctl <command> [arguments]

Commands:
  status
  events [limit]
  scale <deployment> <replicas>
  self-heal [deployment]
  load-test --rps <requests-per-second> --duration <seconds> [--word <word>]
  load-tests
  load-test-status <test-id>
EOF
}

request() {
	local method="$1"
	local path="$2"
	local action="$3"
	local body="${4:-}"
	local formatter="${5:-simple}"
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
	if [[ -n "$response_body" ]]; then
		if command -v python3 >/dev/null 2>&1; then
			printf '%s' "$response_body" | python3 "$SCRIPT_DIR/format.py" "$formatter"
		elif command -v python >/dev/null 2>&1; then
			printf '%s' "$response_body" | python "$SCRIPT_DIR/format.py" "$formatter"
		else
			printf '%s\n' "$response_body"
			printf 'NOTE: Install Python 3 for formatted CLI summaries.\n' >&2
		fi
	fi
}

command="${1:-}"
case "$command" in
	status)
		[[ "$#" -eq 1 ]] || { usage >&2; exit 2; }
		request GET "/api/v1/status" "Retrieved cluster status" "" "status"
		;;
	events)
		[[ "$#" -le 2 ]] || { usage >&2; exit 2; }
		limit="${2:-20}"
		request GET "/api/v1/events?limit=${limit}" "Retrieved recent events" "" "events"
		;;
	scale)
		[[ "$#" -eq 3 ]] || { usage >&2; exit 2; }
		request POST "/api/v1/scale" \
			"Scaled deployment $2 to $3 replicas" \
			"{\"deployment\":\"$2\",\"replicas\":$3}" \
			"simple"
		;;
	self-heal)
		[[ "$#" -le 2 ]] || { usage >&2; exit 2; }
		deployment="${2:-wordle-api}"
		request POST "/api/v1/self-heal" \
			"Triggered self-healing for $deployment" \
			"{\"deployment\":\"$deployment\"}" \
			"simple"
		;;
	load-test)
		shift
		rps=""
		duration=""
		word="hello"
		while [[ "$#" -gt 0 ]]; do
			case "$1" in
				--rps)
					[[ "$#" -ge 2 ]] || { usage >&2; exit 2; }
					rps="$2"
					shift 2
					;;
				--duration)
					[[ "$#" -ge 2 ]] || { usage >&2; exit 2; }
					duration="$2"
					shift 2
					;;
				--word)
					[[ "$#" -ge 2 ]] || { usage >&2; exit 2; }
					word="$2"
					shift 2
					;;
				*)
					usage >&2
					exit 2
					;;
			esac
		done
		[[ "$rps" =~ ^[1-9][0-9]*$ && "$duration" =~ ^[1-9][0-9]*$ ]] || {
			printf 'ERROR: --rps and --duration must be positive integers.\n' >&2
			exit 2
		}
		[[ "$word" =~ ^[A-Za-z]+$ ]] || {
			printf 'ERROR: --word must contain letters only.\n' >&2
			exit 2
		}
		request POST "/api/v1/load-tests" \
			"Started load test at $rps RPS for $duration seconds" \
			"{\"rps\":$rps,\"duration\":$duration,\"word\":\"$word\"}" \
			"test"
		;;
	load-tests)
		[[ "$#" -eq 1 ]] || { usage >&2; exit 2; }
		request GET "/api/v1/load-tests" "Retrieved load-test history" "" "tests"
		;;
	load-test-status)
		[[ "$#" -eq 2 ]] || { usage >&2; exit 2; }
		request GET "/api/v1/load-tests/$2" "Retrieved load-test $2" "" "test"
		;;
	-h|--help)
		usage
		;;
	*)
		usage >&2
		exit 2
		;;
esac
