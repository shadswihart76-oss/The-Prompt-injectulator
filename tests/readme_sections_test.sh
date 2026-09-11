#!/usr/bin/env bash
set -euo pipefail

README="README.md"

[ -f "$README" ]

required_sections=(
  "## Quick Start"
  "## Architecture"
  "## API Reference"
  "## Environment Variables"
  "## Running Tests"
  "## Threat Model & Security Notes"
  "## License"
)

for section in "${required_sections[@]}"; do
  grep -Fq "$section" "$README"
done

echo "README structure smoke test passed."
