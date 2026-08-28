#!/usr/bin/env bash
# =============================================================
# 进阶精选 Top 3 预计算
# 由 daily_refresh.sh 在刷完 MV 后调用；也可手动跑
# 调用后端 POST /api/v1/screen/final-pick，结果落库到 final_picks 表
# =============================================================
set -euo pipefail

API_BASE="${API_BASE:-http://127.0.0.1:3003}"
TOP_N="${TOP_N:-3}"
# 激进+双创优先：8 条短线/突破/技术反弹预设
PRESETS='["ma-trend","volume-price","ma-golden-cross","tech-bounce","limit-up-strong","high-position-breakout","boll-bounce","breakout-5d"]'

echo "→ 进阶精选预计算（top_n=${TOP_N}）"
resp=$(curl -sS -X POST "${API_BASE}/api/v1/screen/final-pick" \
    -H 'Content-Type: application/json' \
    -d "{\"preset_ids\":${PRESETS},\"top_n\":${TOP_N}}" \
    --max-time 60)
trade_date=$(echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('trade_date',''))" 2>/dev/null || echo "")
picks=$(echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d.get('picks',[])))" 2>/dev/null || echo "0")

if [[ -z "$trade_date" ]]; then
    echo "✗ 预计算失败：未返回 trade_date"
    echo "$resp" | head -c 500
    exit 1
fi

echo "✓ trade_date=${trade_date}  落库 ${picks} 只"
echo "$resp" | python3 -c "
import sys,json
d=json.load(sys.stdin)
for i,p in enumerate(d.get('picks',[]),1):
    print(f'  #{i} {p[\"name\"]}({p[\"symbol\"]}) score={p[\"score\"]}')
"
