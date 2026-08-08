#!/bin/sh
set -e

for f in /etc/secrets/*/*; do
  [ -f "$f" ] && . "$f"
done

exec /app/ffmpeg-service
