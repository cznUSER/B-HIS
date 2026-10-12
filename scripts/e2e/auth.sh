#!/usr/bin/env bash
# Hits a running memory-mode API. Does not read or print a database DSN.
set -euo pipefail
BASE="${BASE:-http://127.0.0.1:8080}"

login() {
  curl -sS -D /tmp/bhis-login.hdr "$BASE/api/v1/auth/login" \
    -H 'Content-Type: application/json' \
    -d '{"username":"admin","password":"Admin@123"}'
  echo
}

echo "== login =="
login
echo "== bad password (expect INVALID_CREDENTIALS) =="
curl -sS -D - "$BASE/api/v1/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"wrong"}'
echo
echo "== me without token (expect UNAUTHORIZED) =="
curl -sS -D - "$BASE/api/v1/auth/me"
echo
echo "Set TOKEN from accessToken, then:"
echo "  curl -sS $BASE/api/v1/auth/me -H \"Authorization: Bearer \$TOKEN\""
echo "  curl -sS -D - -o /dev/null -X POST $BASE/api/v1/auth/logout -H \"Authorization: Bearer \$TOKEN\""
