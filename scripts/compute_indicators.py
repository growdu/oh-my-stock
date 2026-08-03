"""
compute_indicators.py
=====================
读取 stock_daily_data，按 symbol 计算 MA/MACD/KDJ/RSI/BOLL，写入 stock_indicators。

技术实现要点：
- MA(N)        : 收盘价 N 日简单移动平均
- MACD (12,26,9): DIF=EMA12-EMA26；DEA=EMA9(DIF)；MACD=2*(DIF-DEA)
- KDJ (9,3,3)  : RSV=(C-Low9)/(High9-Low9)*100；K=SMA(RSV,3)；D=SMA(K,3)；J=3K-2D
- RSI (N)      : N 日相对强弱指标
- BOLL (20,2)  : MID=MA20；UPPER=MID+2*STD；LOWER=MID-2*STD

运行: python compute_indicators.py
依赖: pandas, sqlalchemy, psycopg2-binary, tqdm
"""
import configparser
from datetime import datetime
import pandas as pd
import numpy as np
from sqlalchemy import create_engine, text
from tqdm import tqdm
import os
import sys


def main():
    cfg_path = os.environ.get("CONFIG_INI", "config.ini")
    config = configparser.ConfigParser()
    config.read(cfg_path, encoding="utf-8")
    db_url = config.get("database", "url")
    if db_url.startswith("postgresql://user:password"):
        print("⚠️  请先在 config.ini 中配置真实的数据库 url", file=sys.stderr)
        sys.exit(1)

    engine = create_engine(db_url)

    print("→ 读取 stock_basic_info ...")
    symbols = pd.read_sql("SELECT symbol FROM stock_basic_info", engine)["symbol"].tolist()
    print(f"  共 {len(symbols)} 只股票")

    inserted = 0
    skipped = 0

    with engine.begin() as conn:
        for symbol in tqdm(symbols, desc="计算指标"):
            df = pd.read_sql(
                text("SELECT trade_date, close, high, low, volume FROM stock_daily_data "
                     "WHERE symbol = :s ORDER BY trade_date ASC"),
                conn, params={"s": symbol},
                parse_dates=["trade_date"],
            )
            if df.empty or len(df) < 60:
                skipped += 1
                continue

            df["ma5"]  = df["close"].rolling(5).mean()
            df["ma10"] = df["close"].rolling(10).mean()
            df["ma20"] = df["close"].rolling(20).mean()
            df["ma60"] = df["close"].rolling(60).mean()

            ema12 = df["close"].ewm(span=12, adjust=False).mean()
            ema26 = df["close"].ewm(span=26, adjust=False).mean()
            df["dif"] = ema12 - ema26
            df["dea"] = df["dif"].ewm(span=9, adjust=False).mean()
            df["macd"] = (df["dif"] - df["dea"]) * 2

            low9 = df["low"].rolling(9).min()
            high9 = df["high"].rolling(9).max()
            rsv = (df["close"] - low9) / (high9 - low9).replace(0, np.nan) * 100
            rsv = rsv.fillna(50)
            k_vals = [50.0]
            for v in rsv.iloc[1:]:
                k_vals.append(k_vals[-1] * 2 / 3 + v / 3)
            df["k"] = k_vals
            d_vals = [50.0]
            for v in df["k"].iloc[1:]:
                d_vals.append(d_vals[-1] * 2 / 3 + v / 3)
            df["d"] = d_vals
            df["j"] = 3 * df["k"] - 2 * df["d"]

            for n in (6, 12, 24):
                diff = df["close"].diff()
                gain = diff.clip(lower=0).rolling(n).mean()
                loss = (-diff.clip(upper=0)).rolling(n).mean()
                rs = gain / loss.replace(0, np.nan)
                df[f"rsi{n}"] = 100 - 100 / (1 + rs)

            std20 = df["close"].rolling(20).std()
            df["boll_mid"]   = df["ma20"]
            df["boll_upper"] = df["ma20"] + 2 * std20
            df["boll_lower"] = df["ma20"] - 2 * std20

            keep = ["trade_date",
                    "ma5","ma10","ma20","ma60",
                    "macd","dif","dea",
                    "k","d","j",
                    "rsi6","rsi12","rsi24",
                    "boll_upper","boll_mid","boll_lower"]
            df = df[keep].dropna()

            if df.empty:
                skipped += 1
                continue

            # UPSERT
            for _, row in df.iterrows():
                conn.execute(text("""
                    INSERT INTO stock_indicators
                        (symbol, calc_date, ma5, ma10, ma20, ma60, macd, dif, dea,
                         k, d, j, rsi6, rsi12, rsi24, boll_upper, boll_mid, boll_lower)
                    VALUES (:s,:d,:ma5,:ma10,:ma20,:ma60,:macd,:dif,:dea,
                            :k,:d,:j,:r6,:r12,:r24,:bu,:bm,:bl)
                    ON CONFLICT (symbol, calc_date) DO UPDATE SET
                        ma5=EXCLUDED.ma5, ma10=EXCLUDED.ma10, ma20=EXCLUDED.ma20, ma60=EXCLUDED.ma60,
                        macd=EXCLUDED.macd, dif=EXCLUDED.dif, dea=EXCLUDED.dea,
                        k=EXCLUDED.k, d=EXCLUDED.d, j=EXCLUDED.j,
                        rsi6=EXCLUDED.rsi6, rsi12=EXCLUDED.rsi12, rsi24=EXCLUDED.rsi24,
                        boll_upper=EXCLUDED.boll_upper, boll_mid=EXCLUDED.boll_mid, boll_lower=EXCLUDED.boll_lower
                """), {
                    "s": symbol,
                    "d": row["trade_date"].date(),
                    "ma5": float(row["ma5"]),
                    "ma10": float(row["ma10"]),
                    "ma20": float(row["ma20"]),
                    "ma60": float(row["ma60"]),
                    "macd": float(row["macd"]),
                    "dif": float(row["dif"]),
                    "dea": float(row["dea"]),
                    "k": float(row["k"]),
                    "d": float(row["d"]),
                    "j": float(row["j"]),
                    "r6": float(row["rsi6"]),
                    "r12": float(row["rsi12"]),
                    "r24": float(row["rsi24"]),
                    "bu": float(row["boll_upper"]),
                    "bm": float(row["boll_mid"]),
                    "bl": float(row["boll_lower"]),
                })
            inserted += len(df)

    print(f"✅ 完成: 写入 {inserted} 条指标；跳过 {skipped} 只股票（日线不足 60 条）")


if __name__ == "__main__":
    main()
