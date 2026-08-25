#!/usr/bin/env bash
# install.sh — 一键把 oh-my-stock 的 systemd timer / service 装到 /etc/systemd/system
#
# 重要：如果当前用户 ~/.config/systemd/user/ 下存在老的
# oh-my-stock-timer.service (Type=simple + Restart=always，封装了
# scripts/timer.py)，它会跟我们今天装的两个系统级 timer 在 17:00
# 同时触发，造成 fetch_daily / compute_indicators 双跑。
# 该脚本自动检测 + 清理。

set -euo pipefail
SUDO="${SUDO:-sudo}"
DIR="$(cd "$(dirname "$0")" && pwd)"

echo "▶ 步骤 0：清理 legacy 用户级 unit (oh-my-stock-timer.service)"
LEGACY="$HOME/.config/systemd/user/oh-my-stock-timer.service"
LEGACY_WANTS="$HOME/.config/systemd/user/default.target.wants/oh-my-stock-timer.service"
if [ -f "$LEGACY" ] || [ -f "$LEGACY_WANTS" ]; then
  if command -v systemctl >/dev/null && [ -n "${XDG_RUNTIME_DIR:-}" ]; then
    XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}" \
      systemctl --user disable oh-my-stock-timer.service 2>/dev/null || true
    XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}" \
      systemctl --user stop  oh-my-stock-timer.service 2>/dev/null || true
  fi
  # 兜底：直接删文件（disable 失败也能清）
  rm -f "$LEGACY" "$LEGACY_WANTS"
  XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}" \
    systemctl --user daemon-reload 2>/dev/null || true
  # 同时把还活着的 timer.py 进程也杀掉（auto-restart 已被禁用）
  pkill -TERM -f "scripts/timer.py" 2>/dev/null || true
  echo "  ✓ legacy unit 已停用 + 文件已删"
else
  echo "  . 没有 legacy，无需清理"
fi

echo
echo "▶ 步骤 1: cp $DIR/*.{service,timer} /etc/systemd/system/"
$SUDO cp "$DIR"/*.service "$DIR"/*.timer /etc/systemd/system/

echo "▶ 步骤 2: daemon-reload"
$SUDO systemctl daemon-reload

echo "▶ 步骤 3: enable --now (启动并开机自启)"
$SUDO systemctl enable --now oh-my-stock-daily-refresh.timer
$SUDO systemctl enable --now oh-my-stock-mv-fallback.timer

echo
echo "▶ 下次触发时刻:"
$SUDO systemctl list-timers | grep -E "oh-my-stock" || true

echo
echo "✅ 安装完成。今后由 /etc/systemd/system 下的两个 .timer 接管 17:00 / 18:00"
echo "   查看:  sudo systemctl list-timers oh-my-stock-*"
echo "   日志:  sudo journalctl -u oh-my-stock-daily-refresh.service -e"
echo "   立即跑: sudo systemctl start oh-my-stock-daily-refresh.service"
echo "   取消 legacy 自启: 见 install.sh 步骤 0"
