#!/usr/bin/env python3
"""
Check that every example under examples/ can be built by Docker.

Rules:
1) every docker-compose `context:` resolves to the repo root,
2) every docker-compose `dockerfile:` exists,
3) every Dockerfile `go build ./examples/...` package directory exists,
4) every `FROM golang:X.Y` image satisfies the go.mod `go` directive.

Run with --fix to rewrite violations in place. Fixes are computed from each
file's location: contexts become the relative path to the repo root, and stale
Dockerfile or package paths point at the example's own directory.
"""

from __future__ import annotations

import argparse
import os
import re
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
EXAMPLES_DIR = REPO_ROOT / "examples"

CONTEXT_RE = re.compile(r"^(\s*context:\s*)(\S+)(\s*)$", re.M)
DOCKERFILE_RE = re.compile(r"^(\s*dockerfile:\s*)(\S+)(\s*)$", re.M)
GO_BUILD_RE = re.compile(r"(go build[^\n]*?)(\./examples/\S+)")
GOLANG_FROM_RE = re.compile(r"^(FROM\s+golang:)(\d+)\.(\d+)", re.M)


def go_mod_version() -> tuple[int, int]:
    match = re.search(r"^go (\d+)\.(\d+)", (REPO_ROOT / "go.mod").read_text(), re.M)
    if not match:
        sys.exit("go.mod has no go directive")
    return int(match.group(1)), int(match.group(2))


def rel(path: Path) -> str:
    return str(path.relative_to(REPO_ROOT))


def check_compose(path: Path, fix: bool, problems: list[str]) -> str:
    text = path.read_text()
    want_context = os.path.relpath(REPO_ROOT, path.parent)

    def fix_context(m: re.Match[str]) -> str:
        if (path.parent / m.group(2)).resolve() == REPO_ROOT:
            return m.group(0)
        problems.append(f"{rel(path)}: context {m.group(2)!r} is not the repo root (want {want_context!r})")
        return m.group(1) + want_context + m.group(3)

    def fix_dockerfile(m: re.Match[str]) -> str:
        target = m.group(2)
        # A bare "Dockerfile" is resolved against the context (the repo root Dockerfile).
        if (REPO_ROOT / target).is_file():
            return m.group(0)
        local = path.parent / "Dockerfile"
        problems.append(f"{rel(path)}: dockerfile {target!r} does not exist")
        return m.group(1) + rel(local) + m.group(3) if local.is_file() else m.group(0)

    new = DOCKERFILE_RE.sub(fix_dockerfile, CONTEXT_RE.sub(fix_context, text))
    if fix and new != text:
        path.write_text(new)
    return new


def check_dockerfile(path: Path, min_go: tuple[int, int], fix: bool, problems: list[str]) -> None:
    text = path.read_text()
    here = "./" + rel(path.parent)

    def fix_build(m: re.Match[str]) -> str:
        pkg = m.group(2)
        if (REPO_ROOT / pkg).is_dir():
            return m.group(0)
        problems.append(f"{rel(path)}: go build path {pkg!r} does not exist")
        return m.group(1) + here if (path.parent / "main.go").is_file() else m.group(0)

    def fix_golang(m: re.Match[str]) -> str:
        if (int(m.group(2)), int(m.group(3))) >= min_go:
            return m.group(0)
        problems.append(f"{rel(path)}: golang:{m.group(2)}.{m.group(3)} is older than go.mod's {min_go[0]}.{min_go[1]}")
        return f"{m.group(1)}{min_go[0]}.{min_go[1]}"

    new = GOLANG_FROM_RE.sub(fix_golang, GO_BUILD_RE.sub(fix_build, text))
    if fix and new != text:
        path.write_text(new)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("--fix", action="store_true", help="Rewrite violations in place.")
    args = parser.parse_args()

    min_go = go_mod_version()
    problems: list[str] = []
    composes = sorted(EXAMPLES_DIR.rglob("docker-compose.yml"))
    dockerfiles = sorted(EXAMPLES_DIR.rglob("Dockerfile"))
    for path in composes:
        check_compose(path, args.fix, problems)
    for path in dockerfiles:
        check_dockerfile(path, min_go, args.fix, problems)

    for line in problems:
        print(line)
    verb = "fixed" if args.fix else "found"
    print(f"{len(composes)} compose files, {len(dockerfiles)} Dockerfiles: {len(problems)} problems {verb}")
    return 0 if args.fix or not problems else 1


if __name__ == "__main__":
    sys.exit(main())
