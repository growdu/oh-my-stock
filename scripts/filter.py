import akshare as ak
import pandas as pd
import logging
from time import sleep
from datetime import datetime, timedelta
import os

# ========== 日志设置 ==========
os.makedirs("log", exist_ok=True)
log_file = f"log/up_filter_{datetime.now().strftime('%Y%m%d')}.log"

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
    handlers=[
        logging.FileHandler(log_file, encoding='utf-8'),
        logging.StreamHandler()
    ]
)

# ========== 缓存工具函数 ==========
def is_cache_valid(file_path, max_age_days=7):
    if not os.path.exists(file_path):
        return False
    file_time = datetime.fromtimestamp(os.path.getmtime(file_path))
    return datetime.now() - file_time < timedelta(days=max_age_days)

# ========== 获取创业板股票 ==========
os.makedirs("cache", exist_ok=True)
cy_stocks_file = "cache/cy_stocks.csv"

try:
    if is_cache_valid(cy_stocks_file):
        cy_stocks = pd.read_csv(cy_stocks_file, dtype=str)
        logging.info(f"[缓存] 加载创业板股票列表，共 {len(cy_stocks)} 个")
    else:
        cy_stocks = ak.stock_cy_a_spot_em()
        cy_stocks.to_csv(cy_stocks_file, index=False, encoding="utf-8-sig")
        logging.info(f"[网络] 获取创业板股票列表，共 {len(cy_stocks)} 个")
except Exception as e:
    logging.error(f"获取股票列表失败: {e}")
    exit(1)

# ========== 筛选最近3天连续上涨的股票 ==========
result = []
start_date = (datetime.now() - timedelta(days=7)).strftime("%Y%m%d")
logging.info(f"开始时间为: {start_date}")

os.makedirs("cache/history", exist_ok=True)

for idx, row in cy_stocks.iterrows():
    code = row["代码"]
    name = row["名称"]
    history_file = f"cache/history/{code}.csv"

    try:
        if is_cache_valid(history_file):
            df = pd.read_csv(history_file, dtype=str, parse_dates=["日期"])
        else:
            df = ak.stock_zh_a_hist(symbol=code, period="daily", adjust="qfq", start_date=start_date, timeout=10)
            df.to_csv(history_file, index=False, encoding="utf-8-sig")

        if df.shape[0] < 5:
            continue

        df["收盘"] = pd.to_numeric(df["收盘"], errors="coerce")
        df = df.dropna(subset=["收盘"])
        df = df.tail(3)
        close_diff = df["收盘"].diff().dropna()

        if all(close_diff > 0):
            result.append({"代码": code, "名称": name})
            logging.info(f"{code} {name} 连续上涨3天")
        sleep(0.5)

    except Exception as e:
        logging.warning(f"{code} 处理失败: {e}")
        continue

# ========== 输出结果 ==========
df_result = pd.DataFrame(result)
df_result.to_csv("up_stocks_gem.csv", index=False, encoding="utf-8-sig")
logging.info("筛选完成，已保存至 up_stocks_gem.csv")
