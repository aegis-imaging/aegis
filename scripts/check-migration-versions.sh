#!/usr/bin/env bash
set -euo pipefail

# Fails fast on duplicate goose migration version numbers under
# api/migrate/migrations. Two files claiming the same NNN_ prefix break
# goose's ordering at startup; the API container panics before binding
# port 8080 and Cloud Run rejects the deploy without ever serving a
# request. We've hit this collision multiple times when parallel PRs
# both grab "the next available version" — this guard catches it at
# PR time instead of in deploy logs.
#
# Run locally: ./scripts/check-migration-versions.sh
# CI: invoked from .github/workflows/ci.yml

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="$ROOT_DIR/api/migrate/migrations"
export MIGRATIONS_DIR

python3 - <<'PY'
import os
import re
import sys
from collections import defaultdict
from pathlib import Path

migrations_dir = Path(os.environ["MIGRATIONS_DIR"])
if not migrations_dir.is_dir():
    print(f"[migration-version-guard] missing directory: {migrations_dir}", file=sys.stderr)
    sys.exit(2)

# Migration filenames look like 087_create_comment_edits.sql. Anything
# under the directory that doesn't match the pattern is reported
# separately so a typo doesn't slip in silently.
pattern = re.compile(r"^(\d{3,})_[A-Za-z0-9_]+\.sql$")

by_version: dict[str, list[str]] = defaultdict(list)
unrecognized: list[str] = []

for path in sorted(migrations_dir.iterdir()):
    if not path.is_file():
        continue
    if path.suffix != ".sql":
        unrecognized.append(path.name)
        continue
    m = pattern.match(path.name)
    if not m:
        unrecognized.append(path.name)
        continue
    by_version[m.group(1)].append(path.name)

duplicates = {v: names for v, names in by_version.items() if len(names) > 1}

if duplicates or unrecognized:
    print("[migration-version-guard] migration filename problems detected")
    if duplicates:
        print("\nDuplicate version numbers — goose will panic on startup:")
        for version, names in sorted(duplicates.items()):
            print(f"  version {version}:")
            for n in names:
                print(f"    - {n}")
        print("\nRename one of each pair to the next free version (the highest")
        print("currently in use + 1) and update its commit accordingly.")
    if unrecognized:
        print("\nFiles that don't match the NNN_lower_snake_case.sql pattern:")
        for n in unrecognized:
            print(f"  - {n}")
        print("\nRename to a numeric NNN_*.sql or remove if not a migration.")
    sys.exit(1)

print(f"[migration-version-guard] OK: {sum(len(v) for v in by_version.values())} migrations across {len(by_version)} unique versions.")
PY
