#!/usr/bin/env bash
set -euo pipefail

# MergeOS Bounty Evidence Verifier
# Verifies submitted PRs against bounty requirements

echo "=== MergeOS Bounty Evidence Verifier ==="
echo ""

PR_NUM="${1:-}"
if [ -z "$PR_NUM" ]; then
  echo "Usage: $0 <PR-number>"
  echo "Example: $0 42"
  exit 1
fi

echo "Verifying PR #$PR_NUM..."
echo ""

PR_DATA=$(gh pr view "$PR_NUM" --json title,body,additions,deletions,files,state,mergeable 2>/dev/null || echo "{}")

if [ "$(echo "$PR_DATA" | jq -r '.state')" != "OPEN" ]; then
  echo "FAIL: PR #$PR_NUM is not open"
  exit 1
fi

echo "Title: $(echo "$PR_DATA" | jq -r '.title')"
echo "Files changed: $(echo "$PR_DATA" | jq -r '.additions') additions, $(echo "$PR_DATA" | jq -r '.deletions') deletions"
echo ""

# Check 1: PR body has content
BODY=$(echo "$PR_DATA" | jq -r '.body // ""')
if [ -z "$BODY" ] || [ ${#BODY} -lt 50 ]; then
  echo "WARN: PR body is too short or missing"
else
  echo "PASS: PR body is present (${#BODY} chars)"
fi

# Check 2: Evidence screenshots
FILES=$(echo "$PR_DATA" | jq -r '.files[] | .path // ""' 2>/dev/null || echo "")
EVIDENCE_FILES=$(echo "$FILES" | grep -i "evidence\|screenshot\|proof\|\.png\|\.jpg\|\.gif" || echo "")
if [ -n "$EVIDENCE_FILES" ]; then
  echo "PASS: Evidence screenshots attached:"
  echo "$EVIDENCE_FILES" | while read f; do echo "  - $f"; done
else
  echo "FAIL: No evidence screenshots found in PR"
fi

# Check 3: Tests exist
TEST_FILES=$(echo "$FILES" | grep -i "test\|spec\|*_test.go\|*.test.js" || echo "")
if [ -n "$TEST_FILES" ]; then
  echo "PASS: Test files included:"
  echo "$TEST_FILES" | while read f; do echo "  - $f"; done
else
  echo "WARN: No test files found in PR"
fi

# Check 4: Documentation update
DOC_FILES=$(echo "$FILES" | grep -i "readme\|\.md\|docs/" || echo "")
if [ -n "$DOC_FILES" ]; then
  echo "PASS: Documentation updated"
else
  echo "WARN: No documentation changes"
fi

# Check 5: Run tests
echo ""
echo "=== Running Tests ==="
if [ -f "backend/go.mod" ]; then
  echo "Backend tests:"
  (cd backend && go test ./... 2>&1 | tail -20) || echo "TEST_FAILED"
fi
if [ -f "frontend/package.json" ]; then
  echo "Frontend tests:"
  (cd frontend && npm test 2>&1 | tail -20) || echo "TEST_FAILED"
fi

echo ""
echo "=== Verification Complete ==="
