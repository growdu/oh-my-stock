#!/usr/bin/env bash
# 重新构建前端并让 nginx 重新加载（dist 已配到 nginx 18080 server root）
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FRONT_DIR="$ROOT/front"

echo "==> build front"
cd "$FRONT_DIR"
npm run build

echo "==> reload nginx (18080)"
sudo nginx -t
sudo nginx -s reload

echo "==> done. verify:"
curl -sI "http://127.0.0.1:18080/" | head -3
