#!/usr/bin/env bash
set -euo pipefail

echo "=== cvox local test ==="
echo ""

# Step 1: Build
echo "Building..."
npm run build --silent
echo "Build OK"
echo ""

# Step 2: Test Notification event
echo "--- Test: Notification event ---"
if echo '{"hook_event_name":"Notification"}' | node dist/index.js notify; then
  echo "PASS"
else
  echo "FAIL (exit code $?)"
fi
echo ""

# Step 3: Test Stop event
echo "--- Test: Stop event ---"
if echo '{"hook_event_name":"Stop"}' | node dist/index.js notify; then
  echo "PASS"
else
  echo "FAIL (exit code $?)"
fi
