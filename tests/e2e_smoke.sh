#!/usr/bin/env bash
# E2E smoke test for the unified Prompt Injectulator server.
set -euo pipefail
PORT=8092
BASE="http://localhost:$PORT"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export JWT_SECRET
JWT_SECRET=$(openssl rand -base64 48)
export DEV_MODE=true
export DEV_TEST_EMAIL=dev@localhost
export DEV_TEST_PASSWORD='TestPass!2026x'
export PORT=$PORT

cd "$REPO_ROOT/backend" && go build -o /tmp/injectulator-server ./cmd/server
# Kill any stale server from a previous run before starting fresh.
pkill -f 'injectulator-server' 2>/dev/null || true
sleep 1
/tmp/injectulator-server > /tmp/injectulator-server.log 2>&1 &
SRV=$!
trap 'kill $SRV 2>/dev/null || true' EXIT
sleep 2

pass=0; fail=0
check() { # name, command exit code (0 = pass)
  if [ "$2" = "0" ]; then echo "PASS: $1"; pass=$((pass+1)); else echo "FAIL: $1"; fail=$((fail+1)); fi
}

# 1. register a regular user
REG=$(curl -s -X POST $BASE/api/auth/register -H 'Content-Type: application/json' -d '{"email":"user2@test.local","password":"RegUser123!"}')
echo "$REG" | grep -q '"role":"user"'; check "register regular user (role=user)" $?

UT=$(echo "$REG" | sed 's/.*"token": *"//;s/".*//')

# 2. regular user mock -> 403
CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST $BASE/api/llm/complete -H "Authorization: Bearer $UT" -H 'Content-Type: application/json' -d '{"provider":"mock","prompt":"hi"}')
[ "$CODE" = "403" ]; check "mock provider denied for regular user (403)" $?

# 3. regular user admin status -> 403
CODE=$(curl -s -o /dev/null -w '%{http_code}' $BASE/api/admin/status -H "Authorization: Bearer $UT")
[ "$CODE" = "403" ]; check "admin status denied for regular user (403)" $?

# 4. regular user assessment works
ASMT=$(curl -s -X POST $BASE/api/assessments -H "Authorization: Bearer $UT" -H 'Content-Type: application/json' -d '{"objective":"Check agent tool-call confirmation gates.","surface":"agent","riskFocus":"unsafe-tool-actions"}')
echo "$ASMT" | grep -q '"scenarios"'; check "regular user can generate assessment" $?

# 5. SPA served at /
curl -s $BASE/ | grep -q 'assets/'; check "SPA index served" $?

# 6. no-token request -> 401
CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST $BASE/api/assessments -H 'Content-Type: application/json' -d '{}')
[ "$CODE" = "401" ]; check "unauthenticated assessment denied (401)" $?

# 7. admin login + admin status
AT=$(curl -s -X POST $BASE/api/auth/login -H 'Content-Type: application/json' -d '{"email":"dev@localhost","password":"TestPass!2026x"}' | sed 's/.*"token": *"//;s/".*//')
ADMIN=$(curl -s $BASE/api/admin/status -H "Authorization: Bearer $AT")
echo "$ADMIN" | grep -q '"mock_enabled":true'; check "admin status shows mock enabled" $?

# 8. admin mock complete works
MOCK=$(curl -s -X POST $BASE/api/llm/complete -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' -d '{"provider":"mock","prompt":"inject test"}')
echo "$MOCK" | grep -q '"mode":"mock (no cost)"'; check "admin mock completion (zero cost)" $?

# 9. admin reset usage
RESET=$(curl -s -X POST $BASE/api/admin/reset-usage -H "Authorization: Bearer $AT" -H 'Content-Type: application/json' -d '{"email":"user2@test.local"}')
echo "$RESET" | grep -q 'usage reset'; check "admin budget reset" $?

# 10. security headers present
HDRS=$(curl -s -D - -o /dev/null $BASE/)
echo "$HDRS" | grep -qi 'X-Content-Type-Options: nosniff'; check "security header: nosniff" $?
echo "$HDRS" | grep -qi 'X-Frame-Options'; check "security header: X-Frame-Options" $?
echo "$HDRS" | grep -qi 'Content-Security-Policy'; check "security header: CSP" $?

echo
echo "RESULT: $pass passed, $fail failed"
if [ "$fail" != "0" ]; then exit 1; fi
exit 0