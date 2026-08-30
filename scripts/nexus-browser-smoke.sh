#!/usr/bin/env bash
set -u

# Browser smoke for an owned local Nexus instance. It intentionally accepts a
# bootstrap URL through stdin/env and never prints the URL (the URL contains a
# one-time authentication token).
bootstrap=${NEXUS_BOOTSTRAP_URL:-}
artifact_dir=${NEXUS_E2E_ARTIFACT_DIR:-.e2e-artifacts}

if [[ -z "$bootstrap" ]]; then
  echo "NEXUS_BROWSER_SMOKE_SKIP: NEXUS_BOOTSTRAP_URL is required" >&2
  exit 2
fi
mkdir -p "$artifact_dir"

if ! command -v npx >/dev/null 2>&1; then
  echo "NEXUS_BROWSER_SMOKE_SKIP: npx is unavailable" >&2
  exit 2
fi

# Playwright's CLI performs a real browser navigation and writes only a
# screenshot artifact; the token-bearing URL is deliberately not echoed.
if ! npx --yes playwright screenshot --browser=chromium --device="Desktop Chrome" "$bootstrap" "$artifact_dir/nexus-home.png" >/dev/null 2>"$artifact_dir/browser.stderr"; then
  if grep -q "download new browsers\|Executable doesn't exist" "$artifact_dir/browser.stderr"; then
    echo "NEXUS_BROWSER_SMOKE_SKIP: Playwright browser executable is not installed" >&2
    exit 2
  fi
  echo "NEXUS_BROWSER_SMOKE_FAIL: Playwright navigation failed" >&2
  exit 1
fi

if [[ ! -s "$artifact_dir/nexus-home.png" ]]; then
  echo "NEXUS_BROWSER_SMOKE_FAIL: screenshot artifact was not created" >&2
  exit 1
fi
echo "NEXUS_BROWSER_SMOKE_PASS: local bootstrap rendered in Chromium"
