"""
timer.py
========
定时调度（v2）：
  - 16:00  daily_refresh.sh     (拉 K 线 → 资金流 → 算指标 → 刷 MV)
  - 17:00  refresh_mv.py        (兜底再刷一次，防止 daily 流程异常时数据不一致)

daily_refresh.sh 内部串联：
  fetch_daily → fetch_money_flow → compute_indicators → refresh_mv

支持 Windows / Linux / macOS，自动寻找 .venv/bin/python 或 bash
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

# (启动命令, 每日执行时刻, 显示名)
SCHEDULE = [
    (["bash", str(SCRIPT_DIR / "daily_refresh.sh")],            "16:00", "daily_refresh"),
    ([sys.executable, str(SCRIPT_DIR / "refresh_mv.py")],        "17:00", "refresh_mv_fallback"),
]


def bash_bin() -> str:
    if os.name == "nt":
        cand = SCRIPT_DIR / ".venv" / "Scripts" / "bash.exe"
        if cand.exists(): return str(cand)
    for n in ("bash", "/bin/bash", "/usr/bin/bash"):
        if Path(n).exists(): return n
    return "bash"


def python_bin() -> str:
    """locate venv python, fall back to current interpreter."""
    if os.name == "nt":
        cand = SCRIPT_DIR / ".venv" / "Scripts" / "python.exe"
    else:
        cand = SCRIPT_DIR / ".venv" / "bin" / "python"
    if cand.exists():
        return str(cand)
    return sys.executable


def resolve_cmd(cmd):
    """把 sys.executable / bash 占位符替换成真实路径"""
    out = []
    for x in cmd:
        if x == "bash":
            out.append(bash_bin())
        elif x == sys.executable or x == "python":
            out.append(python_bin())
        else:
            out.append(x)
    return out


def log(msg: str):
    ts = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    print(f"[{ts}] {msg}", flush=True)


def run(cmd):
    if isinstance(cmd, list) and cmd[0] == "bash":
        cmd = ["bash", str(SCRIPT_DIR / cmd[1].name)] if len(cmd) == 2 else cmd
    resolved = resolve_cmd(cmd)
    script_name = Path(resolved[-1]).stem
    stamp = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")
    log_file = LOGS_DIR / f"{stamp}_{script_name}.log"
    log(f"▶  {' '.join(resolved)}  -> {log_file.name}")
    env = os.environ.copy()
    env["PYTHONIOENCODING"] = "utf-8"
    env["CONFIG_INI"] = str(SCRIPT_DIR / "config.ini")
    with open(log_file, "w", encoding="utf-8", buffering=1) as f:
        proc = subprocess.Popen(
            resolved,
            stdout=f, stderr=subprocess.STDOUT,
            text=True, encoding="utf-8",
            env=env, bufsize=1,
        )
    return proc


def job(cmd, when):
    run(cmd)


def main():
    log("=== oh-my-stock 调度器启动 ===")
    log(f"Python: {python_bin()}")
    log(f"Bash:   {bash_bin()}")
    log(f"任务数: {len(SCHEDULE)}")
    for cmd, when, name in SCHEDULE:
        schedule.every().day.at(when).do(job, cmd, when)
        log(f"  ⏰ {when}  {name}")

    if IMMEDIATE_RUN:
        log("IMMEDIATE_RUN=1，立即并行执行全部任务")
        procs = [run(cmd) for cmd, _, _ in SCHEDULE]

    try:
        while True:
            schedule.run_pending()
            time.sleep(30)
    except KeyboardInterrupt:
        log("收到 Ctrl+C，退出")


if __name__ == "__main__":
    main()
