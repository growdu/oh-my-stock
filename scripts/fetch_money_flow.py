"""
fetch_money_flow.py — 用东财 push2his.eastmoney.com 拉日级资金流历史（5 天），
写入 stock_money_flow_all（time_span=0）+ 更新 stock_daily_data 的 turnover_rate。
"""
import configparser
import requests
import pandas as pd
from datetime import datetime, timedelta
from sqlalchemy import create_engine, text
from concurrent.futures import ThreadPoolExecutor, as_completed

cfg = configparser.ConfigParser()
cfg.read("config.ini", encoding="utf-8")
DB = cfg.get("database", "url")

S = requests.Session()
S.trust_env = False
S.headers["User-Agent"] = "Mozilla/5.0"
S.headers["Referer"] = "https://quote.eastmoney.com/"

URL = "https://push2his.eastmoney.com/api/qt/stock/fflow/daykline/get"

def secid_of(sym: str) -> str:
    if sym.startswith(("60","68","90","11","13","5")): return f"1.{sym}"
    return f"0.{sym}"

def fetch_one(sym: str, lmt: int = 5) -> list:
    for _ in range(3):
        try:
            r = S.get(URL, params={
                "fields1": "f1,f2,f3,f4",
                "fields2": "f51,f52,f53,f54,f55,f56,f57,f58",
                "klt": 101, "lmt": lmt,
                "secid": secid_of(sym),
            }, timeout=8)
            d = r.json()
            kls = (d.get("data") or {}).get("klines") or []
            rows = []
            for line in kls:
                p = line.split(",")
                if len(p) < 8: continue
                rows.append({
                    "symbol": sym,
                    "trade_date": datetime.strptime(p[0], "%Y-%m-%d").date(),
                    "main_net":   float(p[1]),
                    "retail_net": float(p[2]),  # 小单
                    "mid_net":    float(p[3]),  # 中单
                    "big_net":    float(p[4]),  # 大单
                    "super_net":  float(p[5]),  # 特大单
                    "main_pct":   float(p[6]),
                    "turnover":   float(p[7]),
                })
            return rows
        except Exception:
            continue
    return []

def main():
    eng = create_engine(DB)
    syms = pd.read_sql("SELECT symbol FROM stock_basic_info", eng)["symbol"].tolist()
    print(f"[start] {len(syms)} stocks, 5 days")
    rows = []
    fail = 0
    with ThreadPoolExecutor(max_workers=3) as exe:
        futs = {exe.submit(fetch_one, s, 5): s for s in syms}
        for i, fut in enumerate(as_completed(futs), 1):
            r = fut.result()
            if not r: fail += 1
            else: rows.extend(r)
            if i % 500 == 0:
                print(f"  [{i}/{len(syms)}] rows={len(rows)} fail={fail}", flush=True)
    print(f"[flush] {len(rows)} rows")
    _flush(eng, rows)
    print(f"[done] fail={fail}")

def _flush(eng, rows):
    if not rows: return
    sql = text("""
        INSERT INTO stock_money_flow_all
            (time_span, symbol, net_amount, turnover_rate, turnover, trade_date)
        VALUES (0, :sym, :net, :to, :to_amt, :dt)
        ON CONFLICT (symbol, trade_date, time_span) DO UPDATE SET
            net_amount=EXCLUDED.net_amount,
            turnover_rate=EXCLUDED.turnover_rate,
            turnover=EXCLUDED.turnover
    """)
    # net_amount = main_net; turnover = 成交额 (从 daily 拿)
    # 先拿每只每天的成交额
    with eng.begin() as conn:
        days = sorted({r["trade_date"] for r in rows})
        placeholders = ",".join([f"'{d}'" for d in days])
        tdf = pd.read_sql(f"SELECT symbol, trade_date, turnover FROM stock_daily_data WHERE trade_date IN ({placeholders})", eng)
        tmap = {(r.symbol, r.trade_date): float(r.turnover or 0) for r in tdf.itertuples()}
        params = []
        for r in rows:
            params.append({
                "sym": r["symbol"],
                "net": r["main_net"],
                "to":  r["turnover"],
                "to_amt": tmap.get((r["symbol"], r["trade_date"]), 0),
                "dt":  r["trade_date"],
            })
        # 分批
        for i in range(0, len(params), 1000):
            conn.execute(sql, params[i:i+1000])

if __name__ == "__main__":
    main()
