#!/bin/bash
set -e

export PATH=$PATH:/home/ubuntu/go/bin

echo "=== Building frontend ==="
cd frontend
pnpm install
NODE_OPTIONS='--max-old-space-size=4096' pnpm build
cd ..

echo "=== Building backend ==="
CGO_ENABLED=0 go build -ldflags "-X main.versionString=v1.0.1+1 -X main.buildString=$(date +%Y-%m-%dT%H:%M:%S)" -o libredesk ./cmd/

echo "=== Stuffing assets ==="
stuffbin -a stuff -in libredesk -out libredesk frontend/dist i18n schema.sql static

echo "=== Rebuilding and restarting containers ==="
docker compose build --no-cache
docker compose down
docker compose up -d

echo "=== Done ==="
