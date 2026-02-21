#!/usr/bin/env bash
set -euo pipefail

# Patterns that indicate a hardcoded or placeholder credential in a .tf file.
# Any match is a CI failure — use Secret Manager / auto-generated passwords instead.
WEAK_TF_PATTERN='CHANGE_ME|password\s*=\s*"(?i:changeme|aegis|postgres|password|admin|secret|letmein|root|12345|qwerty|test)"'

FAILED=0

echo "Checking terraform/**/*.tf for insecure credential patterns..."
if rg -n --pcre2 --glob 'terraform/**/*.tf' "${WEAK_TF_PATTERN}" terraform 2>/dev/null; then
  echo
  echo "error: insecure credential placeholder detected in Terraform .tf files." >&2
  echo "       Leave db_password empty to auto-generate, or source from Secret Manager." >&2
  FAILED=1
fi

# Verify no non-example .tfvars files are tracked in git (they must stay gitignored).
echo "Checking that .tfvars files are not tracked in git..."
TRACKED_TFVARS=$(git ls-files -- 'terraform/*.tfvars' 'terraform/**/*.tfvars' 2>/dev/null | grep -v '\.example$' || true)
if [ -n "${TRACKED_TFVARS}" ]; then
  echo
  echo "error: the following .tfvars files are tracked in git and must be gitignored:" >&2
  echo "${TRACKED_TFVARS}" >&2
  echo "       Add them to .gitignore — they may contain real credentials." >&2
  FAILED=1
fi

if [ "${FAILED}" -ne 0 ]; then
  exit 1
fi

echo "ok: no insecure Terraform credential placeholders found."
