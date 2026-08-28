"""
compute_indicators.py (v5) — 使用 SQLAlchemy 2.x Connection + raw psycopg2 connection
"""
import configparser
from datetime import datetime
import pandas as pd
import numpy as np
from sqlalchemy import create_engine, text
from psycopg2.extras import execute_values
from tqdm import tqdm

def calc_one(df: pd.DataFrame) -> pd.DataFrame:
    df = df.sort_values("trade_date").reset_index(drop=True).copy()
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
        # RSI 标准定义：loss=0（连续 n 日全涨）→ RSI=100；gain=0（全跌）→ RSI=0
        # 之前用 loss.replace(0, np.nan) 会让 RSI 变 NaN，再被 json 序列化失败
        with np.errstate(divide='ignore', invalid='ignore'):
            rsi = 100 - 100 / (1 + gain / loss)
        rsi = rsi.where(loss != 0, 100)   # loss=0 时强制 100
        rsi = rsi.where(~((loss == 0) & (gain == 0)), 50)  # 极罕见：n 日全平 → 50
        df[f"rsi{n}"] = rsi
    std20 = df["close"].rolling(20).std()
    df["boll_mid"]   = df["ma20"]
    df["boll_upper"] = df["ma20"] + 2 * std20
    df["boll_lower"] = df["ma20"] - 2 * std20
    # 预计算 lag 列（避免规则查询时跑窗口函数）
    for n in (1, 2, 3):
        df[f"ma5_lag{n}"]  = df["ma5"].shift(n)
        df[f"ma10_lag{n}"] = df["ma10"].shift(n)
        df[f"ma20_lag{n}"] = df["ma20"].shift(n)
    df["ma60_lag1"] = df["ma60"].shift(1)
    df["k_lag1"]    = df["k"].shift(1)
    df["d_lag1"]    = df["d"].shift(1)
    df["dif_lag1"]  = df["dif"].shift(1)
    df["dea_lag1"]  = df["dea"].shift(1)
    df["yang_lag0"] = (df["close"] > df["open"])
    df["yang_lag1"] = df["yang_lag0"].shift(1)
    df["yang_lag2"] = df["yang_lag0"].shift(2)
    df["yang_lag3"] = df["yang_lag0"].shift(3)
    df["close_lag1"] = df["close"].shift(1)
    df["close_lag2"] = df["close"].shift(2)
    df["close_lag3"] = df["close"].shift(3)
    df["close_lag5"] = df["close"].shift(5)
    df["vol_lag1"]   = df["volume"].shift(1)
    df["vol_lag2"]   = df["volume"].shift(2)
    df["vol_lag3"]   = df["volume"].shift(3)
    df["vol_lag5"]   = df["volume"].shift(5)
    # 滚动统计（不含当天的窗口）
    df["vol_avg5"]   = df["volume"].shift(1).rolling(5).mean()
    df["low_min5"]   = df["low"].shift(1).rolling(5).min()
    df["low_min20"]  = df["low"].shift(1).rolling(20).min()
    df["low_min60"]  = df["low"].shift(1).rolling(60).min()
    df["high_max5"]  = df["high"].shift(1).rolling(5).max()
    df["high_max30"] = df["high"].shift(1).rolling(30).max()
    df["high_max60"] = df["high"].shift(1).rolling(60).max()
    df["high_max90"] = df["high"].shift(1).rolling(90).max()

    keep = ["trade_date", "ma5","ma10","ma20","ma60",
            "macd","dif","dea","k","d","j",
            "rsi6","rsi12","rsi24",
            "boll_upper","boll_mid","boll_lower",
            "ma5_lag1","ma5_lag2","ma5_lag3",
            "ma10_lag1","ma10_lag2","ma10_lag3",
            "ma20_lag1","ma20_lag2","ma20_lag3",
            "ma60_lag1","k_lag1","d_lag1","dif_lag1","dea_lag1",
            "yang_lag0","yang_lag1","yang_lag2","yang_lag3",
            "close_lag1","close_lag2","close_lag3","close_lag5",
            "vol_lag1","vol_lag2","vol_lag3","vol_lag5",
            "vol_avg5","low_min5","low_min20","low_min60",
            "high_max5","high_max30","high_max60","high_max90"]
    return df[keep].dropna(subset=["ma5","ma10","ma20","ma60"])

COLS = ["symbol","calc_date","ma5","ma10","ma20","ma60",
        "macd","dif","dea","k","d","j",
        "rsi6","rsi12","rsi24","boll_upper","boll_mid","boll_lower",
        "ma5_lag1","ma5_lag2","ma5_lag3",
        "ma10_lag1","ma10_lag2","ma10_lag3",
        "ma20_lag1","ma20_lag2","ma20_lag3",
        "ma60_lag1","k_lag1","d_lag1","dif_lag1","dea_lag1",
        "yang_lag0","yang_lag1","yang_lag2","yang_lag3",
        "close_lag1","close_lag2","close_lag3","close_lag5",
        "vol_lag1","vol_lag2","vol_lag3","vol_lag5",
        "vol_avg5","low_min5","low_min20","low_min60",
        "high_max5","high_max30","high_max60","high_max90"]

def main():
    cfg = configparser.ConfigParser()
    cfg.read("config.ini", encoding="utf-8")
    db_url = cfg.get("database", "url")
    engine = create_engine(db_url, connect_args={"options": "-c jit=off"})

    print("→ 清空 stock_indicators")
    with engine.begin() as conn:
        conn.execute(text("TRUNCATE stock_indicators"))

    print("→ 读取 stock_basic_info")
    syms = pd.read_sql("SELECT symbol FROM stock_basic_info", engine)["symbol"].tolist()
    print(f"  共 {len(syms)} 只")

    print("→ 读取 stock_daily_data")
    daily = pd.read_sql(text("SELECT symbol, trade_date, close, open, high, low, volume "
                             "FROM stock_daily_data ORDER BY symbol, trade_date"),
                        engine, parse_dates=["trade_date"])

    skipped = 0
    BATCH = 5000
    buffer = []
    total_rows = 0
    for sym in tqdm(syms, desc="计算指标"):
        sub = daily[daily["symbol"] == sym]
        if sub.empty or len(sub) < 60:
            skipped += 1
            continue
        ind = calc_one(sub.copy())
        if ind.empty:
            skipped += 1
            continue
        ind.insert(0, "symbol", sym)
        ind = ind.rename(columns={"trade_date": "calc_date"})
        rows = [tuple(r) for r in ind[COLS].itertuples(index=False, name=None)]
        buffer.extend(rows)
        total_rows += len(rows)
        if len(buffer) >= BATCH:
            _flush(engine, buffer); buffer = []
    if buffer:
        _flush(engine, buffer)

    print(f"✅ 完成: 写入 {total_rows} 条指标；跳过 {skipped} 只")

def _flush(engine, rows):
    col_names = ",".join(COLS)
    update_cols = ",".join([f"{c}=EXCLUDED.{c}" for c in COLS if c not in ("symbol","calc_date")])
    sql = f"INSERT INTO stock_indicators ({col_names}) VALUES %s ON CONFLICT (symbol, calc_date) DO UPDATE SET {update_cols}"
    with engine.begin() as conn:
        raw = conn.connection.driver_connection
        cur = raw.cursor()
        execute_values(cur, sql, rows, page_size=1000)
        cur.close()

if __name__ == "__main__":
    main()
