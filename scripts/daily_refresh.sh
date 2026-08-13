#!/usr/bin/env bash
# daily_refresh.sh — 一键日刷数据管道
# 串联新脚本：basic_info → fetch_daily → fetch_money_flow → compute_indicators → refresh_mv
#
# 用法:
#   ./daily_refresh.sh                 # 默认：跳过 basic_info (首次用 --initial)
#   ./daily_refresh.sh --initial       # 首次安装：含 basic_info 全量刷新
#   ./daily_refresh.sh --skip-fetch    # 只重算指标 + 刷 MV (数据已就位时)
#   ./daily_refresh.sh --skip-compute  # 跳过指标计算 (一般不需要)
#
# 依赖: scripts/config.ini 已配置 DATABASE_URL

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOGS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)/logs"
mkdir -p "$LOGS_DIR"

# 找 python (优先 .venv)
if [[ -x "$SCRIPT_DIR/.venv/bin/python" ]]; then
    PY="$SCRIPT_DIR/.venv/bin/python"
elif command -v python3 >/dev/null 2>&1; then
    PY="$(command -v python3)"
else
    echo "❌ 找不到 python3" >&2; exit 1
fi

# 解析参数
INITIAL=0
SKIP_FETCH=0
SKIP_COMPUTE=0
SKIP_MV=0
for arg in "$@"; do
    case "$arg" in
        --initial)     INITIAL=1 ;;
        --skip-fetch)  SKIP_FETCH=1 ;;
        --skip-compute) SKIP_COMPUTE=1 ;;
        --skip-mv)     SKIP_MV=1 ;;
        -h|--help)
            sed -n '2,12p' "$0"; exit 0 ;;
        *) echo "未知参数: $arg" >&2; exit 2 ;;
    esac
done

export PYTHONIOENCODING=utf-8
export CONFIG_INI="$SCRIPT_DIR/config.ini"
cd "$SCRIPT_DIR"

stamp() { date +"%Y-%m-%d %H:%M:%S"; }
TS="$(date +%Y%m%d_%H%M%S)"
LOG="$LOGS_DIR/daily_refresh_${TS}.log"
exec > >(tee -a "$LOG") 2>&1

echo "=========================================="
echo "[$(stamp)] oh-my-stock 日刷开始"
echo "[$(stamp)] 工作目录: $SCRIPT_DIR"
echo "[$(stamp)] Python:    $PY"
echo "[$(stamp)] 日志:      $LOG"
echo "[$(stamp)] 参数:      initial=$INITIAL skip_fetch=$SKIP_FETCH skip_compute=$SKIP_COMPUTE skip_mv=$SKIP_MV"
echo "=========================================="

T0=$(date +%s)

run_step() {
    local name="$1"; shift
    local script="$1"; shift
    if [[ ! -f "$script" ]]; then
        echo "⚠️  脚本不存在: $script"; return 0
    fi
    echo ""
    echo "▶▶▶ [$(stamp)] $name — $script"
    local s=$(date +%s)
    if "$PY" "$script" "$@"; then
        local d=$(( $(date +%s) - s ))
        echo "✅ [$(stamp)] $name 完成 (${d}s)"
    else
        local rc=$?
        echo "❌ [$(stamp)] $name 失败 (exit=$rc)，中止管道"
        exit $rc
    fi
}

# 1) basic_info（仅 --initial）
if [[ $INITIAL -eq 1 ]]; then
    run_step "拉取基础信息" get_basic_info_lite.py
else
    echo "⏭  跳过基础信息 (用 --initial 才会跑)"
fi

# 2) 日线 K 线
if [[ $SKIP_FETCH -eq 0 ]]; then
    run_step "拉取 K 线" fetch_daily.py
    run_step "拉取资金流" fetch_money_flow.py
else
    echo "⏭  跳过数据拉取 (--skip-fetch)"
fi

# 3) 计算指标
if [[ $SKIP_COMPUTE -eq 0 ]]; then
    run_step "计算技术指标" compute_indicators.py
else
    echo "⏭  跳过指标计算 (--skip-compute)"
fi

# 4) 刷新物化视图
if [[ $SKIP_MV -eq 0 ]]; then
    run_step "刷新 stock_history_mv" refresh_mv.py
else
    echo "⏭  跳过物化视图 (--skip-mv)"
fi

T1=$(date +%s)
echo ""
echo "=========================================="
echo "[$(stamp)] 🎉 日刷完成，总耗时 $((T1-T0)) 秒"
echo "=========================================="
