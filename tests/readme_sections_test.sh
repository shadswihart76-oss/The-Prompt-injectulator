#!/usr/bin/env bash
set -euo pipefail

README="README.md"

[ -f "$README" ]

required_sections=(
  "## Purpose"
  "## Features"
  "## Setup"
  "## Usage"
  "## Examples"
  "## Limitations"
  "## Maintenance Notes"
)

for section in "${required_sections[@]}"; do
  grep -Fq "$section" "$README"
done

echo "README structure smoke test passed."
