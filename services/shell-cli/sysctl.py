#!/usr/bin/env python3
import argparse
import json
import os
import sys
import urllib.error
import urllib.request

from format import print_events, print_simple, print_status, print_test, print_tests


DEFAULT_URL = os.environ.get("CONTROL_PLANE_URL", "http://localhost:8080")
FORMATTERS = {
    "status": print_status,
    "events": print_events,
    "test": print_test,
    "tests": print_tests,
}


def request(method: str, path: str, action: str, formatter: str = "simple", payload: dict | None = None) -> None:
    body = None if payload is None else json.dumps(payload).encode()
    headers = {"Content-Type": "application/json"} if body else {}
    request = urllib.request.Request(
        f"{DEFAULT_URL.rstrip('/')}{path}", data=body, headers=headers, method=method
    )
    try:
        with urllib.request.urlopen(request) as response:
            data = json.load(response)
            print(f"SUCCESS: {action}.")
            FORMATTERS.get(formatter, print_simple)(data)
    except urllib.error.HTTPError as error:
        detail = error.read().decode()
        print(f"ERROR: {action} failed (HTTP {error.code}).", file=sys.stderr)
        if detail:
            print(detail, file=sys.stderr)
        raise SystemExit(1) from error
    except urllib.error.URLError as error:
        print(f"ERROR: Could not reach the control plane at {DEFAULT_URL}.", file=sys.stderr)
        print("       Start the port-forward and check CONTROL_PLANE_URL, then retry.", file=sys.stderr)
        raise SystemExit(1) from error


def main() -> None:
    parser = argparse.ArgumentParser(prog="sysctl")
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("status")
    events = subparsers.add_parser("events")
    events.add_argument("limit", nargs="?", type=int, default=20)

    scale = subparsers.add_parser("scale")
    scale.add_argument("deployment")
    scale.add_argument("replicas", type=int)

    heal = subparsers.add_parser("self-heal")
    heal.add_argument("deployment", nargs="?", default="wordle-api")

    load_test = subparsers.add_parser("load-test")
    load_test.add_argument("--rps", type=int, required=True)
    load_test.add_argument("--duration", type=int, required=True)
    load_test.add_argument("--word", default="hello")

    subparsers.add_parser("load-tests")
    load_test_status = subparsers.add_parser("load-test-status")
    load_test_status.add_argument("test_id")

    args = parser.parse_args()
    if args.command == "status":
        request("GET", "/api/v1/status", "Retrieved cluster status", "status")
    elif args.command == "events":
        request("GET", f"/api/v1/events?limit={args.limit}", "Retrieved recent events", "events")
    elif args.command == "scale":
        request(
            "POST",
            "/api/v1/scale",
            f"Scaled deployment {args.deployment} to {args.replicas} replicas",
            payload={"deployment": args.deployment, "replicas": args.replicas},
        )
    elif args.command == "self-heal":
        request(
            "POST",
            "/api/v1/self-heal",
            f"Triggered self-healing for {args.deployment}",
            payload={"deployment": args.deployment},
        )
    elif args.command == "load-test":
        if not 1 <= args.rps <= 1000 or not 1 <= args.duration <= 300:
            parser.error("--rps must be 1-1000 and --duration must be 1-300 seconds")
        if not args.word.isalpha():
            parser.error("--word must contain letters only")
        request(
            "POST",
            "/api/v1/load-tests",
            f"Started load test at {args.rps} RPS for {args.duration} seconds",
            "test",
            {"rps": args.rps, "duration": args.duration, "word": args.word},
        )
    elif args.command == "load-tests":
        request("GET", "/api/v1/load-tests", "Retrieved load-test history", "tests")
    elif args.command == "load-test-status":
        request("GET", f"/api/v1/load-tests/{args.test_id}", f"Retrieved load-test {args.test_id}", "test")

if __name__ == "__main__":
    main()
