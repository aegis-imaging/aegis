#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MAIN_GO="$ROOT_DIR/api/main.go"
MATRIX_MD="$ROOT_DIR/docs/planning/access-matrix-project-site-scoping.md"
export MAIN_GO MATRIX_MD

python3 - <<'PY'
import os
import re
import sys
from pathlib import Path

main_go = Path(os.environ["MAIN_GO"])
matrix_md = Path(os.environ["MATRIX_MD"])

main_text = main_go.read_text(encoding="utf-8")
matrix_text = matrix_md.read_text(encoding="utf-8")

route_re = re.compile(r'mux\.Handle(?:Func)?\("([^"]+)",\s*(auth|adminOnly)\(')
matrix_re = re.compile(r'^\|\s*`(auth|adminOnly)`\s*\|\s*`([^`]+)`\s*\|', re.MULTILINE)

main_routes = {(wrapper, route) for route, wrapper in route_re.findall(main_text)}
matrix_routes = set(matrix_re.findall(matrix_text))

missing = sorted(main_routes - matrix_routes)
extra = sorted(matrix_routes - main_routes)

if missing or extra:
    print("[access-matrix-guard] access matrix mismatch detected")
    if missing:
        print("\nRoutes in api/main.go missing from access matrix:")
        for wrapper, route in missing:
            print(f"  - {wrapper}: {route}")
    if extra:
        print("\nRoutes present in access matrix but not in api/main.go:")
        for wrapper, route in extra:
            print(f"  - {wrapper}: {route}")
    print("\nUpdate docs/planning/access-matrix-project-site-scoping.md to match authenticated route inventory.")
    sys.exit(1)

print(f"[access-matrix-guard] OK: {len(main_routes)} authenticated routes mapped.")
PY
