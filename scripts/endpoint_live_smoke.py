#!/usr/bin/env python3
from __future__ import annotations

import subprocess
import time
from dataclasses import dataclass, field
from typing import Callable, Mapping, Sequence
from urllib import error as urlerror
from urllib import parse as urlparse
from urllib import request as urlrequest


Validator = Callable[[int, Mapping[str, str], bytes], tuple[bool, str]]


@dataclass(frozen=True)
class HTTPCheck:
    service: str
    name: str
    method: str
    path: str
    headers: Mapping[str, str] = field(default_factory=dict)
    body: bytes | None = None
    validator: Validator | None = None


def run_subprocess(cmd: Sequence[str], env: dict[str, str]) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        cmd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        env=env,
    )


def is_local_endpoint(endpoint_url: str) -> bool:
    parsed = urlparse.urlparse(endpoint_url)
    host = (parsed.hostname or "").strip().lower()
    return host in {"localhost", "127.0.0.1", "::1"}


def run_compose_up(compose_file: str, service: str, rebuild: bool, env: dict[str, str]) -> tuple[bool, str]:
    up_args = ["up", "-d"]
    if rebuild:
        up_args.append("--build")
    up_args.append(service)

    candidates = [
        ["docker", "compose", "-f", compose_file, *up_args],
        ["docker-compose", "-f", compose_file, *up_args],
    ]
    errors: list[str] = []
    for cmd in candidates:
        cp = run_subprocess(cmd, env)
        if cp.returncode == 0:
            return True, " ".join(cmd)
        text = "\n".join(part for part in [cp.stdout.strip(), cp.stderr.strip()] if part).strip()
        errors.append(f"{' '.join(cmd)} -> {text}")
    return False, " | ".join(errors)


def run_compose_cleanup(compose_file: str, service: str, env: dict[str, str]) -> tuple[bool, str]:
    candidates = [
        ["docker", "compose", "-f", compose_file, "rm", "-s", "-f", service],
        ["docker-compose", "-f", compose_file, "rm", "-s", "-f", service],
        ["docker", "compose", "-f", compose_file, "down", "--remove-orphans"],
        ["docker-compose", "-f", compose_file, "down", "--remove-orphans"],
    ]
    errors: list[str] = []
    for cmd in candidates:
        cp = run_subprocess(cmd, env)
        if cp.returncode == 0:
            return True, " ".join(cmd)
        text = "\n".join(part for part in [cp.stdout.strip(), cp.stderr.strip()] if part).strip()
        errors.append(f"{' '.join(cmd)} -> {text}")
    return False, " | ".join(errors)


def wait_for_stackyard_health(health_url: str, attempts: int, delay_sec: float) -> bool:
    for _ in range(attempts):
        try:
            req = urlrequest.Request(health_url, method="GET")
            with urlrequest.urlopen(req, timeout=2.0) as resp:
                if 200 <= int(resp.status) < 300:
                    return True
        except (urlerror.URLError, TimeoutError, ValueError, ConnectionError, OSError):
            pass
        time.sleep(delay_sec)
    return False


def ensure_stackyard_up(
    endpoint_url: str,
    compose_file: str,
    service: str,
    rebuild: bool,
    health_path: str,
    attempts: int,
    delay_sec: float,
    env: dict[str, str],
) -> None:
    if not is_local_endpoint(endpoint_url):
        return

    ok, detail = run_compose_up(compose_file, service, rebuild, env)
    if not ok:
        raise RuntimeError(f"failed to start Stackyard container: {detail}")

    base = endpoint_url.rstrip("/")
    path = health_path if health_path.startswith("/") else f"/{health_path}"
    health_url = f"{base}{path}"
    if not wait_for_stackyard_health(health_url, attempts, delay_sec):
        raise RuntimeError(f"Stackyard health check failed at {health_url} after {attempts} attempts")


def execute_http_check(
    endpoint_url: str,
    check: HTTPCheck,
    timeout_sec: float,
) -> tuple[int | None, dict[str, str], bytes, str]:
    base = endpoint_url.rstrip("/")
    path = check.path if check.path.startswith("/") else f"/{check.path}"
    request_url = f"{base}{path}"
    req = urlrequest.Request(request_url, data=check.body, method=check.method)
    for key, value in check.headers.items():
        req.add_header(key, value)

    try:
        with urlrequest.urlopen(req, timeout=timeout_sec) as resp:
            return int(resp.status), _normalize_headers(resp.headers.items()), resp.read(), ""
    except urlerror.HTTPError as err:
        return int(err.code), _normalize_headers(err.headers.items()), err.read(), ""
    except (urlerror.URLError, TimeoutError, ValueError, ConnectionError, OSError) as err:
        return None, {}, b"", str(err)


def run_live_smoke_suite(
    *,
    provider: str,
    endpoint_url: str,
    checks: Sequence[HTTPCheck],
    start_stackyard: bool,
    rebuild_stackyard: bool,
    compose_file: str,
    stackyard_service: str,
    health_path: str,
    health_attempts: int,
    health_delay_sec: float,
    timeout_sec: float,
    env: dict[str, str],
) -> tuple[dict[str, object], bool]:
    selected_services = sorted({check.service for check in checks})
    payload: dict[str, object] = {
        "provider": provider,
        "enabled": True,
        "endpoint_url": endpoint_url,
        "selected_services": selected_services,
        "selected_checks": len(checks),
        "passed_checks": 0,
        "failed_checks": 0,
        "strict": True,
        "skipped": len(checks) == 0,
        "startup_error": "",
        "checks": [],
    }
    cleanup_required = bool(checks) and start_stackyard and is_local_endpoint(endpoint_url)
    if not checks:
        return payload, False

    if start_stackyard:
        try:
            ensure_stackyard_up(
                endpoint_url=endpoint_url,
                compose_file=compose_file,
                service=stackyard_service,
                rebuild=rebuild_stackyard,
                health_path=health_path,
                attempts=health_attempts,
                delay_sec=health_delay_sec,
                env=env,
            )
        except RuntimeError as err:
            payload["failed_checks"] = len(checks)
            payload["strict"] = False
            payload["startup_error"] = str(err)
            return payload, cleanup_required

    results: list[dict[str, object]] = []
    passed = 0
    failed = 0
    for check in checks:
        http_status, headers, body, transport_error = execute_http_check(endpoint_url, check, timeout_sec)
        if transport_error:
            ok = False
            detail = transport_error
        else:
            if check.validator is None:
                ok = http_status is not None and 200 <= http_status < 300
                detail = f"HTTP {http_status}" if http_status is not None else "missing HTTP status"
            else:
                ok, detail = check.validator(http_status or 0, headers, body)
        if ok:
            passed += 1
        else:
            failed += 1
        results.append(
            {
                "service": check.service,
                "name": check.name,
                "method": check.method,
                "path": check.path,
                "http_status": http_status,
                "ok": ok,
                "detail": detail,
                "transport_error": transport_error,
            }
        )

    payload["checks"] = results
    payload["passed_checks"] = passed
    payload["failed_checks"] = failed
    payload["strict"] = failed == 0
    return payload, cleanup_required


def _normalize_headers(items: Sequence[tuple[str, str]]) -> dict[str, str]:
    out: dict[str, str] = {}
    for key, value in items:
        out[str(key).strip().lower()] = str(value).strip()
    return out
