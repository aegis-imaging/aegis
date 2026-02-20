#!/usr/bin/env bash
set -euo pipefail

PATTERN='CHANGE_ME|password\s*=\s*"(?i:changeme)"'

if rg -n --pcre2 --glob 'terraform/**/*.tf' "${PATTERN}" terraform; then
  echo
  echo "error: insecure credential placeholder detected in Terraform code." >&2
  echo "Replace placeholder values with Secret Manager/Secrets Manager patterns." >&2
  exit 1
fi

echo "ok: no insecure Terraform credential placeholders found."
