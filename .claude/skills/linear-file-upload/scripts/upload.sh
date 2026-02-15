#!/bin/bash
set -euo pipefail

if [ $# -eq 0 ]; then
  echo "Usage: upload.sh <filepath>" >&2
  exit 1
fi

if [ ! -f "$1" ]; then
  echo "Error: file not found: $1" >&2
  exit 1
fi

curl -sf -F "file=@$1" https://0x0.st
