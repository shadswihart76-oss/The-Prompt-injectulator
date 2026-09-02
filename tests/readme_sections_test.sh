#!/usr/bin/env bash
set -euo pipefail

README="README.md"

[ -f "$README" ]

required_sections=(
  "## Overview"
  "## Features"
  "## Installation"
  "## Usage"
  "## Examples"
  "## Configuration"
  "## Limitations"
  "## Responsible Use and Safety"
  "## Contributing"
  "## License"
)

for section in "${required_sections[@]}"; do
  grep -Fq "$section" "$README"
done

grep -Fq "assets/banner.png" "$README"

echo "README section smoke test passed."
