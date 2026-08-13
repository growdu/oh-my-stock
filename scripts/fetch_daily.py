"""
新浪 K 线 fetcher：拉 5000+ 只 A 股近 60 个交易日 K 线。
沪深京前缀 (sh/sz/bj) + 6 位代码。
"""
import os, sys, time
import requests
import configparser
import pandas as pd
from datetime import datetime
from concurrent.futures import ThreadPoolExecutor, as_completed
from sqlalchemy import create_engine, text

cfg = configparser.ConfigParser()
cfg.read("config.ini", encoding="utf-8")
DB = cfg.get("database", "url")

URL = "https://money.finance.sina.com.cn/quotes_service/api/json_v2.php/CN_MarketData.getKLineData"
HEADERS = {
    "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
                  "(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Referer": "https://finance.sina.com.cn/",
}
SESS = requests.Session()
SESS.headers.update(HEADERS)
SESS.trust_env = False

def prefix_of(symbol: str) -> str:
    if symbol.startswith(("60", "68", "90", "11", "13", "5")):
        return "sh"
    if symbol.startswith(("43", "83", "87", "88")):
        return "bj"
    return "sz"

def fetch_one(symbol: str, datalen: int = 300) -> list:
    s = f"{prefix_of(symbol)}{symbol}"
    for attempt in range(3):
        try:
            r = SESS.get(URL, params={"symbol": s, "scale": 240, "datalen": datalen}, timeout=8)
            txt = r.text.strip()
            if r.status_code != 200 or not txt.startswith("["):
                time.sleep(0.3)
                continue
            import json
            data = json.loads(txt)
            rows = []
            prev_close = None
            for k in data:
                close = float(k["close"])
                rows.append({
                    "symbol": symbol,
                    "trade_date": datetime.strptime(k["day"], "%Y-%m-%d").date(),
                    "open": float(k["open"]),
                    "high": float(k["high"]),
                    "low":  float(k["low"]),
                    "close": close,
                    "volume": int(k["volume"]),
                    "turnover": float(k["close"]) * int(k["volume"]),  # 估算
                    "change_percent": (close - prev_close) / prev_close * 100 if prev_close else 0,
                    "change_amount": close - prev_close if prev_close else 0,
                    "amplitude": (float(k["high"]) - float(k["low"])) / prev_close * 100 if prev_close else 0,
                    "turnover_rate": None,  # 需要 outstanding_shares 后算
                })
                prev_close = close
            return rows
        except Exception as e:
            time.sleep(0.3)
    return []

def _upsert_method(table, conn, keys, data_iter):
    from sqlalchemy.dialects.postgresql import insert
    rows = list(data_iter)
    insert_stmt = insert(table.table).values(rows)
    update_cols = {c.name: insert_stmt.excluded[c.name] for c in table.table.columns
                   if c.name not in ("id",)}
    upsert = insert_stmt.on_conflict_do_update(
        index_elements=["symbol", "trade_date"],
        set_=update_cols)
    conn.execute(upsert)

def main():
    eng = create_engine(DB, pool_size=10, max_overflow=20)
    syms = pd.read_sql("SELECT symbol FROM stock_basic_info ORDER BY symbol", eng)["symbol"].tolist()
    print(f"[start] {len(syms)} stocks", flush=True)

    total = 0
    fail = 0
    BATCH = 200
    batch = []
    with ThreadPoolExecutor(max_workers=10) as exe:
        futures = {exe.submit(fetch_one, s, 300): s for s in syms}
        for i, fut in enumerate(as_completed(futures), 1):
            rows = fut.result()
            if not rows:
                fail += 1
            else:
                batch.extend(rows)
            if len(batch) >= BATCH:
                df = pd.DataFrame(batch)
                df.to_sql("stock_daily_data", eng, if_exists="append", index=False, method=_upsert_method)
                total += len(batch); batch = []
            if i % 200 == 0 or i == len(syms):
                print(f"  [{i}/{len(syms)}] wrote={total} fail={fail}", flush=True)
        if batch:
            df = pd.DataFrame(batch)
            df.to_sql("stock_daily_data", eng, if_exists="append", index=False, method=_upsert_method)
            total += len(batch)
    print(f"[done] wrote {total} rows, fail {fail} symbols", flush=True)

if __name__ == "__main__":
    main()
