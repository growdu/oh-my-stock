"""
fetch_quote.py — 用腾讯 qt.gtimg.cn 实时行情批量补 stock_daily_data 字段
（turnover_rate / pe_ttm / pb）和写资金流估算到 stock_money_flow_all。

字段位置（0-based, ~-split）：
  28  turnover_rate 换手率%
  29  pe_ttm        市盈率(动)
  36  pb            市净率
  6   volume        成交量(手)
  7   out_amount    外盘(手)
  8   in_amount     内盘(手)
"""
import configparser, sys, time
import requests
import pandas as pd
from sqlalchemy import create_engine, text

cfg = configparser.ConfigParser()
cfg.read("config.ini", encoding="utf-8")
DB = cfg.get("database", "url")

S = requests.Session()
S.trust_env = False
S.headers["User-Agent"] = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

def fetch_batch(prefixed: list) -> dict:
    """一次拉多只股票：https://qt.gtimg.cn/q=sh600519,sz000001,..."""
    if not prefixed:
        return {}
    url = "https://qt.gtimg.cn/q=" + ",".join(prefixed)
    try:
        r = S.get(url, timeout=10)
        out = {}
        for line in r.text.split(";"):
            line = line.strip()
            if "=" not in line: continue
            k, v = line.split("=", 1)
            sym = k.strip().lstrip("v_")
            parts = v.strip().strip('"').split("~")
            if len(parts) < 40: continue
            out[sym] = parts
        return out
    except Exception:
        return {}

def parse_quote(parts: list) -> dict:
    """从 40+ 字段解析需要的列"""
    def f(i):
        try: return float(parts[i])
        except: return None
    return {
        "price":      f(3),
        "last_close": f(4),
        "open":       f(5),
        "volume_hand":f(6),       # 成交量 手
        "out_hand":   f(7),       # 外盘 手
        "in_hand":    f(8),       # 内盘 手
        "change_amt": f(31),
        "change_pct": f(32),
        "high":       f(33),
        "low":        f(34),
        "turnover_w": f(37),      # 成交额 万
        "turnover":   f(38),      # 换手率 %
        "pe_ttm":     f(39),      # 市盈率(动)
        "amplitude":  f(43),
        "pb":         f(46),      # 市净率
        "cap_circ":   f(44),      # 流通市值 亿
        "cap_total":  f(45),      # 总市值 亿
    }

def main():
    eng = create_engine(DB)
    syms = pd.read_sql("SELECT symbol, market FROM stock_basic_info", eng)
    print(f"[start] {len(syms)} stocks")
    latest = pd.read_sql("SELECT MAX(trade_date) AS d FROM stock_daily_data", eng).iloc[0, 0]
    print(f"[info] latest trade_date = {latest}")

    # build prefixed list
    def prefix(s, m):
        m = str(m)
        if "科创板" in m or s.startswith("688"): return "sh"
        if "创业板" in m or s.startswith(("300","301")): return "sz"
        if "北交所" in m or s.startswith(("8","43","92","83","87","88")): return "bj"
        if "沪" in m or s.startswith(("60","90","11","13","5")): return "sh"
        return "sz"
    syms["prefixed"] = [f"{prefix(r.symbol, r.market)}{r.symbol}" for r in syms.itertuples()]

    BATCH = 60  # 一次拉 60 只
    updates_daily = []   # (turnover, pe, pb, sym) for latest day
    updates_mf = []      # (sym, ts, net_amount) for money flow
    fail = 0
    total = len(syms)
    for i in range(0, total, BATCH):
        batch = syms.iloc[i:i+BATCH]
        prefixed = batch["prefixed"].tolist()
        result = fetch_batch(prefixed)
        for _, row in batch.iterrows():
            sym = row["symbol"]; pre = row["prefixed"]
            key = pre
            parts = result.get(key)
            if not parts or len(parts) < 40:
                fail += 1
                continue
            q = parse_quote(parts)
            if q["turnover"] is not None and q["pe_ttm"] is not None and q["pb"] is not None:
                updates_daily.append((q["turnover"], q["pe_ttm"], q["pb"], sym, latest))
            # 资金流估算：外盘-内盘 (手) × 100 × 均价
            if q["out_hand"] is not None and q["in_hand"] is not None and q["price"] is not None:
                net = (q["out_hand"] - q["in_hand"]) * 100 * q["price"]
                inflow = q["out_hand"] * 100 * q["price"]
                outflow = q["in_hand"] * 100 * q["price"]
                updates_mf.append((sym, 0, q["price"], q["change_pct"], q["turnover"],
                                   inflow, outflow, net, q["turnover_w"]*10000, latest))
        if (i // BATCH) % 20 == 0:
            print(f"  [{i+BATCH}/{total}] updates_daily={len(updates_daily)} mf={len(updates_mf)} fail={fail}", flush=True)
        # flush every 500
        if len(updates_daily) >= 500:
            _flush_daily(eng, updates_daily); updates_daily = []
        if len(updates_mf) >= 500:
            _flush_mf(eng, updates_mf); updates_mf = []
    if updates_daily: _flush_daily(eng, updates_daily)
    if updates_mf: _flush_mf(eng, updates_mf)
    print(f"[done] daily_updates=processed, mf=processed, fail={fail}")

def _flush_daily(eng, rows):
    sql = text("""
        UPDATE stock_daily_data SET
            turnover_rate = v.turnover,
            pe_ttm        = v.pe,
            pb            = v.pb
        FROM (SELECT unnest(:syms)::text AS s,
                     unnest(:dates)::date AS d,
                     unnest(:tos)::numeric AS turnover,
                     unnest(:pes)::numeric AS pe,
                     unnest(:pbs)::numeric AS pb) v
        WHERE stock_daily_data.symbol = v.s AND stock_daily_data.trade_date = v.d
    """)
    with eng.begin() as conn:
        conn.execute(sql, {
            "syms": [r[3] for r in rows],
            "dates": [r[4] for r in rows],
            "tos":  [r[0] for r in rows],
            "pes":  [r[1] for r in rows],
            "pbs":  [r[2] for r in rows],
        })

def _flush_mf(eng, rows):
    sql = text("""
        INSERT INTO stock_money_flow_all
            (time_span, symbol, latest_price, change_percent, turnover_rate,
             inflow_amount, outflow_amount, net_amount, turnover, trade_date)
        VALUES (:ts, :sym, :price, :chg, :to, :in_, :out_, :net, :turn, :dt)
        ON CONFLICT (symbol, trade_date, time_span) DO UPDATE SET
            latest_price=EXCLUDED.latest_price, change_percent=EXCLUDED.change_percent,
            turnover_rate=EXCLUDED.turnover_rate,
            inflow_amount=EXCLUDED.inflow_amount, outflow_amount=EXCLUDED.outflow_amount,
            net_amount=EXCLUDED.net_amount, turnover=EXCLUDED.turnover
    """)
    with eng.begin() as conn:
        conn.execute(sql, [
            {"ts":r[1],"sym":r[0],"price":r[2],"chg":r[3],"to":r[4],
             "in_":r[5],"out_":r[6],"net":r[7],"turn":r[8],"dt":r[9]}
            for r in rows
        ])

if __name__ == "__main__":
    main()
