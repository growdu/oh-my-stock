import datetime
import os
import subprocess
import time
from pathlib import Path
import schedule

# ===== 配置区 =====
IMMEDIATE_RUN = 1  # 设为 1 立即执行一次，0 仅定时执行
SCRIPTS = [
    r"F:\pythonwork\oh-my-stock\scripts\get_stock_daily.py",
    r"F:\pythonwork\oh-my-stock\scripts\get_money_flow.py",
]
# =================

def get_venv_python():
    """获取虚拟环境的 Python 路径（适用于 uv/venv）"""
    venv_python = Path(".venv/Scripts/python.exe" if os.name == "nt" else ".venv/bin/python")
    if not venv_python.exists():
        raise FileNotFoundError(f"虚拟环境未找到: {venv_python}")
    return str(venv_python)

def log_with_timestamp(message):
    """带时间戳的日志记录"""
    timestamp = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    print(f"[{timestamp}] {message}")

def run_script(script_path):
    """执行脚本并实时记录日志（带时间戳）"""
    venv_python = get_venv_python()
    timestamp = datetime.datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
    log_dir = Path("logs")
    log_dir.mkdir(exist_ok=True)
    log_file = log_dir / f"{timestamp}_{Path(script_path).stem}.log"

    try:
        # 创建子进程环境变量，强制使用UTF-8编码
        env = os.environ.copy()
        env["PYTHONIOENCODING"] = "utf-8"
        
        log_with_timestamp(f"开始执行脚本: {script_path}")
        
        # 实时写入日志的解决方案
        with open(log_file, "w", encoding="utf-8", buffering=1) as f:  # 行缓冲模式
            process = subprocess.Popen(
                [venv_python, script_path],
                stdout=f,
                stderr=subprocess.STDOUT,
                text=True,
                encoding="utf-8",
                env=env,
                bufsize=1  # 行缓冲
            )
        
        log_with_timestamp(f"脚本已启动 -> 日志文件: {log_file}")
        return process
    except Exception as e:
        log_with_timestamp(f"执行失败: {script_path} ({e})")
        return None

if __name__ == "__main__":
    processes = []
    
    # 立即执行（如果 IMMEDIATE_RUN = 1）
    if IMMEDIATE_RUN:
        log_with_timestamp("=== 开始立即并行执行脚本 ===")
        for script in SCRIPTS:
            p = run_script(script)
            if p:
                processes.append(p)

    # 定时调度（每天 16:00）
    log_with_timestamp("=== 定时任务启动（每天 16:00，并行执行）===")
    for script in SCRIPTS:
        schedule.every().day.at("16:00").do(run_script, script)

    # 主循环
    while True:
        schedule.run_pending()
        
        # 检查正在运行的进程
        for p in processes[:]:
            if p.poll() is not None:  # 进程已结束
                log_with_timestamp(f"脚本执行完成，退出码: {p.returncode}")
                processes.remove(p)
        
        time.sleep(60)