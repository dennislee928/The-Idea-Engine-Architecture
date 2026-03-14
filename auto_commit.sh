#!/bin/bash

echo "Starting auto-commit every 1 minute. Press [CTRL+C] to stop."

while true; do
  sleep 60
  git add .
  
  # Only commit if there are changes to avoid empty commit errors
  if ! git diff-index --quiet HEAD --; then
    git commit -m "Auto-commit: $(date +'%Y-%m-%d %H:%M:%S')"
    echo "Committed changes at $(date +'%H:%M:%S')"
  fi
done