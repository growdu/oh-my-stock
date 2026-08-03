"""
timer.py
========
定时调度：
  - 16:00  拉增量日线   (get_stock_daily.py)
  - 16:05  更新资金流    (get_money_flow_v2.py)
  - 16:30  增量拉财报    (get_financial_info.py)
  - 17:00  计算技术指标  (compute_indicators.py)
  - 17:05  刷新物化视图  (refresh_mv.py)

支持 Windows / Linux / macOS，自动寻找 .venv/bin/python
"""
import datetime
import os
import subprocess
import sys
from pathlib import Path
import schedule
import time

SCRIPT_DIR = Path(__file__).resolve().parent
ROOT_DIR = SCRIPT_DIR.parent
LOGS_DIR = ROOT_DIR / "logs"
LOGS_DIR.mkdir(exist_ok=True)

IMMEDIATE_RUN = int(os.environ.get("IMMEDIATE_RUN", "0"))

# (脚本相对路径, 每日执行时刻)
SCHEDULE = [
    (SCRIPT_DIR / "get_stock_daily.py",      "16:00"),
    (SCRIPT_DIR / "get_money_flow_v2.py",    "16:05"),
    (SCRIPT_DIR / "get_financial_info.py",   "16:30"),
    (SCRIPT_DIR / "compute_indicators.py",   "17:00"),
    (SCRIPT_DIR / "refresh_mv.py",           "17:05"),
]


def python_bin() -> str:
    """locate venv python, fall back to current interpreter."""
    if os.name == "nt":
        cand = SCRIPT_DIR / ".venv" / "Scripts" / "python.exe"
    else:
        cand = SCRIPT_DIR / ".venv" / "bin" / "python"
    if cand.exists():
        return str(cand)
    return sys.executable


def log(msg: str):
    ts = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    print(f"[{ts}] {msg}", flush=True)


def run(script: Path):
    if not script.exists():
        log(f"⚠️  脚本不存在: {script}")
        return None
    stamp = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")
    log_file = LOGS_DIR / f"{stamp}_{script.stem}.log"
    log(f"▶  {script.name}  -> {log_file.name}")
    env = os.environ.copy()
    env["PYTHONIOENCODING"] = "utf-8"
    env["CONFIG_INI"] = str(SCRIPT_DIR / "config.ini")
    with open(log_file, "w", encoding="utf-8", buffering=1) as f:
        proc = subprocess.Popen(
            [python_bin(), str(script)],
            stdout=f, stderr=subprocess.STDOUT,
            text=True, encoding="utf-8",
            env=env, bufsize=1,
        )
    return proc


def job(script: Path, when: str):
    run(script)


def main():
    log("=== oh-my-stock 调度器启动 ===")
    log(f"Python: {python_bin()}")
    log(f"脚本数: {len(SCHEDULE)}")
    for sc, when in SCHEDULE:
        schedule.every().day.at(when).do(job, sc, when)
        log(f"  ⏰ {when}  {sc.name}")

    if IMMEDIATE_RUN:
        log("IMMEDIATE_RUN=1，立即并行执行全部脚本")
        procs = [run(sc) for sc, _ in SCHEDULE]

    try:
        while True:
            schedule.run_pending()
            time.sleep(30)
    except KeyboardInterrupt:
        log("收到 Ctrl+C，退出")


if __name__ == "__main__":
    main()
