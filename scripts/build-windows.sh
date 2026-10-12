#!/usr/bin/env bash
# Cross-build bin/b-his-host.exe (Windows amd64) with the embedded login UI.
# Demo account: admin / Admin@123. Memory database unless config says otherwise.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root/web"
npm ci
npm run build:demo
rm -rf "$root/internal/webui/assets"
mkdir -p "$root/internal/webui/assets"
cp -a "$root/web/dist/." "$root/internal/webui/assets/"
cd "$root"
mkdir -p "$root/bin"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o "$root/bin/b-his-host.exe" ./cmd/host
echo "Built $root/bin/b-his-host.exe"
echo "On Windows, double-click it. Login: http://127.0.0.1:8080/login"
echo "Demo account: admin / Admin@123"
