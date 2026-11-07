#!/usr/bin/env python3
import argparse
import json
import os
import sys
import urllib.error
import urllib.request


DEFAULT_URL = os.environ.get("CONTROL_PLANE_URL", "http://localhost:5000")


def request(method: str, path: str, payload: dict | None = None) -> None:
    body = None if payload is None else json.dumps(payload).encode()
    headers = {"Content-Type": "application/json"} if body else {}
    request = urllib.request.Request(
        f"{DEFAULT_URL.rstrip('/')}{path}", data=body, headers=headers, method=method
    )
    try:
        with urllib.request.urlopen(request) as response:
            print(json.dumps(json.load(response), indent=2))
    except urllib.error.HTTPError as error:
        detail = error.read().decode()
        print(f"control plane error ({error.code}): {detail}", file=sys.stderr)
        raise SystemExit(1) from error
    except urllib.error.URLError as error:
        print(f"cannot reach control plane at {DEFAULT_URL}: {error.reason}", file=sys.stderr)
        raise SystemExit(1) from error


def main() -> None:
    parser = argparse.ArgumentParser(prog="sysctl")
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("status")
    events = subparsers.add_parser("events")
    events.add_argument("--limit", type=int, default=20)

    scale = subparsers.add_parser("scale")
    scale.add_argument("deployment")
    scale.add_argument("replicas", type=int)

    heal = subparsers.add_parser("self-heal")
    heal.add_argument("deployment", nargs="?", default="wordle-api")

    balancing = subparsers.add_parser("load-balancing")
    balancing.add_argument("algorithm")

    args = parser.parse_args()
    if args.command == "status":
        request("GET", "/api/v1/status")
    elif args.command == "events":
        request("GET", f"/api/v1/events?limit={args.limit}")
    elif args.command == "scale":
        request("POST", "/api/v1/scale", {"deployment": args.deployment, "replicas": args.replicas})
    elif args.command == "self-heal":
        request("POST", "/api/v1/self-heal", {"deployment": args.deployment})
    elif args.command == "load-balancing":
        request("POST", "/api/v1/load-balancing", {"algorithm": args.algorithm})


if __name__ == "__main__":
    main()
