#!/usr/bin/env python3
import json
import sys
from datetime import datetime


def value(data, key, default="--"):
    current = data.get(key, default)
    return default if current is None or current == "" else current


def timestamp(value_to_format):
    if not value_to_format or value_to_format == "--":
        return "--"
    try:
        return datetime.fromisoformat(value_to_format.replace("Z", "+00:00")).strftime("%Y-%m-%d %H:%M:%S")
    except ValueError:
        return str(value_to_format)


def print_status(data):
    print("Cluster status")
    for deployment in data.get("deployments", []):
        print(
            f"  {deployment['name']:<16} "
            f"ready {deployment.get('ready_replicas', 0)}/"
            f"{deployment.get('desired_replicas', 0)}  "
            f"available {deployment.get('available_replicas', 0)}"
        )
    pods = data.get("wordle_api_pods", [])
    print(f"\nWordle API pods ({len(pods)})")
    for pod in pods:
        print(f"  {pod.get('phase', '--'):<10} {pod.get('name', '--')}")


def print_events(data):
    events = data if isinstance(data, list) else data.get("events", [])
    if not events:
        print("No events recorded.")
        return
    for event in events:
        print(f"{timestamp(event.get('timestamp'))}  [{event.get('type', '--')}] {event.get('message', '--')}")


def test_line(test):
    return (
        f"{test.get('id', '--'):<25} "
        f"{test.get('status', '--'):<10} "
        f"{test.get('requested_rps', '--'):>5} RPS  "
        f"{test.get('duration_seconds', '--'):>4}s  "
        f"requests {test.get('requests_sent', 0):>6}  "
        f"ok {test.get('successes', 0):>6}  "
        f"errors {test.get('errors', 0):>6}  "
        f"p50 {test.get('p50_latency_ms', 0):>7} ms  "
        f"p95 {test.get('p95_latency_ms', 0):>7} ms"
    )


def print_tests(data):
    tests = data if isinstance(data, list) else data.get("tests", [])
    if not tests:
        print("No load tests recorded.")
        return
    print("Load tests")
    print("  ID                      STATUS       RPS  TIME  requests       ok  errors      p50 ms      p95 ms")
    for test in tests:
        print(f"  {test_line(test)}")


def print_test(data):
    total = data.get("successes", 0) + data.get("errors", 0)
    success_rate = (data.get("successes", 0) / total * 100) if total else 0
    print(f"Load test {value(data, 'id')}")
    print(f"  Status:       {value(data, 'status')}")
    print(f"  Started:      {timestamp(data.get('started_at'))}")
    print(f"  Finished:     {timestamp(data.get('finished_at'))}")
    print(f"  Target:       {value(data, 'requested_rps')} RPS for {value(data, 'duration_seconds')}s")
    print(f"  Requests:     {value(data, 'requests_sent')}")
    print(f"  Successes:    {value(data, 'successes')}")
    print(f"  Errors:       {value(data, 'errors')}")
    print(f"  Success rate: {success_rate:.1f}%")
    print(f"  Min latency:  {value(data, 'min_latency_ms')} ms")
    print(f"  Average:      {value(data, 'average_latency_ms')} ms")
    print(f"  p50:          {value(data, 'p50_latency_ms')} ms")
    print(f"  p95:          {value(data, 'p95_latency_ms')} ms")
    statuses = data.get("status_codes", {})
    if statuses:
        print("  Status codes: " + ", ".join(f"{code}={count}" for code, count in sorted(statuses.items())))


def print_simple(data):
    if isinstance(data, dict):
        for key, item in data.items():
            print(f"  {key.replace('_', ' ').capitalize()}: {item}")
    else:
        print(data)


def main():
    if len(sys.argv) != 2:
        raise SystemExit("formatter requires a response type")
    mode = sys.argv[1]
    data = json.load(sys.stdin)
    if mode == "status":
        print_status(data)
    elif mode == "events":
        print_events(data)
    elif mode == "tests":
        print_tests(data)
    elif mode == "test":
        print_test(data)
    else:
        print_simple(data)


if __name__ == "__main__":
    main()
