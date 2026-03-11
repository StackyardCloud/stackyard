#!/usr/bin/env python3
"""
Helpers for splitting AWS coverage workloads into deterministic shards.
"""

from __future__ import annotations

import fnmatch
import importlib.util
import re
import sys
from functools import lru_cache
from pathlib import Path
from typing import Iterable


SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parent
SERVER_DIR = REPO_ROOT / "internal" / "server"
AWSCLI_COVERAGE_SCRIPT = SCRIPT_DIR / "awscli-endpoint-coverage.py"

OPS_NAME_RE = re.compile(r'\{[^}]*\bName:\s*"([^"]+)"')


@lru_cache(maxsize=1)
def _load_awscli_module():
    spec = importlib.util.spec_from_file_location("awscli_endpoint_coverage", AWSCLI_COVERAGE_SCRIPT)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"unable to load {AWSCLI_COVERAGE_SCRIPT}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def discover_services() -> list[str]:
    module = _load_awscli_module()
    return sorted(str(service) for service in module.SERVICE_CONFIG)


def split_service_selector(raw: str) -> list[str]:
    module = _load_awscli_module()
    tokens: list[str] = []
    for token in raw.split(","):
        cleaned = token.strip()
        if not cleaned:
            continue
        if any(ch in cleaned for ch in "*?["):
            tokens.append(cleaned.lower())
        else:
            tokens.append(module.normalize_service_name(cleaned))
    return tokens or ["*"]


def filter_services(services: Iterable[str], selector: str) -> list[str]:
    patterns = split_service_selector(selector)
    return sorted(service for service in services if any(fnmatch.fnmatch(service, pattern) for pattern in patterns))


def validate_shard_args(shard_count: int, shard_index: int) -> None:
    if shard_count < 1:
        raise ValueError("--shard-count must be at least 1")
    if shard_index < 0 or shard_index >= shard_count:
        raise ValueError("--shard-index must be between 0 and shard-count - 1")


def parse_ops_file(path: Path) -> list[str]:
    src = path.read_text(encoding="utf-8")
    names = OPS_NAME_RE.findall(src)
    deduped: list[str] = []
    seen: set[str] = set()
    for name in names:
        if name in seen:
            continue
        seen.add(name)
        deduped.append(name)
    return deduped


def aws_endpoint_service_weights(services: Iterable[str]) -> dict[str, int]:
    weights: dict[str, int] = {}
    for service in services:
        ops_file = SERVER_DIR / f"{service}_ops.go"
        if ops_file.exists():
            weights[service] = max(len(parse_ops_file(ops_file)), 1)
        else:
            weights[service] = 1
    return weights


def _count_lines(path: Path) -> int:
    try:
        with path.open(encoding="utf-8") as handle:
            return sum(1 for _ in handle)
    except OSError:
        return 0


def aws_doc_service_weights(services: Iterable[str]) -> dict[str, int]:
    weights: dict[str, int] = {}
    for service in services:
        total = 0
        for path in sorted(SERVER_DIR.glob(f"{service}*.go")):
            if path.name.endswith("_ops.go"):
                continue
            total += _count_lines(path)
        weights[service] = max(total, 1)
    return weights


def assign_balanced_shards(services: Iterable[str], weights: dict[str, int], shard_count: int) -> list[list[str]]:
    validate_shard_args(shard_count, 0)
    groups = [[] for _ in range(shard_count)]
    totals = [0] * shard_count
    ordered = sorted(services, key=lambda service: (-weights.get(service, 1), service))
    for service in ordered:
        shard = min(range(shard_count), key=lambda index: (totals[index], index))
        groups[shard].append(service)
        totals[shard] += weights.get(service, 1)
    for group in groups:
        group.sort()
    return groups


def select_shard(services: Iterable[str], weights: dict[str, int], shard_count: int, shard_index: int) -> list[str]:
    validate_shard_args(shard_count, shard_index)
    return assign_balanced_shards(services, weights, shard_count)[shard_index]
