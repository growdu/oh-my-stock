import os
import pandas as pd
import akshare as ak
import psycopg2
from sqlalchemy import create_engine, text
import configparser
from datetime import datetime

# === 读取配置 ===
config = configparser.ConfigParser()
config.read("config.ini", encoding="utf-8")

DB_URI = config["database"]["url"]
engine = create_engine(DB_URI)
conn = engine.connect()

# === 获取已有的财务数据（避免重复） ===
def get_existing_financial_keys():
    query = "SELECT symbol, report_date, report_type FROM stock_financial_data"
    df = pd.read_sql(query, engine)
    df["key"] = df["symbol"] + "_" + df["report_date"].astype(str) + "_" + df["report_type"]
    return set(df["key"].tolist())

# === 获取所有股票代码 ===
def get_all_symbols():
    df = pd.read_sql("SELECT symbol FROM stock_basic_info", engine)
    return df["symbol"].tolist()

def fetch_and_store_financial(symbol, existing_keys):
    try:
        df = ak.stock_financial_abstract(symbol=symbol)
    except Exception as e:
        print(f"获取 {symbol} 财务数据失败: {e}")
        return

    if df.empty:
        print(f"{symbol} 无财务数据")
        return

    # Step 1: 宽表转成长表
    df_long = df.melt(id_vars=["选项", "指标"], var_name="report_date", value_name="value")

    # Step 2: 列映射（这里用指标中文映射到数据库字段）
    indicator_map = {
        "每股收益": "eps",
        "稀释每股收益": "eps_diluted",
        "营业总收入": "total_revenue",
        "营业利润": "operating_profit",
        "净利润": "net_profit",
        "总资产": "total_assets",
        "总负债": "total_liabilities",
        "股东权益": "equity",
        "净资产收益率": "roe",
        "毛利率": "gross_margin",
        "经营活动现金流量净额": "operating_cash_flow",
        "投资活动现金流量净额": "investing_cash_flow",
        "筹资活动现金流量净额": "financing_cash_flow"
    }

    # Step 3: 把指标中文映射成数据库字段
    df_long["db_field"] = df_long["指标"].map(indicator_map)

    # Step 4: pivot成长表结构 => 每个 report_date 一行
    df_pivot = df_long.pivot_table(
        index="report_date",
        columns="db_field",
        values="value",
        aggfunc="first"
    ).reset_index()

    # Step 5: 生成 report_type
    df_pivot["report_type"] = df_pivot["report_date"].apply(lambda x: "Q" + x[4:6] if pd.notnull(x) else None)

    # Step 6: 日期格式化
    df_pivot["report_date"] = pd.to_datetime(df_pivot["report_date"], format="%Y%m%d")
    current_year = datetime.now().year
    df_pivot["report_year"] = pd.to_datetime(df_pivot["report_date"], errors="coerce").dt.year
    df_pivot = df_pivot[df_pivot["report_year"] >= current_year - 2]
    # Step 7: 插入数据
    insert_data = []
    for _, row in df_pivot.iterrows():
        key = f"{symbol}_{row['report_date'].date()}_{row['report_type']}"
        if key in existing_keys:
            continue

        insert_data.append({
            "symbol": symbol,
            "report_date": row["report_date"].date(),
            "report_type": row["report_type"],
            "eps": row.get("eps"),
            "eps_diluted": row.get("eps_diluted"),
            "total_revenue": row.get("total_revenue"),
            "operating_profit": row.get("operating_profit"),
            "net_profit": row.get("net_profit"),
            "total_assets": row.get("total_assets"),
            "total_liabilities": row.get("total_liabilities"),
            "equity": row.get("equity"),
            "roe": row.get("roe"),
            "gross_margin": row.get("gross_margin"),
            "operating_cash_flow": row.get("operating_cash_flow"),
            "investing_cash_flow": row.get("investing_cash_flow"),
            "financing_cash_flow": row.get("financing_cash_flow")
        })

    if insert_data:
        pd.DataFrame(insert_data).to_sql("stock_financial_data", engine, if_exists="append", index=False)
        print(f"{symbol} 插入 {len(insert_data)} 条财务数据")
    else:
        print(f"{symbol} 无需更新")

# === 主流程 ===
def main():
    existing_keys = get_existing_financial_keys()
    symbols = get_all_symbols()

    for symbol in symbols:
        fetch_and_store_financial(symbol, existing_keys)

if __name__ == "__main__":
    main()
