#!/usr/bin/env bash
# restart_timer.sh — 一键重启已修复的调度器
# 用法:  bash scripts/restart_timer.sh
#
# 行为：
#   1. 杀掉所有旧的 scripts/timer.py
#   2. 用 nohup 后台启动新的
#   3. 输出重定向到 logs/timer.out.log / logs/timer.err.log
#
# 配合 IMMEDIATE_RUN 可手动补一次，详见 README 与 timer.py 头部注释

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
LOGS_DIR="$ROOT_DIR/logs"
mkdir -p "$LOGS_DIR"

echo "▶ 杀掉旧的 timer.py (如有) ..."
pkill -TERM -f "$SCRIPT_DIR/timer.py" 2>/dev/null || true
sleep 2
pkill -KILL -f "$SCRIPT_DIR/timer.py" 2>/dev/null || true

echo "▶ 启动新的 timer.py ..."
cd "$ROOT_DIR"
nohup python3 "$SCRIPT_DIR/timer.py"     > "$LOGS_DIR/timer.out.log" 2> "$LOGS_DIR/timer.err.log" &
disown
NEW=$!
echo "  PID=$NEW"

sleep 2
if kill -0 "$NEW" 2>/dev/null; then
  echo "✅ 调度器已启动，PID=$NEW"
  echo "   stdout: $LOGS_DIR/timer.out.log"
  echo "   stderr: $LOGS_DIR/timer.err.log"
else
  echo "❌ 启动失败，看 $LOGS_DIR/timer.err.log"
  tail -20 "$LOGS_DIR/timer.err.log" || true
  exit 1
fi

echo
echo "【可选】手动补一次今天的全量刷新（追上 14 号之后缺的 10 天数据）："
echo "    bash $SCRIPT_DIR/daily_refresh.sh"
echo
echo "【可选】IMMEDIATE_RUN 立即跑一次调度器的两个任务："
echo "    IMMEDIATE_RUN=1 python3 $SCRIPT_DIR/timer.py"
