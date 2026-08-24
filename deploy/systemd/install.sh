#!/usr/bin/env bash
# install.sh — 一键把 oh-my-stock 的 systemd timer / service 装上
set -euo pipefail
SUDO="${SUDO:-sudo}"
DIR="$(cd "$(dirname "$0")" && pwd)"

echo "▶ cp $DIR/*.service /etc/systemd/system/"
$SUDO cp "$DIR"/*.service "$DIR"/*.timer /etc/systemd/system/

echo "▶ daemon-reload"
$SUDO systemctl daemon-reload

echo "▶ enable --now (启动并开机自启)"
$SUDO systemctl enable --now oh-my-stock-daily-refresh.timer
$SUDO systemctl enable --now oh-my-stock-mv-fallback.timer

echo "▶ list 相关 timer"
systemctl list-timers | grep -E "oh-my-stock|next" || true
echo
echo "✅ 安装完成。今后由 systemd 接管 17:00 / 18:00 调度"
echo "   查看:  systemctl list-timers oh-my-stock-*"
echo "   日志:  sudo journalctl -u oh-my-stock-daily-refresh.service -e"
