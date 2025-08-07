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

# ========== 获取创业板股票 ==========
try:
    all_stocks_df = ak.stock_info_a_code_name()
    stock_list_df = all_stocks_df[all_stocks_df["code"].str.startswith("300")]
    stock_codes = stock_list_df["code"].tolist()
    logging.info(f"共获取创业板股票数量: {len(stock_codes)}")
except Exception as e:
    logging.error(f"获取股票列表失败: {e}")
    exit(1)

result = []
count = 0
# 当前时间减 7 天，并格式化为 20250501 的形式
start_date = (datetime.now() - timedelta(days=7)).strftime("%Y%m%d")
print("开始时间为：", start_date)

# ========== 遍历股票 ==========
for code in stock_codes:
    # count += 1
    # if count > 2:
    #     break
    try:
        df = ak.stock_zh_a_hist(symbol=code, period="daily", 
        adjust="qfq",start_date=start_date, timeout=10)
        if df.shape[0] < 5:
            continue
        #print(df.columns)
        df = df.tail(3)
        close_diff = df["收盘"].diff().dropna()
        #close_diff = df["收盘价"].diff().dropna()
        if all(close_diff > 0):
            result.append(code)
            logging.info(f"{code} 连续上涨5天")
        sleep(0.5)
    except Exception as e:
        logging.warning(f"{code} 处理失败: {e}")
        continue

# ========== 输出结果 ==========
df_result = pd.DataFrame(result, columns=["连续5日上涨的创业板股票代码"])
df_result.to_csv("up_stocks_gem.csv", index=False, encoding="utf-8-sig")
logging.info("筛选完成，已保存至 up_stocks_gem.csv")
