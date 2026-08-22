#!/bin/sh
set -e

for f in /etc/secrets/*/*; do
  [ -e "$f" ] || continue

  echo "Found secret file: $f"

  ls -la "$f"

  [ -f "$f" ] && export $(grep -v '^#' "$f" | xargs)
done

echo "env vars: $(env | cut -d= -f1 | sort | tr '\n' ' ')"

exec "$@"
