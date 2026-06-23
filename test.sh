#!/usr/bin/env bash
set -euo pipefail

echo "=== cvox local test ==="
echo ""

# Step 1: Build
echo "Building..."
go build -o cvox .
echo "Build OK"
echo ""

# Step 2: Test PermissionRequest event
echo "--- Test: PermissionRequest event ---"
if echo '{"hook_event_name":"PermissionRequest"}' | ./cvox notify; then
  echo "PASS"
else
  echo "FAIL (exit code $?)"
fi
echo ""

# Step 3: Test Stop event
echo "--- Test: Stop event ---"
if echo '{"hook_event_name":"Stop"}' | ./cvox notify; then
  echo "PASS"
else
  echo "FAIL (exit code $?)"
fi
