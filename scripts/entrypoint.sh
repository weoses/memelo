#!/bin/sh
set -e

for f in /etc/secrets/*/*; do
  [ -f "$f" ] && . "$f"
done

echo "env vars: $(env | cut -d= -f1 | sort | tr '\n' ' ')"

exec "$@"
